package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service reúne as regras de identidade, autenticação e sessões.
type Service struct {
	store           *Store
	idleTimeout     time.Duration
	absoluteTimeout time.Duration
	now             func() time.Time
	dummyHash       string
}

// NewService cria o serviço de identidade.
func NewService(db *pgxpool.Pool, idleTimeout, absoluteTimeout time.Duration) *Service {
	// Hash descartável usado para equalizar o tempo de resposta quando o
	// e-mail informado não existe.
	dummyHash, err := HashPassword("cultivar-dummy-password-for-timing")
	if err != nil {
		dummyHash = ""
	}

	return &Service{
		store:           NewStore(db),
		idleTimeout:     idleTimeout,
		absoluteTimeout: absoluteTimeout,
		now:             time.Now,
		dummyHash:       dummyHash,
	}
}

// Register valida os dados e cria uma nova conta local.
func (s *Service) Register(ctx context.Context, input RegisterInput) (*Account, error) {
	input.Email = NormalizeEmail(input.Email)
	input.Handle = NormalizeHandle(input.Handle)
	input.DisplayName = strings.TrimSpace(input.DisplayName)

	if err := ValidateEmail(input.Email); err != nil {
		return nil, err
	}
	if err := ValidateHandle(input.Handle); err != nil {
		return nil, err
	}
	if err := ValidateDisplayName(input.DisplayName); err != nil {
		return nil, err
	}
	if err := ValidatePassword(input.Password); err != nil {
		return nil, err
	}

	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	return s.store.CreateAccount(ctx, input, passwordHash)
}

// Authenticate verifica as credenciais e retorna a conta correspondente.
func (s *Service) Authenticate(ctx context.Context, email, password string) (*Account, error) {
	credentials, err := s.store.FindCredentialsByEmail(ctx, NormalizeEmail(email))
	if errors.Is(err, ErrNotFound) {
		// Mantém um custo de verificação semelhante ao de uma conta existente.
		VerifyPassword(s.dummyHash, password)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if !VerifyPassword(credentials.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	if credentials.Status != "active" {
		return nil, ErrAccountDisabled
	}

	return &credentials.Account, nil
}

// StartSession cria uma sessão e retorna o token opaco que deve ser enviado
// ao navegador. Apenas o hash do token é persistido.
func (s *Service) StartSession(ctx context.Context, accountID, ipAddress, userAgent string) (string, error) {
	token, err := newSessionToken()
	if err != nil {
		return "", err
	}

	now := s.now()
	_, err = s.store.CreateSession(
		ctx,
		accountID,
		hashSessionToken(token),
		now.Add(s.idleTimeout),
		now.Add(s.absoluteTimeout),
		ipAddress,
		userAgent,
	)
	if err != nil {
		return "", err
	}

	return token, nil
}

// AuthenticateSession valida o token e retorna a conta da sessão.
func (s *Service) AuthenticateSession(ctx context.Context, token string) (*Account, error) {
	if token == "" {
		return nil, ErrSessionExpired
	}

	account, sessionID, err := s.store.FindSessionAccount(ctx, hashSessionToken(token))
	if err != nil {
		return nil, err
	}

	// A sessão continua válida mesmo que a atualização do último acesso falhe.
	_ = s.store.TouchSession(ctx, sessionID, s.now().Add(s.idleTimeout))

	return account, nil
}

// EndSession revoga a sessão correspondente ao token.
func (s *Service) EndSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.store.RevokeSession(ctx, hashSessionToken(token))
}

// FindPersonByHandle busca o perfil público de uma pessoa.
func (s *Service) FindPersonByHandle(ctx context.Context, handle string) (*Person, error) {
	return s.store.FindPersonByHandle(ctx, NormalizeHandle(handle))
}

func newSessionToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashSessionToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
