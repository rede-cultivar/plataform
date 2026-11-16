package cultivation

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store implementa o acesso ao banco para o contexto de diário de cultivo.
type Store struct {
	db *pgxpool.Pool
}

// NewStore cria um repositório de diário de cultivo.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

const cycleColumns = `
	c.id::text,
	c.person_id::text,
	p.handle,
	p.display_name,
	c.name,
	c.species,
	c.variety,
	c.category,
	c.environment,
	c.location,
	c.substrate,
	c.container,
	c.technique,
	c.stage,
	c.status,
	c.visibility,
	c.started_on,
	c.finished_on,
	c.outcome,
	(SELECT count(*) FROM cultivation_entries e WHERE e.cycle_id = c.id),
	c.created_at,
	c.updated_at
`

const entryColumns = `
	e.id::text,
	e.cycle_id::text,
	e.author_id::text,
	p.handle,
	p.display_name,
	e.occurred_at,
	e.body,
	e.stage,
	e.measurements,
	e.interventions,
	e.occurrences,
	e.harvest_quantity,
	e.harvest_unit,
	COALESCE(e.practice_id::text, ''),
	COALESCE(pr.title, ''),
	e.visibility,
	e.created_at,
	e.updated_at
`

type rowScanner interface {
	Scan(dest ...any) error
}

// CreateCycle insere um novo ciclo e retorna sua representação completa.
func (s *Store) CreateCycle(ctx context.Context, personID string, input CycleInput) (*Cycle, error) {
	var id string

	err := s.db.QueryRow(ctx, `
		INSERT INTO cultivation_cycles (
			person_id, name, species, variety, category, environment, location,
			substrate, container, technique, stage, visibility, started_on
		)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id::text
	`,
		personID,
		input.Name,
		input.Species,
		input.Variety,
		input.Category,
		input.Environment,
		input.Location,
		input.Substrate,
		input.Container,
		input.Technique,
		input.Stage,
		input.Visibility,
		input.StartedOn,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return s.FindCycle(ctx, id)
}

// UpdateCycle atualiza os dados editáveis de um ciclo.
func (s *Store) UpdateCycle(ctx context.Context, id string, input CycleUpdateInput) (*Cycle, error) {
	_, err := s.db.Exec(ctx, `
		UPDATE cultivation_cycles
		SET name = $2,
		    species = $3,
		    variety = $4,
		    category = $5,
		    environment = $6,
		    location = $7,
		    substrate = $8,
		    container = $9,
		    technique = $10,
		    stage = $11,
		    status = $12,
		    visibility = $13,
		    started_on = $14,
		    finished_on = $15,
		    outcome = $16,
		    updated_at = now()
		WHERE id::text = $1
	`,
		id,
		input.Name,
		input.Species,
		input.Variety,
		input.Category,
		input.Environment,
		input.Location,
		input.Substrate,
		input.Container,
		input.Technique,
		input.Stage,
		input.Status,
		input.Visibility,
		input.StartedOn,
		input.FinishedOn,
		input.Outcome,
	)
	if err != nil {
		return nil, err
	}

	return s.FindCycle(ctx, id)
}

// FindCycle busca um ciclo pelo identificador.
func (s *Store) FindCycle(ctx context.Context, id string) (*Cycle, error) {
	row := s.db.QueryRow(ctx, `
		SELECT `+cycleColumns+`
		FROM cultivation_cycles c
		JOIN people p ON p.id = c.person_id
		WHERE c.id::text = $1
	`, id)

	cycle, err := scanCycle(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return cycle, nil
}

// ListPersonCycles lista os ciclos de uma pessoa, incluindo privados.
func (s *Store) ListPersonCycles(ctx context.Context, personID string) ([]Cycle, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+cycleColumns+`
		FROM cultivation_cycles c
		JOIN people p ON p.id = c.person_id
		WHERE c.person_id = $1::uuid
		ORDER BY c.started_on DESC, c.created_at DESC
	`, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCycles(rows)
}

// ListPublicCycles lista ciclos públicos, aplicando os filtros informados.
func (s *Store) ListPublicCycles(ctx context.Context, filter CycleFilter) ([]Cycle, error) {
	filter = NormalizeFilter(filter)

	rows, err := s.db.Query(ctx, `
		SELECT `+cycleColumns+`
		FROM cultivation_cycles c
		JOIN people p ON p.id = c.person_id
		WHERE c.visibility = 'public'
		  AND c.status <> 'archived'
		  AND ($1 = '' OR c.species ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR c.environment = $2)
		  AND ($3 = '' OR c.stage = $3)
		  AND ($4 = '' OR c.status = $4)
		ORDER BY c.started_on DESC, c.created_at DESC
		LIMIT $5
	`,
		filter.Species,
		string(filter.Environment),
		string(filter.Stage),
		string(filter.Status),
		filter.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCycles(rows)
}

// CreateEntry insere um registro em um ciclo.
func (s *Store) CreateEntry(ctx context.Context, cycleID, authorID string, input EntryInput) (*Entry, error) {
	measurements, err := json.Marshal(input.Measurements)
	if err != nil {
		return nil, err
	}

	var stage *string
	if input.Stage != nil {
		value := string(*input.Stage)
		stage = &value
	}

	var practiceID any
	if input.PracticeID != "" {
		practiceID = input.PracticeID
	}

	var id string
	err = s.db.QueryRow(ctx, `
		INSERT INTO cultivation_entries (
			cycle_id, author_id, occurred_at, body, stage, measurements,
			interventions, occurrences, harvest_quantity, harvest_unit,
			practice_id, visibility
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11::uuid, $12)
		RETURNING id::text
	`,
		cycleID,
		authorID,
		input.OccurredAt,
		input.Body,
		stage,
		string(measurements),
		input.Interventions,
		input.Occurrences,
		input.HarvestQuantity,
		input.HarvestUnit,
		practiceID,
		input.Visibility,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return s.FindEntry(ctx, id)
}

// FindEntry busca um registro pelo identificador.
func (s *Store) FindEntry(ctx context.Context, id string) (*Entry, error) {
	row := s.db.QueryRow(ctx, `
		SELECT `+entryColumns+`
		FROM cultivation_entries e
		JOIN people p ON p.id = e.author_id
		LEFT JOIN practices pr ON pr.id = e.practice_id
		WHERE e.id::text = $1
	`, id)

	entry, err := scanEntry(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return entry, nil
}

// UpdateEntry atualiza os dados editáveis de um registro.
func (s *Store) UpdateEntry(ctx context.Context, id string, input EntryInput) (*Entry, error) {
	measurements, err := json.Marshal(input.Measurements)
	if err != nil {
		return nil, err
	}

	var stage *string
	if input.Stage != nil {
		value := string(*input.Stage)
		stage = &value
	}

	var practiceID any
	if input.PracticeID != "" {
		practiceID = input.PracticeID
	}

	_, err = s.db.Exec(ctx, `
		UPDATE cultivation_entries
		SET occurred_at = $2,
		    body = $3,
		    stage = $4,
		    measurements = $5::jsonb,
		    interventions = $6,
		    occurrences = $7,
		    harvest_quantity = $8,
		    harvest_unit = $9,
		    practice_id = $10::uuid,
		    visibility = $11,
		    updated_at = now()
		WHERE id::text = $1
	`,
		id,
		input.OccurredAt,
		input.Body,
		stage,
		string(measurements),
		input.Interventions,
		input.Occurrences,
		input.HarvestQuantity,
		input.HarvestUnit,
		practiceID,
		input.Visibility,
	)
	if err != nil {
		return nil, err
	}

	return s.FindEntry(ctx, id)
}

// ListEntries lista os registros de um ciclo em ordem cronológica inversa.
func (s *Store) ListEntries(ctx context.Context, cycleID string) ([]Entry, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+entryColumns+`
		FROM cultivation_entries e
		JOIN people p ON p.id = e.author_id
		LEFT JOIN practices pr ON pr.id = e.practice_id
		WHERE e.cycle_id::text = $1
		ORDER BY e.occurred_at DESC, e.created_at DESC
	`, cycleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func scanCycles(rows pgx.Rows) ([]Cycle, error) {
	cycles := make([]Cycle, 0)
	for rows.Next() {
		cycle, err := scanCycle(rows)
		if err != nil {
			return nil, err
		}
		cycles = append(cycles, *cycle)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cycles, nil
}

func scanCycle(row rowScanner) (*Cycle, error) {
	var cycle Cycle
	var finishedOn *time.Time

	err := row.Scan(
		&cycle.ID,
		&cycle.PersonID,
		&cycle.PersonHandle,
		&cycle.PersonDisplayName,
		&cycle.Name,
		&cycle.Species,
		&cycle.Variety,
		&cycle.Category,
		&cycle.Environment,
		&cycle.Location,
		&cycle.Substrate,
		&cycle.Container,
		&cycle.Technique,
		&cycle.Stage,
		&cycle.Status,
		&cycle.Visibility,
		&cycle.StartedOn,
		&finishedOn,
		&cycle.Outcome,
		&cycle.EntryCount,
		&cycle.CreatedAt,
		&cycle.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	cycle.FinishedOn = finishedOn
	return &cycle, nil
}

func scanEntry(row rowScanner) (*Entry, error) {
	var entry Entry
	var stage *string
	var measurements []byte
	var harvestQuantity *float64

	err := row.Scan(
		&entry.ID,
		&entry.CycleID,
		&entry.AuthorID,
		&entry.AuthorHandle,
		&entry.AuthorDisplayName,
		&entry.OccurredAt,
		&entry.Body,
		&stage,
		&measurements,
		&entry.Interventions,
		&entry.Occurrences,
		&harvestQuantity,
		&entry.HarvestUnit,
		&entry.PracticeID,
		&entry.PracticeTitle,
		&entry.Visibility,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if stage != nil {
		value := Stage(*stage)
		entry.Stage = &value
	}
	if len(measurements) > 0 {
		if err := json.Unmarshal(measurements, &entry.Measurements); err != nil {
			return nil, err
		}
	}
	if entry.Measurements == nil {
		entry.Measurements = map[string]string{}
	}
	entry.HarvestQuantity = harvestQuantity

	return &entry, nil
}
