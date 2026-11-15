// Package identity reúne o contexto de identidade e participação.
//
// Contas, pessoas, perfis, sessões e permissões são implementados aqui,
// conforme os ADRs 0002, 0003 e 0008.
package identity

import (
	"errors"
	"time"
)

// Erros de domínio do contexto de identidade.
var (
	ErrInvalidCredentials = errors.New("credenciais inválidas")
	ErrAccountDisabled    = errors.New("conta indisponível")
	ErrEmailTaken         = errors.New("e-mail já cadastrado")
	ErrHandleTaken        = errors.New("identificador já em uso")
	ErrInvalidEmail       = errors.New("e-mail inválido")
	ErrInvalidHandle      = errors.New("identificador inválido")
	ErrInvalidDisplayName = errors.New("nome de exibição inválido")
	ErrWeakPassword       = errors.New("senha fraca")
	ErrNotFound           = errors.New("registro não encontrado")
	ErrSessionExpired     = errors.New("sessão expirada")
)

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
	EmailVerified bool
	CreatedAt     time.Time
	Person        Person
}

// RegisterInput reúne os dados necessários para criar uma conta.
type RegisterInput struct {
	Email       string
	Password    string
	Handle      string
	DisplayName string
}

// Session identifica uma sessão ativa de uma conta.
type Session struct {
	ID                string
	AccountID         string
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
}
