package cultivation

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service reúne as regras do diário de cultivo.
type Service struct {
	store *Store
	now   func() time.Time
}

// NewService cria o serviço de diário de cultivo.
func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		store: NewStore(db),
		now:   time.Now,
	}
}

// CreateCycle valida e cria um ciclo para a pessoa responsável.
func (s *Service) CreateCycle(ctx context.Context, personID string, input CycleInput) (*Cycle, error) {
	input = NormalizeCycleInput(input)
	if err := ValidateCycleInput(input); err != nil {
		return nil, err
	}

	return s.store.CreateCycle(ctx, personID, input)
}

// UpdateCycle valida e atualiza um ciclo da pessoa responsável.
func (s *Service) UpdateCycle(ctx context.Context, personID, cycleID string, input CycleUpdateInput) (*Cycle, error) {
	cycle, err := s.store.FindCycle(ctx, cycleID)
	if err != nil {
		return nil, err
	}
	if !cycle.IsOwner(personID) {
		return nil, ErrForbidden
	}

	input = NormalizeCycleUpdateInput(input)
	if err := ValidateCycleUpdateInput(input); err != nil {
		return nil, err
	}

	if input.Status == StatusFinished {
		if input.Stage != StageFinished {
			input.Stage = StageFinished
		}
		if input.FinishedOn == nil {
			finished := s.now()
			input.FinishedOn = &finished
		}
	}
	if input.Status == StatusActive {
		input.FinishedOn = nil
	}

	return s.store.UpdateCycle(ctx, cycleID, input)
}

// FindCycle busca um ciclo e os registros visíveis para quem consulta.
func (s *Service) FindCycle(ctx context.Context, viewerID, cycleID string) (*Cycle, []Entry, error) {
	cycle, err := s.store.FindCycle(ctx, cycleID)
	if err != nil {
		return nil, nil, err
	}
	if !cycle.CanView(viewerID) {
		return nil, nil, ErrNotFound
	}

	entries, err := s.store.ListEntries(ctx, cycleID)
	if err != nil {
		return nil, nil, err
	}

	visible := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.CanView(viewerID, cycle) {
			visible = append(visible, entry)
		}
	}

	return cycle, visible, nil
}

// FindEntry busca um registro pelo identificador, sem verificar visibilidade.
//
// A verificação de acesso é responsabilidade de quem consulta, que precisa
// carregar o ciclo correspondente.
func (s *Service) FindEntry(ctx context.Context, entryID string) (*Entry, error) {
	return s.store.FindEntry(ctx, entryID)
}

// ListPersonCycles lista os ciclos de uma pessoa.
func (s *Service) ListPersonCycles(ctx context.Context, personID string) ([]Cycle, error) {
	return s.store.ListPersonCycles(ctx, personID)
}

// ListPublicCycles lista ciclos públicos para descoberta.
func (s *Service) ListPublicCycles(ctx context.Context, filter CycleFilter) ([]Cycle, error) {
	return s.store.ListPublicCycles(ctx, filter)
}

// CreateEntry valida e cria um registro em um ciclo da pessoa responsável.
func (s *Service) CreateEntry(ctx context.Context, personID, cycleID string, input EntryInput) (*Entry, error) {
	cycle, err := s.store.FindCycle(ctx, cycleID)
	if err != nil {
		return nil, err
	}
	if !cycle.IsOwner(personID) {
		return nil, ErrForbidden
	}

	input = NormalizeEntryInput(input)
	if err := ValidateEntryInput(input); err != nil {
		return nil, err
	}

	return s.store.CreateEntry(ctx, cycleID, personID, input)
}

// UpdateEntry valida e atualiza um registro do ciclo da pessoa responsável.
func (s *Service) UpdateEntry(ctx context.Context, personID, cycleID, entryID string, input EntryInput) (*Entry, error) {
	cycle, err := s.store.FindCycle(ctx, cycleID)
	if err != nil {
		return nil, err
	}
	if !cycle.IsOwner(personID) {
		return nil, ErrForbidden
	}

	entry, err := s.store.FindEntry(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if entry.CycleID != cycle.ID {
		return nil, ErrForbidden
	}

	input = NormalizeEntryInput(input)
	if err := ValidateEntryInput(input); err != nil {
		return nil, err
	}

	return s.store.UpdateEntry(ctx, entryID, input)
}
