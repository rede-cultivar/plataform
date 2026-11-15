package practices

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store implementa o acesso ao banco para o contexto de práticas.
type Store struct {
	db *pgxpool.Pool
}

// NewStore cria um repositório de práticas.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

const practiceColumns = `
	p.id::text,
	p.author_id::text,
	pe.handle,
	pe.display_name,
	p.title,
	p.summary,
	p.category,
	p.difficulty,
	p.time_estimate,
	p.cost_estimate,
	p.materials,
	p.steps,
	p.safety,
	p.adaptations,
	p.source_references,
	p.license,
	p.version,
	p.status,
	p.visibility,
	p.published_at,
	p.created_at,
	p.updated_at
`

type rowScanner interface {
	Scan(dest ...any) error
}

// CreatePractice cria a prática e sua primeira versão.
func (s *Store) CreatePractice(ctx context.Context, authorID string, input Input, status Status, visibility Visibility, publishedAt *time.Time) (*Practice, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	materials, err := json.Marshal(input.Materials)
	if err != nil {
		return nil, err
	}
	steps, err := json.Marshal(input.Steps)
	if err != nil {
		return nil, err
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO practices (
			author_id, title, summary, category, difficulty, time_estimate,
			cost_estimate, materials, steps, safety, adaptations,
			source_references, license, status, visibility, published_at
		)
		VALUES (
			$1::uuid, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb,
			$10, $11, $12, $13, $14, $15, $16
		)
		RETURNING id::text
	`,
		authorID,
		input.Title,
		input.Summary,
		input.Category,
		input.Difficulty,
		input.TimeEstimate,
		input.CostEstimate,
		string(materials),
		string(steps),
		input.Safety,
		input.Adaptations,
		input.References,
		input.License,
		status,
		visibility,
		publishedAt,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	if err := insertVersion(ctx, tx, id, authorID, 1, input); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.FindPractice(ctx, id)
}

// UpdatePractice atualiza a prática e registra uma nova versão.
func (s *Store) UpdatePractice(ctx context.Context, id, authorID string, input Input, status Status, visibility Visibility, publishedAt *time.Time) (*Practice, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	materials, err := json.Marshal(input.Materials)
	if err != nil {
		return nil, err
	}
	steps, err := json.Marshal(input.Steps)
	if err != nil {
		return nil, err
	}

	var version int
	err = tx.QueryRow(ctx, `
		UPDATE practices
		SET title = $2,
		    summary = $3,
		    category = $4,
		    difficulty = $5,
		    time_estimate = $6,
		    cost_estimate = $7,
		    materials = $8::jsonb,
		    steps = $9::jsonb,
		    safety = $10,
		    adaptations = $11,
		    source_references = $12,
		    license = $13,
		    version = version + 1,
		    status = $14,
		    visibility = $15,
		    published_at = $16,
		    updated_at = now()
		WHERE id::text = $1
		RETURNING version
	`,
		id,
		input.Title,
		input.Summary,
		input.Category,
		input.Difficulty,
		input.TimeEstimate,
		input.CostEstimate,
		string(materials),
		string(steps),
		input.Safety,
		input.Adaptations,
		input.References,
		input.License,
		status,
		visibility,
		publishedAt,
	).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := insertVersion(ctx, tx, id, authorID, version, input); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.FindPractice(ctx, id)
}

// SetPracticeStatus altera a situação editorial sem criar nova versão de conteúdo.
func (s *Store) SetPracticeStatus(ctx context.Context, id string, status Status, visibility Visibility, publishedAt *time.Time) (*Practice, error) {
	_, err := s.db.Exec(ctx, `
		UPDATE practices
		SET status = $2,
		    visibility = $3,
		    published_at = $4,
		    updated_at = now()
		WHERE id::text = $1
	`, id, status, visibility, publishedAt)
	if err != nil {
		return nil, err
	}

	return s.FindPractice(ctx, id)
}

// FindPractice busca uma prática pelo identificador.
func (s *Store) FindPractice(ctx context.Context, id string) (*Practice, error) {
	row := s.db.QueryRow(ctx, `
		SELECT `+practiceColumns+`
		FROM practices p
		JOIN people pe ON pe.id = p.author_id
		WHERE p.id::text = $1
	`, id)

	practice, err := scanPractice(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return practice, nil
}

// ListPublicPractices lista práticas publicadas e públicas.
func (s *Store) ListPublicPractices(ctx context.Context, filter Filter) ([]Practice, error) {
	filter = NormalizeFilter(filter)

	rows, err := s.db.Query(ctx, `
		SELECT `+practiceColumns+`
		FROM practices p
		JOIN people pe ON pe.id = p.author_id
		WHERE p.status = 'published'
		  AND p.visibility = 'public'
		  AND ($1 = '' OR p.category ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR p.difficulty = $2)
		ORDER BY p.published_at DESC NULLS LAST, p.created_at DESC
		LIMIT $3
	`,
		filter.Category,
		string(filter.Difficulty),
		filter.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPractices(rows)
}

// ListAuthorPractices lista todas as práticas de uma pessoa, incluindo rascunhos.
func (s *Store) ListAuthorPractices(ctx context.Context, authorID string) ([]Practice, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+practiceColumns+`
		FROM practices p
		JOIN people pe ON pe.id = p.author_id
		WHERE p.author_id = $1::uuid
		ORDER BY p.updated_at DESC, p.created_at DESC
	`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPractices(rows)
}

func scanPractices(rows pgx.Rows) ([]Practice, error) {
	result := make([]Practice, 0)
	for rows.Next() {
		practice, err := scanPractice(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *practice)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanPractice(row rowScanner) (*Practice, error) {
	var practice Practice
	var materials, steps []byte
	var publishedAt *time.Time

	err := row.Scan(
		&practice.ID,
		&practice.AuthorID,
		&practice.AuthorHandle,
		&practice.AuthorDisplayName,
		&practice.Title,
		&practice.Summary,
		&practice.Category,
		&practice.Difficulty,
		&practice.TimeEstimate,
		&practice.CostEstimate,
		&materials,
		&steps,
		&practice.Safety,
		&practice.Adaptations,
		&practice.References,
		&practice.License,
		&practice.Version,
		&practice.Status,
		&practice.Visibility,
		&publishedAt,
		&practice.CreatedAt,
		&practice.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(materials) > 0 {
		if err := json.Unmarshal(materials, &practice.Materials); err != nil {
			return nil, err
		}
	}
	if len(steps) > 0 {
		if err := json.Unmarshal(steps, &practice.Steps); err != nil {
			return nil, err
		}
	}
	if practice.Materials == nil {
		practice.Materials = []string{}
	}
	if practice.Steps == nil {
		practice.Steps = []string{}
	}
	practice.PublishedAt = publishedAt

	return &practice, nil
}

func insertVersion(ctx context.Context, tx pgx.Tx, practiceID, authorID string, version int, input Input) error {
	snapshot, err := json.Marshal(map[string]any{
		"title":         input.Title,
		"summary":       input.Summary,
		"category":      input.Category,
		"difficulty":    input.Difficulty,
		"time_estimate": input.TimeEstimate,
		"cost_estimate": input.CostEstimate,
		"materials":     input.Materials,
		"steps":         input.Steps,
		"safety":        input.Safety,
		"adaptations":   input.Adaptations,
		"references":    input.References,
		"license":       input.License,
	})
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO practice_versions (practice_id, version, author_id, snapshot)
		VALUES ($1::uuid, $2, $3::uuid, $4::jsonb)
	`, practiceID, version, authorID, string(snapshot))
	return err
}
