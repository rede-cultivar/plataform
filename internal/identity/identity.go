// Package identity reúne o contexto de identidade e participação.
//
// Contas, pessoas, perfis, sessões e permissões são implementados aqui,
// conforme os ADRs 0002, 0003 e 0008.
package identity

import (
	"errors"
	"time"
)

// Papéis de instância, conforme o ADR 0008.
const (
	RoleAdmin     = "admin"
	RoleModerator = "moderator"
	RoleMember    = "member"
)

// Situações da conta. A situação pending é usada quando o cadastro exige
// aprovação administrativa.
const (
	StatusActive    = "active"
	StatusPending   = "pending"
	StatusSuspended = "suspended"
	StatusClosed    = "closed"
)

// Erros de domínio do contexto de identidade.
var (
	ErrInvalidCredentials   = errors.New("credenciais inválidas")
	ErrAccountDisabled      = errors.New("conta indisponível")
	ErrAccountPending       = errors.New("conta aguardando aprovação")
	ErrInvalidAccountStatus = errors.New("situação de conta inválida")
	ErrInvalidRole          = errors.New("papel de instância inválido")
	ErrEmailTaken           = errors.New("e-mail já cadastrado")
	ErrHandleTaken          = errors.New("identificador já em uso")
	ErrInvalidEmail         = errors.New("e-mail inválido")
	ErrInvalidHandle        = errors.New("identificador inválido")
	ErrInvalidDisplayName   = errors.New("nome de exibição inválido")
	ErrInvalidBio           = errors.New("biografia inválida")
	ErrWeakPassword         = errors.New("senha fraca")
	ErrNotFound             = errors.New("registro não encontrado")
	ErrSessionExpired       = errors.New("sessão expirada")
	ErrCannotFollowSelf     = errors.New("não é possível seguir a própria pessoa")
)

// FollowStats reúne as contagens de seguidores e de pessoas seguidas.
type FollowStats struct {
	Followers int
	Following int
}

// Person representa a identidade individual de um participante.
type Person struct {
	ID          string
	Handle      string
	DisplayName string
	Bio         string
	CreatedAt   time.Time
}

// Account representa as credenciais de acesso associadas a uma pessoa.
type Account struct {
	ID            string
	Email         string
	Status        string
	Role          string
	EmailVerified bool
	CreatedAt     time.Time
	Person        Person
}

// IsAdmin informa se a conta pode administrar a instância.
func (a *Account) IsAdmin() bool {
	return a != nil && a.Role == RoleAdmin
}

// RegisterInput reúne os dados necessários para criar uma conta.
type RegisterInput struct {
	Email       string
	Password    string
	Handle      string
	DisplayName string
	// Status define a situação inicial da conta. Vazio equivale a active.
	// O cadastro com aprovação usa pending.
	Status string
}

// Session identifica uma sessão ativa de uma conta.
type Session struct {
	ID                string
	AccountID         string
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
}

// Stats reúne contagens usadas pelo NodeInfo e por relatórios simples.
type Stats struct {
	TotalAccounts  int
	ActiveMonth    int
	ActiveHalfyear int
}
