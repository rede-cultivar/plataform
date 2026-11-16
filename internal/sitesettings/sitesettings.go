// Package sitesettings reúne as configurações administrativas da instância.
//
// O modelo de cadastro segue o Mastodon: aberto, sujeito a aprovação ou
// fechado, com uma mensagem configurável exibida quando o cadastro está
// fechado. As configurações ficam no banco para que possam ser alteradas pelo
// painel administrativo sem reiniciar a aplicação.
package sitesettings

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// ModeOpen permite que qualquer pessoa crie uma conta.
	ModeOpen = "open"
	// ModeApproved permite o cadastro, mas a conta fica pendente até a
	// aprovação de uma pessoa administradora.
	ModeApproved = "approved"
	// ModeNone bloqueia novos cadastros.
	ModeNone = "none"

	// MaxClosedMessageLength limita a mensagem exibida com o cadastro fechado.
	MaxClosedMessageLength = 500

	// DefaultClosedMessage é usada quando nenhuma mensagem foi configurada.
	DefaultClosedMessage = "Os cadastros estão fechados no momento."
)

// Erros de validação das configurações do site.
var (
	ErrInvalidMode          = errors.New("modo de cadastro inválido")
	ErrClosedMessageTooLong = errors.New("mensagem de cadastros fechados muito longa")
)

// Settings representa as configurações administrativas da instância.
type Settings struct {
	RegistrationsMode          string
	ClosedRegistrationsMessage string
	UpdatedAt                  time.Time
}

// UpdateInput reúne os campos alteráveis pelo painel administrativo.
type UpdateInput struct {
	RegistrationsMode          string
	ClosedRegistrationsMessage string
}

// Default retorna as configurações usadas quando a instância ainda não tem
// uma linha persistida. O padrão preserva o comportamento de cadastro aberto.
func Default() *Settings {
	return &Settings{RegistrationsMode: ModeOpen}
}

// Open informa se o cadastro está disponível, seja aberto ou com aprovação.
func (s *Settings) Open() bool {
	return s == nil || s.RegistrationsMode != ModeNone
}

// RequiresApproval informa se novas contas dependem de aprovação.
func (s *Settings) RequiresApproval() bool {
	return s != nil && s.RegistrationsMode == ModeApproved
}

// ClosedMessage retorna a mensagem efetiva para o período de cadastros
// fechados, aplicando o texto padrão quando nenhum foi configurado.
func (s *Settings) ClosedMessage() string {
	if s == nil {
		return DefaultClosedMessage
	}

	message := strings.TrimSpace(s.ClosedRegistrationsMessage)
	if message == "" {
		return DefaultClosedMessage
	}

	return message
}

// Validate verifica os campos de atualização antes de persistir.
func Validate(input UpdateInput) error {
	switch input.RegistrationsMode {
	case ModeOpen, ModeApproved, ModeNone:
	default:
		return ErrInvalidMode
	}

	if utf8.RuneCountInString(input.ClosedRegistrationsMessage) > MaxClosedMessageLength {
		return ErrClosedMessageTooLong
	}

	return nil
}

// Service lê e atualiza as configurações do site.
type Service struct {
	db *pgxpool.Pool
}

// NewService cria o serviço de configurações do site.
func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// Get retorna as configurações persistidas, aplicando os padrões quando a
// linha ainda não existe.
func (s *Service) Get(ctx context.Context) (*Settings, error) {
	if s == nil || s.db == nil {
		return Default(), nil
	}

	var settings Settings
	err := s.db.QueryRow(ctx, `
		SELECT registrations_mode, closed_registrations_message, updated_at
		FROM site_settings
		WHERE id = TRUE
	`).Scan(
		&settings.RegistrationsMode,
		&settings.ClosedRegistrationsMessage,
		&settings.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Default(), nil
	}
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// Update valida e persiste as configurações do site.
func (s *Service) Update(ctx context.Context, input UpdateInput) (*Settings, error) {
	input.RegistrationsMode = strings.ToLower(strings.TrimSpace(input.RegistrationsMode))
	input.ClosedRegistrationsMessage = strings.TrimSpace(input.ClosedRegistrationsMessage)

	if err := Validate(input); err != nil {
		return nil, err
	}
	if s == nil || s.db == nil {
		return nil, errors.New("serviço de configurações indisponível")
	}

	var settings Settings
	err := s.db.QueryRow(ctx, `
		INSERT INTO site_settings (id, registrations_mode, closed_registrations_message, updated_at)
		VALUES (TRUE, $1, $2, now())
		ON CONFLICT (id) DO UPDATE
		SET registrations_mode = EXCLUDED.registrations_mode,
		    closed_registrations_message = EXCLUDED.closed_registrations_message,
		    updated_at = now()
		RETURNING registrations_mode, closed_registrations_message, updated_at
	`, input.RegistrationsMode, input.ClosedRegistrationsMessage).Scan(
		&settings.RegistrationsMode,
		&settings.ClosedRegistrationsMessage,
		&settings.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}
