package identity

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store implementa o acesso ao banco para o contexto de identidade.
type Store struct {
	db *pgxpool.Pool
}

// NewStore cria um repositório de identidade.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// Credentials reúne a conta e o hash de senha usado na autenticação.
type Credentials struct {
	Account
	PasswordHash string
}

// CreateAccount cria a pessoa e a conta em uma única transação.
func (s *Store) CreateAccount(ctx context.Context, input RegisterInput, passwordHash string) (*Account, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var person Person
	err = tx.QueryRow(ctx, `
		INSERT INTO people (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text, handle, display_name, bio, created_at
	`, input.Handle, input.DisplayName).Scan(
		&person.ID,
		&person.Handle,
		&person.DisplayName,
		&person.Bio,
		&person.CreatedAt,
	)
	if isUniqueViolation(err, "people_handle_key") {
		return nil, ErrHandleTaken
	}
	if err != nil {
		return nil, err
	}

	var account Account
	var verifiedAt *time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO accounts (person_id, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id::text, email, status, email_verified_at, created_at
	`, person.ID, input.Email, passwordHash).Scan(
		&account.ID,
		&account.Email,
		&account.Status,
		&verifiedAt,
		&account.CreatedAt,
	)
	if isUniqueViolation(err, "accounts_email_key") {
		return nil, ErrEmailTaken
	}
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	account.EmailVerified = verifiedAt != nil
	account.Person = person

	return &account, nil
}

// FindCredentialsByEmail busca a conta e o hash de senha pelo e-mail.
func (s *Store) FindCredentialsByEmail(ctx context.Context, email string) (*Credentials, error) {
	var credentials Credentials
	var verifiedAt *time.Time

	err := s.db.QueryRow(ctx, `
		SELECT a.id::text, a.email, a.status, a.email_verified_at, a.created_at, a.password_hash,
		       p.id::text, p.handle, p.display_name, p.bio, p.created_at
		FROM accounts a
		JOIN people p ON p.id = a.person_id
		WHERE lower(a.email) = lower($1)
	`, email).Scan(
		&credentials.ID,
		&credentials.Email,
		&credentials.Status,
		&verifiedAt,
		&credentials.CreatedAt,
		&credentials.PasswordHash,
		&credentials.Person.ID,
		&credentials.Person.Handle,
		&credentials.Person.DisplayName,
		&credentials.Person.Bio,
		&credentials.Person.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	credentials.EmailVerified = verifiedAt != nil

	return &credentials, nil
}

// FindPersonByHandle busca o perfil público de uma pessoa.
func (s *Store) FindPersonByHandle(ctx context.Context, handle string) (*Person, error) {
	var person Person

	err := s.db.QueryRow(ctx, `
		SELECT id::text, handle, display_name, bio, created_at
		FROM people
		WHERE lower(handle) = lower($1)
	`, handle).Scan(
		&person.ID,
		&person.Handle,
		&person.DisplayName,
		&person.Bio,
		&person.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &person, nil
}

// FindPersonByID busca o perfil público pelo identificador interno.
func (s *Store) FindPersonByID(ctx context.Context, id string) (*Person, error) {
	var person Person

	err := s.db.QueryRow(ctx, `
		SELECT id::text, handle, display_name, bio, created_at
		FROM people
		WHERE id::text = $1
	`, id).Scan(
		&person.ID,
		&person.Handle,
		&person.DisplayName,
		&person.Bio,
		&person.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &person, nil
}

// UpdatePersonProfile atualiza o nome de exibição e a biografia.
func (s *Store) UpdatePersonProfile(ctx context.Context, personID, displayName, bio string) (*Person, error) {
	var person Person

	err := s.db.QueryRow(ctx, `
		UPDATE people
		SET display_name = $2,
		    bio = $3,
		    updated_at = now()
		WHERE id::text = $1
		RETURNING id::text, handle, display_name, bio, created_at
	`, personID, displayName, bio).Scan(
		&person.ID,
		&person.Handle,
		&person.DisplayName,
		&person.Bio,
		&person.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &person, nil
}

// AccountStats retorna contagens de contas usadas pelo NodeInfo.
func (s *Store) AccountStats(ctx context.Context) (Stats, error) {
	var stats Stats

	err := s.db.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE a.status = 'active'),
			count(*) FILTER (WHERE a.status = 'active' AND EXISTS (
				SELECT 1 FROM sessions s
				WHERE s.account_id = a.id AND s.last_seen_at > now() - interval '30 days'
			)),
			count(*) FILTER (WHERE a.status = 'active' AND EXISTS (
				SELECT 1 FROM sessions s
				WHERE s.account_id = a.id AND s.last_seen_at > now() - interval '180 days'
			))
		FROM accounts a
	`).Scan(&stats.TotalAccounts, &stats.ActiveMonth, &stats.ActiveHalfyear)
	if err != nil {
		return Stats{}, err
	}

	return stats, nil
}

// CreateSession registra uma nova sessão com o hash do token.
func (s *Store) CreateSession(ctx context.Context, accountID string, tokenHash []byte, idleExpiresAt, absoluteExpiresAt time.Time, ipAddress, userAgent string) (*Session, error) {
	var session Session

	err := s.db.QueryRow(ctx, `
		INSERT INTO sessions (account_id, token_hash, idle_expires_at, absolute_expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, account_id::text, idle_expires_at, absolute_expires_at
	`, accountID, tokenHash, idleExpiresAt, absoluteExpiresAt, ipAddress, userAgent).Scan(
		&session.ID,
		&session.AccountID,
		&session.IdleExpiresAt,
		&session.AbsoluteExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

// FindSessionAccount busca a conta associada a uma sessão válida.
func (s *Store) FindSessionAccount(ctx context.Context, tokenHash []byte) (*Account, string, error) {
	var account Account
	var verifiedAt *time.Time
	var sessionID string

	err := s.db.QueryRow(ctx, `
		SELECT a.id::text, a.email, a.status, a.email_verified_at, a.created_at,
		       p.id::text, p.handle, p.display_name, p.bio, p.created_at,
		       s.id::text
		FROM sessions s
		JOIN accounts a ON a.id = s.account_id
		JOIN people p ON p.id = a.person_id
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.idle_expires_at > now()
		  AND s.absolute_expires_at > now()
		  AND a.status = 'active'
	`, tokenHash).Scan(
		&account.ID,
		&account.Email,
		&account.Status,
		&verifiedAt,
		&account.CreatedAt,
		&account.Person.ID,
		&account.Person.Handle,
		&account.Person.DisplayName,
		&account.Person.Bio,
		&account.Person.CreatedAt,
		&sessionID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrSessionExpired
	}
	if err != nil {
		return nil, "", err
	}

	account.EmailVerified = verifiedAt != nil

	return &account, sessionID, nil
}

// TouchSession atualiza o último acesso e estende o prazo de inatividade.
func (s *Store) TouchSession(ctx context.Context, sessionID string, idleExpiresAt time.Time) error {
	_, err := s.db.Exec(ctx, `
		UPDATE sessions
		SET last_seen_at = now(),
		    idle_expires_at = LEAST($2, absolute_expires_at)
		WHERE id = $1
	`, sessionID, idleExpiresAt)
	return err
}

// RevokeSession revoga a sessão correspondente ao hash do token.
func (s *Store) RevokeSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.db.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	return err
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
