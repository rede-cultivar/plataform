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

	viewer, err := s.viewerFor(ctx, viewerID, cycle.PersonID)
	if err != nil {
		return nil, nil, err
	}
	if !cycle.CanView(viewer) {
		return nil, nil, ErrNotFound
	}

	entries, err := s.store.ListEntries(ctx, cycleID)
	if err != nil {
		return nil, nil, err
	}

	visible := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.CanView(viewer, cycle) {
			visible = append(visible, entry)
		}
	}

	return cycle, visible, nil
}

// FindCycleAsFollower busca um ciclo e os registros visíveis a um seguidor.
//
// Deve ser usado apenas depois de validar um token de capacidade da página do
// ciclo. O seguidor enxerga registros públicos e restritos a seguidores, mas
// nunca registros privados. Não há vínculo local: o token é a própria
// autorização.
func (s *Service) FindCycleAsFollower(ctx context.Context, cycleID string) (*Cycle, []Entry, error) {
	cycle, err := s.store.FindCycle(ctx, cycleID)
	if err != nil {
		return nil, nil, err
	}
	if cycle.Visibility == VisibilityPrivate {
		return nil, nil, ErrNotFound
	}

	entries, err := s.store.ListEntries(ctx, cycleID)
	if err != nil {
		return nil, nil, err
	}

	visible := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.EffectiveVisibility(cycle) != VisibilityPrivate {
			visible = append(visible, entry)
		}
	}

	return cycle, visible, nil
}

// viewerFor descreve quem consulta o conteúdo de uma pessoa.
func (s *Service) viewerFor(ctx context.Context, viewerID, ownerID string) (Viewer, error) {
	viewer := Viewer{PersonID: viewerID}
	if viewerID == "" || viewerID == ownerID {
		return viewer, nil
	}

	following, err := s.store.IsFollowing(ctx, viewerID, ownerID)
	if err != nil {
		return Viewer{}, err
	}
	viewer.Following = following
	return viewer, nil
}

// FindEntry busca um registro pelo identificador, sem verificar visibilidade.
//
// A verificação de acesso é responsabilidade de quem consulta, que precisa
// carregar o ciclo correspondente.
func (s *Service) FindEntry(ctx context.Context, entryID string) (*Entry, error) {
	return s.store.FindEntry(ctx, entryID)
}

// ListPersonCycles lista os ciclos de uma pessoa, incluindo privados.
//
// Deve ser usado apenas pela própria pessoa ou por fluxos internos. Para
// exibir o perfil de alguém, use ListVisibleCycles.
func (s *Service) ListPersonCycles(ctx context.Context, personID string) ([]Cycle, error) {
	return s.store.ListPersonCycles(ctx, personID)
}

// ListVisibleCycles lista os ciclos de uma pessoa visíveis para quem consulta.
func (s *Service) ListVisibleCycles(ctx context.Context, viewerID, personID string) ([]Cycle, error) {
	return s.store.ListVisibleCycles(ctx, viewerID, personID)
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
