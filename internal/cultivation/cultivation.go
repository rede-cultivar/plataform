package cultivation

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Erros de domínio do contexto de diário de cultivo.
var (
	ErrNotFound  = errors.New("registro não encontrado")
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

// Visibility representa quem pode visualizar um ciclo.
type Visibility string

const (
	VisibilityPrivate   Visibility = "private"
	VisibilityFollowers Visibility = "followers"
	VisibilityPublic    Visibility = "public"
)

// Valid informa se a visibilidade é reconhecida.
func (v Visibility) Valid() bool {
	switch v {
	case VisibilityPrivate, VisibilityFollowers, VisibilityPublic:
		return true
	default:
		return false
	}
}

// Label retorna o rótulo em português usado na interface.
func (v Visibility) Label() string {
	switch v {
	case VisibilityPrivate:
		return "Privado"
	case VisibilityFollowers:
		return "Seguidores"
	case VisibilityPublic:
		return "Público"
	default:
		return string(v)
	}
}

// Stage representa o estágio observado de um cultivo.
type Stage string

const (
	StageGermination Stage = "germination"
	StageSeedling    Stage = "seedling"
	StageVegetative  Stage = "vegetative"
	StageFlowering   Stage = "flowering"
	StageFruiting    Stage = "fruiting"
	StageHarvest     Stage = "harvest"
	StageFinished    Stage = "finished"
)

// Valid informa se o estágio é reconhecido.
func (s Stage) Valid() bool {
	switch s {
	case StageGermination, StageSeedling, StageVegetative, StageFlowering,
		StageFruiting, StageHarvest, StageFinished:
		return true
	default:
		return false
	}
}

// Label retorna o rótulo em português usado na interface.
func (s Stage) Label() string {
	switch s {
	case StageGermination:
		return "Germinação"
	case StageSeedling:
		return "Muda"
	case StageVegetative:
		return "Vegetativo"
	case StageFlowering:
		return "Floração"
	case StageFruiting:
		return "Frutificação"
	case StageHarvest:
		return "Colheita"
	case StageFinished:
		return "Encerrado"
	default:
		return string(s)
	}
}

// Environment representa o ambiente em que o cultivo acontece.
type Environment string

const (
	EnvironmentIndoor     Environment = "indoor"
	EnvironmentOutdoor    Environment = "outdoor"
	EnvironmentGreenhouse Environment = "greenhouse"
	EnvironmentNursery    Environment = "nursery"
	EnvironmentOther      Environment = "other"
)

// Valid informa se o ambiente é reconhecido.
func (e Environment) Valid() bool {
	switch e {
	case EnvironmentIndoor, EnvironmentOutdoor, EnvironmentGreenhouse,
		EnvironmentNursery, EnvironmentOther:
		return true
	default:
		return false
	}
}

// Label retorna o rótulo em português usado na interface.
func (e Environment) Label() string {
	switch e {
	case EnvironmentIndoor:
		return "Interno"
	case EnvironmentOutdoor:
		return "Externo"
	case EnvironmentGreenhouse:
		return "Estufa"
	case EnvironmentNursery:
		return "Viveiro"
	case EnvironmentOther:
		return "Outro"
	default:
		return string(e)
	}
}

// Status representa a situação de um ciclo de cultivo.
type Status string

const (
	StatusActive   Status = "active"
	StatusFinished Status = "finished"
	StatusArchived Status = "archived"
)

// Valid informa se o status é reconhecido.
func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusFinished, StatusArchived:
		return true
	default:
		return false
	}
}

// Label retorna o rótulo em português usado na interface.
func (s Status) Label() string {
	switch s {
	case StatusActive:
		return "Em andamento"
	case StatusFinished:
		return "Concluído"
	case StatusArchived:
		return "Arquivado"
	default:
		return string(s)
	}
}

// EntryVisibility representa a visibilidade de um registro.
//
// O valor "inherit" indica que o registro usa a visibilidade do ciclo.
type EntryVisibility string

const (
	EntryVisibilityInherit   EntryVisibility = "inherit"
	EntryVisibilityPrivate   EntryVisibility = "private"
	EntryVisibilityFollowers EntryVisibility = "followers"
	EntryVisibilityPublic    EntryVisibility = "public"
)

// Valid informa se a visibilidade do registro é reconhecida.
func (v EntryVisibility) Valid() bool {
	switch v {
	case EntryVisibilityInherit, EntryVisibilityPrivate,
		EntryVisibilityFollowers, EntryVisibilityPublic:
		return true
	default:
		return false
	}
}

// Label retorna o rótulo em português usado na interface.
func (v EntryVisibility) Label() string {
	switch v {
	case EntryVisibilityInherit:
		return "Herdar do ciclo"
	case EntryVisibilityPrivate:
		return "Privado"
	case EntryVisibilityFollowers:
		return "Seguidores"
	case EntryVisibilityPublic:
		return "Público"
	default:
		return string(v)
	}
}

// Cycle representa o acompanhamento de um cultivo ao longo do tempo.
type Cycle struct {
	ID                string
	PersonID          string
	PersonHandle      string
	PersonDisplayName string

	Name        string
	Species     string
	Variety     string
	Category    string
	Environment Environment
	Location    string
	Substrate   string
	Container   string
	Technique   string
	Stage       Stage
	Status      Status
	Visibility  Visibility
	StartedOn   time.Time
	FinishedOn  *time.Time
	Outcome     string

	EntryCount int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CanView informa se uma pessoa pode visualizar o ciclo.
//
// A visibilidade para seguidores ainda não está implementada; por segurança,
// apenas a pessoa responsável e o público geral têm acesso.
func (c *Cycle) CanView(viewerID string) bool {
	if c.Visibility == VisibilityPublic {
		return true
	}
	return viewerID != "" && viewerID == c.PersonID
}

// IsOwner informa se a pessoa é responsável pelo ciclo.
func (c *Cycle) IsOwner(personID string) bool {
	return personID != "" && c.PersonID == personID
}

// Entry representa um acontecimento datado dentro de um ciclo de cultivo.
type Entry struct {
	ID       string
	CycleID  string
	AuthorID string

	AuthorHandle      string
	AuthorDisplayName string

	OccurredAt      time.Time
	Body            string
	Stage           *Stage
	Measurements    map[string]string
	Interventions   []string
	Occurrences     []string
	HarvestQuantity *float64
	HarvestUnit     string
	PracticeID      string
	PracticeTitle   string
	Visibility      EntryVisibility

	CreatedAt time.Time
	UpdatedAt time.Time
}

// EffectiveVisibility calcula a visibilidade efetiva do registro.
//
// A visibilidade do registro nunca pode ser mais ampla que a do ciclo.
func (e Entry) EffectiveVisibility(cycle *Cycle) Visibility {
	if cycle.Visibility == VisibilityPrivate {
		return VisibilityPrivate
	}

	switch e.Visibility {
	case EntryVisibilityPrivate:
		return VisibilityPrivate
	case EntryVisibilityFollowers:
		return VisibilityFollowers
	case EntryVisibilityPublic:
		if cycle.Visibility == VisibilityFollowers {
			return VisibilityFollowers
		}
		return VisibilityPublic
	default:
		return cycle.Visibility
	}
}

// CanView informa se uma pessoa pode visualizar o registro.
func (e Entry) CanView(viewerID string, cycle *Cycle) bool {
	visibility := e.EffectiveVisibility(cycle)
	if visibility == VisibilityPublic {
		return true
	}
	return viewerID != "" && viewerID == cycle.PersonID
}

// EffectiveVisibilityLabel retorna o rótulo da visibilidade efetiva.
func (e Entry) EffectiveVisibilityLabel(cycle *Cycle) string {
	return e.EffectiveVisibility(cycle).Label()
}

// StageValue retorna o estágio como texto, ou vazio quando não informado.
func (e Entry) StageValue() string {
	if e.Stage == nil {
		return ""
	}
	return string(*e.Stage)
}

// HarvestQuantityValue formata apenas a quantidade colhida.
func (e Entry) HarvestQuantityValue() string {
	if e.HarvestQuantity == nil {
		return ""
	}
	return strconv.FormatFloat(*e.HarvestQuantity, 'f', -1, 64)
}

// HarvestLabel formata a quantidade colhida com a unidade.
func (e Entry) HarvestLabel() string {
	quantity := e.HarvestQuantityValue()
	if quantity == "" {
		return ""
	}
	if e.HarvestUnit == "" {
		return quantity
	}
	return quantity + " " + e.HarvestUnit
}

// CycleInput reúne os dados necessários para criar um ciclo.
type CycleInput struct {
	Name        string
	Species     string
	Variety     string
	Category    string
	Environment Environment
	Location    string
	Substrate   string
	Container   string
	Technique   string
	Stage       Stage
	Visibility  Visibility
	StartedOn   time.Time
}

// CycleUpdateInput reúne os dados editáveis de um ciclo.
type CycleUpdateInput struct {
	Name        string
	Species     string
	Variety     string
	Category    string
	Environment Environment
	Location    string
	Substrate   string
	Container   string
	Technique   string
	Stage       Stage
	Status      Status
	Visibility  Visibility
	StartedOn   time.Time
	FinishedOn  *time.Time
	Outcome     string
}

// EntryInput reúne os dados necessários para criar um registro.
type EntryInput struct {
	OccurredAt      time.Time
	Body            string
	Stage           *Stage
	Measurements    map[string]string
	Interventions   []string
	Occurrences     []string
	HarvestQuantity *float64
	HarvestUnit     string
	PracticeID      string
	Visibility      EntryVisibility
}

// CycleFilter reúne os filtros aplicados à descoberta de ciclos públicos.
type CycleFilter struct {
	Species     string
	Environment Environment
	Stage       Stage
	Status      Status
	Limit       int
}

// NormalizeFilter preenche valores padrão e remove filtros inválidos.
func NormalizeFilter(filter CycleFilter) CycleFilter {
	filter.Species = trim(filter.Species)
	if !filter.Environment.Valid() {
		filter.Environment = ""
	}
	if !filter.Stage.Valid() {
		filter.Stage = ""
	}
	if !filter.Status.Valid() {
		filter.Status = ""
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	return filter
}
