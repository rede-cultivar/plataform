package federation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rede-cultivar/plataform/internal/cultivation"
	"github.com/rede-cultivar/plataform/internal/identity"
	"github.com/rede-cultivar/plataform/internal/media"
)

const (
	// maxInboxBytes limita o corpo de uma atividade recebida.
	maxInboxBytes = 1 << 20
	// maxDeliveryAttempts encerra entregas que falham repetidamente.
	maxDeliveryAttempts = 8
)

// Service reúne as regras de publicação e recebimento federado.
type Service struct {
	store       *Store
	urls        URLs
	logger      *slog.Logger
	client      *http.Client
	verifier    *Verifier
	now         func() time.Time
	maxAttempts int
}

// NewService cria o serviço de federação.
//
// O parâmetro allowPrivate libera endereços privados em desenvolvimento, para
// testes com instâncias locais. Em produção deve permanecer falso.
func NewService(db *pgxpool.Pool, urls URLs, logger *slog.Logger, allowPrivate bool) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	client := NewHTTPClient(allowPrivate)

	return &Service{
		store:       NewStore(db),
		urls:        urls,
		logger:      logger,
		client:      client,
		verifier:    NewVerifier(client),
		now:         time.Now,
		maxAttempts: maxDeliveryAttempts,
	}
}

// ActorKey devolve o par de chaves do Actor, criando-o na primeira chamada.
func (s *Service) ActorKey(ctx context.Context, personID string) (*KeyPair, error) {
	record, err := s.store.Key(ctx, personID)
	if err == nil {
		return &KeyPair{PublicPEM: record.PublicPEM, PrivatePEM: record.PrivatePEM}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	generated, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	record, err = s.store.EnsureKey(ctx, personID, generated.PublicPEM, generated.PrivatePEM)
	if err != nil {
		return nil, err
	}

	return &KeyPair{PublicPEM: record.PublicPEM, PrivatePEM: record.PrivatePEM}, nil
}

// PublishEntry publica um registro público como Note e distribui a atividade.
//
// Registros privados ou visíveis apenas para seguidores não são publicados.
func (s *Service) PublishEntry(
	ctx context.Context,
	person identity.Person,
	cycle cultivation.Cycle,
	entry cultivation.Entry,
	assets []media.Asset,
) error {
	if entry.EffectiveVisibility(&cycle) != cultivation.VisibilityPublic {
		return nil
	}

	if _, err := s.ActorKey(ctx, person.ID); err != nil {
		return err
	}

	note := NewEntryNote(s.urls, person, cycle, entry, assets)

	id, err := newActivityID()
	if err != nil {
		return err
	}

	create := NewCreate(s.urls, person, note, id)
	payload, err := json.Marshal(create)
	if err != nil {
		return fmt.Errorf("codificar atividade Create: %w", err)
	}

	activityDBID, inserted, err := s.store.InsertActivity(
		ctx, person.ID, create.ID, "Create", note.ID, payload, "public", "outbound",
	)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	followers, err := s.store.ListFollowers(ctx, person.ID)
	if err != nil {
		return err
	}

	for _, follower := range followers {
		inbox := follower.SharedInboxURL
		if inbox == "" {
			inbox = follower.InboxURL
		}
		if err := s.store.EnqueueDelivery(ctx, activityDBID, inbox); err != nil {
			s.logger.Warn("enqueue federation delivery", "actor", follower.ActorID, "error", err)
		}
	}

	return nil
}

// Outbox devolve a coleção de atividades públicas de saída.
func (s *Service) Outbox(ctx context.Context, personID string) (OrderedCollection, error) {
	activities, err := s.store.ListOutbox(ctx, personID, 50)
	if err != nil {
		return OrderedCollection{}, err
	}

	items := make([]any, 0, len(activities))
	for _, activity := range activities {
		items = append(items, json.RawMessage(activity.Payload))
	}

	return OrderedCollection{
		Context:      activityStreamsContext,
		ID:           s.urls.Outbox(personID),
		Type:         "OrderedCollection",
		TotalItems:   len(items),
		OrderedItems: items,
	}, nil
}

// Followers devolve a coleção de seguidores aceitos.
func (s *Service) Followers(ctx context.Context, personID string) (OrderedCollection, error) {
	follows, err := s.store.ListFollowers(ctx, personID)
	if err != nil {
		return OrderedCollection{}, err
	}

	items := make([]any, 0, len(follows))
	for _, follow := range follows {
		items = append(items, follow.ActorID)
	}

	return OrderedCollection{
		Context:      activityStreamsContext,
		ID:           s.urls.Followers(personID),
		Type:         "OrderedCollection",
		TotalItems:   len(items),
		OrderedItems: items,
	}, nil
}

// FindActivity devolve o documento de uma atividade de saída.
func (s *Service) FindActivity(ctx context.Context, activityID string) (json.RawMessage, error) {
	activity, err := s.store.FindActivity(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if activity.Visibility != "public" {
		return nil, ErrNotFound
	}
	return activity.Payload, nil
}

// InboxActivity representa uma atividade recebida no inbox.
type InboxActivity struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Actor  string `json:"actor"`
	Object any    `json:"object"`
}

// HandleInbox valida a assinatura e processa uma atividade recebida.
func (s *Service) HandleInbox(ctx context.Context, personID string, req *http.Request) error {
	body, err := io.ReadAll(io.LimitReader(req.Body, maxInboxBytes+1))
	if err != nil {
		return fmt.Errorf("%w: leitura do corpo falhou", ErrInvalidActivity)
	}
	if len(body) > maxInboxBytes {
		return fmt.Errorf("%w: corpo grande demais", ErrInvalidActivity)
	}

	remote, err := s.verifier.VerifyRequest(ctx, req, body)
	if err != nil {
		return err
	}

	var activity InboxActivity
	if err := json.Unmarshal(body, &activity); err != nil {
		return fmt.Errorf("%w: JSON inválido", ErrInvalidActivity)
	}
	if activity.ID == "" || activity.Type == "" || activity.Actor == "" {
		return fmt.Errorf("%w: campos obrigatórios ausentes", ErrInvalidActivity)
	}
	if !sameURL(activity.Actor, remote.ID) {
		return fmt.Errorf("%w: Actor da atividade não confere com a assinatura", ErrInvalidSignature)
	}

	switch activity.Type {
	case "Follow":
		return s.handleFollow(ctx, personID, remote, activity, body)
	case "Undo":
		return s.handleUndo(ctx, personID, remote, activity, body)
	case "Delete":
		_ = s.store.RemoveFollow(ctx, personID, remote.ID)
		return s.recordInbound(ctx, personID, activity, body)
	default:
		return s.recordInbound(ctx, personID, activity, body)
	}
}

func (s *Service) handleFollow(ctx context.Context, personID string, remote *RemoteActor, activity InboxActivity, body []byte) error {
	if objectID(activity.Object) != s.urls.Actor(personID) {
		return fmt.Errorf("%w: Follow não aponta para este Actor", ErrInvalidActivity)
	}
	if remote.Inbox == "" {
		return fmt.Errorf("%w: Actor remoto sem inbox", ErrInvalidActivity)
	}

	_, inserted, err := s.store.InsertActivity(
		ctx, personID, activity.ID, "Follow", objectID(activity.Object), body, "public", "inbound",
	)
	if err != nil {
		return err
	}

	if err := s.store.UpsertFollow(ctx, personID, remote.ID, remote.Inbox, remote.SharedInbox(), s.now()); err != nil {
		return err
	}

	if !inserted {
		return nil
	}

	return s.sendAccept(ctx, personID, remote, body)
}

func (s *Service) handleUndo(ctx context.Context, personID string, remote *RemoteActor, activity InboxActivity, body []byte) error {
	if objectType(activity.Object) == "Follow" {
		if actor := embeddedActor(activity.Object); actor != "" && !sameURL(actor, remote.ID) {
			return fmt.Errorf("%w: Undo de Follow de outro Actor", ErrInvalidActivity)
		}
		if err := s.store.RemoveFollow(ctx, personID, remote.ID); err != nil {
			return err
		}
	} else if id := objectID(activity.Object); id != "" {
		activityType, err := s.store.ActivityType(ctx, id)
		if err == nil && activityType == "Follow" {
			if err := s.store.RemoveFollow(ctx, personID, remote.ID); err != nil {
				return err
			}
		} else if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}

	return s.recordInbound(ctx, personID, activity, body)
}

func (s *Service) recordInbound(ctx context.Context, personID string, activity InboxActivity, body []byte) error {
	_, _, err := s.store.InsertActivity(
		ctx, personID, activity.ID, activity.Type, objectID(activity.Object), body, "public", "inbound",
	)
	return err
}

func (s *Service) sendAccept(ctx context.Context, personID string, remote *RemoteActor, followPayload []byte) error {
	id, err := newActivityID()
	if err != nil {
		return err
	}

	accept := struct {
		Context string          `json:"@context"`
		ID      string          `json:"id"`
		Type    string          `json:"type"`
		Actor   string          `json:"actor"`
		Object  json.RawMessage `json:"object"`
	}{
		Context: activityStreamsContext,
		ID:      s.urls.Activity(id),
		Type:    "Accept",
		Actor:   s.urls.Actor(personID),
		Object:  json.RawMessage(followPayload),
	}

	payload, err := json.Marshal(accept)
	if err != nil {
		return fmt.Errorf("codificar atividade Accept: %w", err)
	}

	activityDBID, inserted, err := s.store.InsertActivity(
		ctx, personID, accept.ID, "Accept", "", payload, "public", "outbound",
	)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	return s.store.EnqueueDelivery(ctx, activityDBID, remote.InboxURL())
}

// ProcessDeliveries entrega as atividades pendentes e devolve quantas foram
// concluídas com sucesso.
func (s *Service) ProcessDeliveries(ctx context.Context) (int, error) {
	deliveries, err := s.store.PendingDeliveries(ctx, 20)
	if err != nil {
		return 0, err
	}

	delivered := 0
	for _, delivery := range deliveries {
		if err := s.deliver(ctx, delivery); err != nil {
			attempts := delivery.Attempts + 1
			final := attempts >= s.maxAttempts
			next := s.now().Add(deliveryBackoff(attempts))

			if markErr := s.store.MarkFailed(ctx, delivery.ID, attempts, next, truncateError(err), final); markErr != nil {
				s.logger.Error("mark federation delivery failed", "delivery", delivery.ID, "error", markErr)
			}
			s.logger.Warn("federation delivery failed",
				"delivery", delivery.ID,
				"inbox", delivery.InboxURL,
				"attempts", attempts,
				"final", final,
				"error", err,
			)
			continue
		}

		if err := s.store.MarkDelivered(ctx, delivery.ID); err != nil {
			s.logger.Error("mark federation delivery delivered", "delivery", delivery.ID, "error", err)
			continue
		}
		delivered++
	}

	return delivered, nil
}

func (s *Service) deliver(ctx context.Context, delivery DeliveryRecord) error {
	key, err := ParsePrivateKey(delivery.PrivatePEM)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, delivery.InboxURL, bytes.NewReader(delivery.Payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", ActivityContentType)
	req.Header.Set("Accept", ActivityContentType)

	if err := SignRequest(req, delivery.Payload, s.urls.Key(delivery.PersonID), key); err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("inbox respondeu %d", resp.StatusCode)
	}

	return nil
}

func objectID(object any) string {
	switch value := object.(type) {
	case string:
		return value
	case map[string]any:
		id, _ := value["id"].(string)
		return id
	default:
		return ""
	}
}

func objectType(object any) string {
	switch value := object.(type) {
	case string:
		return ""
	case map[string]any:
		activityType, _ := value["type"].(string)
		return activityType
	default:
		return ""
	}
}

func embeddedActor(object any) string {
	value, ok := object.(map[string]any)
	if !ok {
		return ""
	}
	actor, _ := value["actor"].(string)
	return actor
}

func deliveryBackoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 6 {
		attempts = 6
	}
	backoff := time.Minute * time.Duration(1<<attempts)
	if backoff > time.Hour {
		return time.Hour
	}
	return backoff
}

func truncateError(err error) string {
	message := strings.TrimSpace(err.Error())
	if len(message) > 500 {
		return message[:500]
	}
	return message
}
