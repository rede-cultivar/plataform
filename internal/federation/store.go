package federation

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound indica que um recurso federado não existe.
var ErrNotFound = errors.New("recurso federado não encontrado")

// Store implementa o acesso ao banco para a federação.
type Store struct {
	db *pgxpool.Pool
}

// NewStore cria um repositório de federação.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// KeyRecord reúne as chaves PEM de um Actor.
type KeyRecord struct {
	PublicPEM  string
	PrivatePEM string
}

// EnsureKey grava o par de chaves quando ainda não existe e devolve o par
// efetivamente armazenado.
func (s *Store) EnsureKey(ctx context.Context, personID, publicPEM, privatePEM string) (*KeyRecord, error) {
	_, err := s.db.Exec(ctx, `
		INSERT INTO actor_keys (person_id, public_key, private_key)
		VALUES ($1::uuid, $2, $3)
		ON CONFLICT (person_id) DO NOTHING
	`, personID, publicPEM, privatePEM)
	if err != nil {
		return nil, err
	}

	return s.Key(ctx, personID)
}

// Key devolve o par de chaves da pessoa.
func (s *Store) Key(ctx context.Context, personID string) (*KeyRecord, error) {
	var record KeyRecord
	err := s.db.QueryRow(ctx, `
		SELECT public_key, private_key
		FROM actor_keys
		WHERE person_id = $1::uuid
	`, personID).Scan(&record.PublicPEM, &record.PrivatePEM)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// Follow representa um Actor remoto que segue uma pessoa local.
type Follow struct {
	ActorID        string
	InboxURL       string
	SharedInboxURL string
	AcceptedAt     *time.Time
	CreatedAt      time.Time
}

// UpsertFollow registra ou atualiza um seguidor remoto.
func (s *Store) UpsertFollow(ctx context.Context, personID, actorID, inboxURL, sharedInboxURL string, acceptedAt time.Time) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO federation_follows (
			person_id, actor_id, inbox_url, shared_inbox_url, accepted_at
		)
		VALUES ($1::uuid, $2, $3, $4, $5)
		ON CONFLICT (person_id, actor_id) DO UPDATE
		SET inbox_url = EXCLUDED.inbox_url,
		    shared_inbox_url = EXCLUDED.shared_inbox_url,
		    accepted_at = COALESCE(federation_follows.accepted_at, EXCLUDED.accepted_at),
		    updated_at = now()
	`, personID, actorID, inboxURL, sharedInboxURL, acceptedAt)
	return err
}

// RemoveFollow remove um seguidor remoto.
func (s *Store) RemoveFollow(ctx context.Context, personID, actorID string) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM federation_follows
		WHERE person_id = $1::uuid
		  AND actor_id = $2
	`, personID, actorID)
	return err
}

// ListFollowers lista os seguidores aceitos de uma pessoa.
func (s *Store) ListFollowers(ctx context.Context, personID string) ([]Follow, error) {
	rows, err := s.db.Query(ctx, `
		SELECT actor_id, inbox_url, shared_inbox_url, accepted_at, created_at
		FROM federation_follows
		WHERE person_id = $1::uuid
		  AND accepted_at IS NOT NULL
		ORDER BY created_at
	`, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	follows := make([]Follow, 0)
	for rows.Next() {
		var follow Follow
		if err := rows.Scan(
			&follow.ActorID,
			&follow.InboxURL,
			&follow.SharedInboxURL,
			&follow.AcceptedAt,
			&follow.CreatedAt,
		); err != nil {
			return nil, err
		}
		follows = append(follows, follow)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return follows, nil
}

// ActivityRecord representa uma atividade federada persistida.
type ActivityRecord struct {
	ID           string
	PersonID     string
	ActivityID   string
	ActivityType string
	ObjectID     string
	Payload      json.RawMessage
	Visibility   string
	CreatedAt    time.Time
}

// InsertActivity grava uma atividade e informa se ela é nova.
//
// A restrição única de activity_id garante idempotência no recebimento.
func (s *Store) InsertActivity(
	ctx context.Context,
	personID, activityID, activityType, objectID string,
	payload []byte,
	visibility, direction string,
) (string, bool, error) {
	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO federation_activities (
			person_id, activity_id, activity_type, object_id, payload,
			visibility, direction
		)
		VALUES ($1::uuid, $2, $3, $4, $5::jsonb, $6, $7)
		ON CONFLICT (activity_id) DO NOTHING
		RETURNING id::text
	`, personID, activityID, activityType, objectID, string(payload), visibility, direction).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

// ActivityType devolve o tipo de uma atividade já recebida.
func (s *Store) ActivityType(ctx context.Context, activityID string) (string, error) {
	var activityType string
	err := s.db.QueryRow(ctx, `
		SELECT activity_type
		FROM federation_activities
		WHERE activity_id = $1
	`, activityID).Scan(&activityType)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return activityType, nil
}

const activityColumns = `
	id::text,
	person_id::text,
	activity_id,
	activity_type,
	object_id,
	payload,
	visibility,
	created_at
`

// FindActivity busca uma atividade pelo identificador público.
func (s *Store) FindActivity(ctx context.Context, activityID string) (*ActivityRecord, error) {
	row := s.db.QueryRow(ctx, `
		SELECT `+activityColumns+`
		FROM federation_activities
		WHERE activity_id = $1
	`, activityID)

	activity, err := scanActivity(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return activity, nil
}

// ListOutbox lista as atividades públicas de saída de uma pessoa.
func (s *Store) ListOutbox(ctx context.Context, personID string, limit int) ([]ActivityRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := s.db.Query(ctx, `
		SELECT `+activityColumns+`
		FROM federation_activities
		WHERE person_id = $1::uuid
		  AND direction = 'outbound'
		  AND visibility = 'public'
		ORDER BY created_at DESC
		LIMIT $2
	`, personID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := make([]ActivityRecord, 0)
	for rows.Next() {
		activity, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}
		activities = append(activities, *activity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return activities, nil
}

// DeliveryRecord reúne os dados necessários para entregar uma atividade.
type DeliveryRecord struct {
	ID           string
	ActivityDBID string
	ActivityID   string
	InboxURL     string
	Attempts     int
	PersonID     string
	Payload      json.RawMessage
	PrivatePEM   string
}

// EnqueueDelivery agenda uma entrega para uma caixa de entrada.
func (s *Store) EnqueueDelivery(ctx context.Context, activityDBID, inboxURL string) error {
	if inboxURL == "" {
		return nil
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO federation_deliveries (activity_id, inbox_url)
		VALUES ($1::uuid, $2)
		ON CONFLICT (activity_id, inbox_url) DO NOTHING
	`, activityDBID, inboxURL)
	return err
}

// PendingDeliveries lista as entregas prontas para nova tentativa.
func (s *Store) PendingDeliveries(ctx context.Context, limit int) ([]DeliveryRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := s.db.Query(ctx, `
		SELECT
			d.id::text,
			d.activity_id::text,
			a.activity_id,
			d.inbox_url,
			d.attempts,
			a.person_id::text,
			a.payload,
			k.private_key
		FROM federation_deliveries d
		JOIN federation_activities a ON a.id = d.activity_id
		JOIN actor_keys k ON k.person_id = a.person_id
		WHERE d.status = 'pending'
		  AND d.next_attempt_at <= now()
		ORDER BY d.next_attempt_at, d.created_at
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deliveries := make([]DeliveryRecord, 0)
	for rows.Next() {
		var delivery DeliveryRecord
		if err := rows.Scan(
			&delivery.ID,
			&delivery.ActivityDBID,
			&delivery.ActivityID,
			&delivery.InboxURL,
			&delivery.Attempts,
			&delivery.PersonID,
			&delivery.Payload,
			&delivery.PrivatePEM,
		); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return deliveries, nil
}

// MarkDelivered registra uma entrega concluída.
func (s *Store) MarkDelivered(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE federation_deliveries
		SET status = 'delivered',
		    attempts = attempts + 1,
		    last_error = '',
		    updated_at = now()
		WHERE id::text = $1
	`, id)
	return err
}

// MarkFailed registra uma falha e agenda a próxima tentativa.
func (s *Store) MarkFailed(ctx context.Context, id string, attempts int, nextAttempt time.Time, lastError string, final bool) error {
	status := "pending"
	if final {
		status = "failed"
	}

	_, err := s.db.Exec(ctx, `
		UPDATE federation_deliveries
		SET status = $2,
		    attempts = $3,
		    next_attempt_at = $4,
		    last_error = $5,
		    updated_at = now()
		WHERE id::text = $1
	`, id, status, attempts, nextAttempt, lastError)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanActivity(row rowScanner) (*ActivityRecord, error) {
	var activity ActivityRecord
	if err := row.Scan(
		&activity.ID,
		&activity.PersonID,
		&activity.ActivityID,
		&activity.ActivityType,
		&activity.ObjectID,
		&activity.Payload,
		&activity.Visibility,
		&activity.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &activity, nil
}
