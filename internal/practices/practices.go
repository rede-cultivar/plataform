package practices

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Erros de domínio do contexto de conhecimento e práticas.
var (
	ErrNotFound  = errors.New("prática não encontrada")
	ErrForbidden = errors.New("operação não permitida")
)

// ValidationError descreve um campo inválido informado pela pessoa.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}

// Difficulty representa o nível de dificuldade de uma prática.
type Difficulty string

const (
	DifficultyBeginner     Difficulty = "beginner"
	DifficultyIntermediate Difficulty = "intermediate"
	DifficultyAdvanced     Difficulty = "advanced"
)

// Valid informa se a dificuldade é reconhecida.
func (d Difficulty) Valid() bool {
	switch d {
	case DifficultyBeginner, DifficultyIntermediate, DifficultyAdvanced:
		return true
	default:
		return false
	}
}

// Label retorna o rótulo em português usado na interface.
func (d Difficulty) Label() string {
	switch d {
	case DifficultyBeginner:
		return "Iniciante"
	case DifficultyIntermediate:
		return "Intermediário"
	case DifficultyAdvanced:
		return "Avançado"
	default:
		return string(d)
	}
}

// Status representa a situação editorial de uma prática.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

// Valid informa se o status é reconhecido.
func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusPublished, StatusArchived:
		return true
	default:
		return false
	}
}

// Label retorna o rótulo em português usado na interface.
func (s Status) Label() string {
	switch s {
	case StatusDraft:
		return "Rascunho"
	case StatusPublished:
		return "Publicada"
	case StatusArchived:
		return "Arquivada"
	default:
		return string(s)
	}
}

// Visibility representa quem pode visualizar uma prática.
type Visibility string

const (
	VisibilityPrivate Visibility = "private"
	VisibilityPublic  Visibility = "public"
)

// Valid informa se a visibilidade é reconhecida.
func (v Visibility) Valid() bool {
	switch v {
	case VisibilityPrivate, VisibilityPublic:
		return true
	default:
		return false
	}
}

// Label retorna o rótulo em português usado na interface.
func (v Visibility) Label() string {
	switch v {
	case VisibilityPrivate:
		return "Privada"
	case VisibilityPublic:
		return "Pública"
	default:
		return string(v)
	}
}

// Practice representa um conhecimento prático reproduzível.
type Practice struct {
	ID       string
	AuthorID string

	AuthorHandle      string
	AuthorDisplayName string

	Title        string
	Summary      string
	Category     string
	Difficulty   Difficulty
	TimeEstimate string
	CostEstimate string
	Materials    []string
	Steps        []string
	Safety       string
	Adaptations  string
	References   string
	License      string
	Version      int
	Status       Status
	Visibility   Visibility
	PublishedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CanView informa se uma pessoa pode visualizar a prática.
func (p *Practice) CanView(viewerID string) bool {
	if p.Status == StatusPublished && p.Visibility == VisibilityPublic {
		return true
	}
	return viewerID != "" && viewerID == p.AuthorID
}

// IsOwner informa se a pessoa é autora da prática.
func (p *Practice) IsOwner(personID string) bool {
	return personID != "" && p.AuthorID == personID
}

// Input reúne os dados editáveis de uma prática.
type Input struct {
	Title        string
	Summary      string
	Category     string
	Difficulty   Difficulty
	TimeEstimate string
	CostEstimate string
	Materials    []string
	Steps        []string
	Safety       string
	Adaptations  string
	References   string
	License      string
	Publish      bool
}

// Filter reúne os filtros aplicados à listagem de práticas públicas.
type Filter struct {
	Category   string
	Difficulty Difficulty
	Limit      int
}

// NormalizeFilter preenche valores padrão e remove filtros inválidos.
func NormalizeFilter(filter Filter) Filter {
	filter.Category = strings.TrimSpace(filter.Category)
	if !filter.Difficulty.Valid() {
		filter.Difficulty = ""
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	return filter
}
