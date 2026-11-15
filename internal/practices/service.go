package practices

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service reúne as regras do contexto de conhecimento e práticas.
type Service struct {
	store *Store
	now   func() time.Time
}

// NewService cria o serviço de práticas.
func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		store: NewStore(db),
		now:   time.Now,
	}
}

// CreatePractice valida e cria uma prática.
func (s *Service) CreatePractice(ctx context.Context, authorID string, input Input) (*Practice, error) {
	input = NormalizeInput(input)
	if err := ValidateInput(input, input.Publish); err != nil {
		return nil, err
	}

	status := StatusDraft
	visibility := VisibilityPrivate
	var publishedAt *time.Time

	if input.Publish {
		status = StatusPublished
		visibility = VisibilityPublic
		now := s.now()
		publishedAt = &now
	}

	return s.store.CreatePractice(ctx, authorID, input, status, visibility, publishedAt)
}

// UpdatePractice valida e atualiza uma prática da pessoa autora.
func (s *Service) UpdatePractice(ctx context.Context, authorID, practiceID string, input Input) (*Practice, error) {
	practice, err := s.store.FindPractice(ctx, practiceID)
	if err != nil {
		return nil, err
	}
	if !practice.IsOwner(authorID) {
		return nil, ErrForbidden
	}

	input = NormalizeInput(input)
	publish := input.Publish || practice.Status == StatusPublished
	if err := ValidateInput(input, publish); err != nil {
		return nil, err
	}

	status := practice.Status
	visibility := practice.Visibility
	publishedAt := practice.PublishedAt

	if input.Publish {
		status = StatusPublished
		visibility = VisibilityPublic
		if publishedAt == nil {
			now := s.now()
			publishedAt = &now
		}
	}
	if status == StatusDraft {
		visibility = VisibilityPrivate
	}

	return s.store.UpdatePractice(ctx, practiceID, authorID, input, status, visibility, publishedAt)
}

// FindPractice busca uma prática visível para quem consulta.
func (s *Service) FindPractice(ctx context.Context, viewerID, practiceID string) (*Practice, error) {
	practice, err := s.store.FindPractice(ctx, practiceID)
	if err != nil {
		return nil, err
	}
	if !practice.CanView(viewerID) {
		return nil, ErrNotFound
	}
	return practice, nil
}

// ListPublicPractices lista práticas publicadas para descoberta.
func (s *Service) ListPublicPractices(ctx context.Context, filter Filter) ([]Practice, error) {
	return s.store.ListPublicPractices(ctx, filter)
}

// ListAuthorPractices lista as práticas de uma pessoa, incluindo rascunhos.
func (s *Service) ListAuthorPractices(ctx context.Context, authorID string) ([]Practice, error) {
	return s.store.ListAuthorPractices(ctx, authorID)
}

// PublishPractice publica uma prática existente após validar seu conteúdo.
func (s *Service) PublishPractice(ctx context.Context, authorID, practiceID string) (*Practice, error) {
	practice, err := s.store.FindPractice(ctx, practiceID)
	if err != nil {
		return nil, err
	}
	if !practice.IsOwner(authorID) {
		return nil, ErrForbidden
	}

	input := NormalizeInput(Input{
		Title:        practice.Title,
		Summary:      practice.Summary,
		Category:     practice.Category,
		Difficulty:   practice.Difficulty,
		TimeEstimate: practice.TimeEstimate,
		CostEstimate: practice.CostEstimate,
		Materials:    practice.Materials,
		Steps:        practice.Steps,
		Safety:       practice.Safety,
		Adaptations:  practice.Adaptations,
		References:   practice.References,
		License:      practice.License,
	})
	if err := ValidateInput(input, true); err != nil {
		return nil, err
	}

	publishedAt := practice.PublishedAt
	if publishedAt == nil {
		now := s.now()
		publishedAt = &now
	}

	return s.store.SetPracticeStatus(ctx, practiceID, StatusPublished, VisibilityPublic, publishedAt)
}

// ArchivePractice arquiva uma prática da pessoa autora.
func (s *Service) ArchivePractice(ctx context.Context, authorID, practiceID string) (*Practice, error) {
	practice, err := s.store.FindPractice(ctx, practiceID)
	if err != nil {
		return nil, err
	}
	if !practice.IsOwner(authorID) {
		return nil, ErrForbidden
	}

	return s.store.SetPracticeStatus(ctx, practiceID, StatusArchived, practice.Visibility, practice.PublishedAt)
}
