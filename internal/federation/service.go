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
	"sync"
	"time"
	"unicode/utf8"

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
	// backfillLimit é a quantidade de publicações recentes enviada a um novo
	// seguidor quando o Follow é aceito.
	backfillLimit = 20
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

	profileMu sync.Mutex
	profiles  map[string]cachedFollower
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
		profiles:    make(map[string]cachedFollower),
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

// PublishEntry publica um registro como Note e distribui a atividade.
//
// Registros privados não são publicados. Registros visíveis apenas para
// seguidores são entregues sem incluir a coleção Public.
func (s *Service) PublishEntry(
	ctx context.Context,
	person identity.Person,
	cycle cultivation.Cycle,
	entry cultivation.Entry,
	assets []media.Asset,
) error {
	visibility, ok := activityVisibility(entry.EffectiveVisibility(&cycle))
	if !ok {
		return nil
	}

	if _, err := s.ActorKey(ctx, person.ID); err != nil {
		return err
	}

	note := NewEntryNote(s.urls, person, cycle, entry, assets)
	return s.publishCreate(ctx, person, note, s.urls.Note(entry.ID), visibility)
}

// PublishEntryUpdate redistribui uma publicação já federada após uma alteração.
//
// O Mastodon só atualiza publicações que já recebeu, por isso a Note precisa ter
// sido distribuída antes. Quando não há publicação anterior, o registro é
// publicado como Create. Quando a visibilidade muda entre público e seguidores,
// o identificador do objeto muda: a versão anterior recebe Delete e a nova é
// publicada como Create.
func (s *Service) PublishEntryUpdate(
	ctx context.Context,
	person identity.Person,
	cycle cultivation.Cycle,
	entry cultivation.Entry,
	assets []media.Asset,
) error {
	visibility, ok := activityVisibility(entry.EffectiveVisibility(&cycle))
	if !ok {
		return nil
	}

	if _, err := s.ActorKey(ctx, person.ID); err != nil {
		return err
	}

	note := NewEntryNote(s.urls, person, cycle, entry, assets)
	objectID := s.urls.Note(entry.ID)

	latest, err := s.store.LatestObjectActivity(ctx, person.ID, objectID)
	if err != nil {
		return err
	}
	if latest.Type == "" || latest.Type == "Delete" {
		return s.publishCreate(ctx, person, note, objectID, visibility)
	}
	if latest.Visibility != visibility {
		if err := s.publishEntryDelete(ctx, person, cycle, entry); err != nil {
			return err
		}
		return s.publishCreate(ctx, person, note, objectID, visibility)
	}

	note.Updated = s.now().UTC().Format(time.RFC3339)

	id, err := newActivityID()
	if err != nil {
		return err
	}

	update := NewUpdate(s.urls, person, note, id)
	payload, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("codificar atividade Update: %w", err)
	}

	activityDBID, inserted, err := s.store.InsertActivity(
		ctx, person.ID, update.ID, "Update", objectID, payload, visibility, "outbound",
	)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	return s.enqueueFollowers(ctx, person.ID, activityDBID)
}

// PublishEntryDelete remove do Fediverso uma publicação que deixou de ser
// federada.
//
// A atividade só é enviada quando existe uma publicação anterior. Registros
// nunca publicados não geram tráfego. O Tombstone reaproveita o identificador da
// última versão publicada, inclusive quando ela tinha token de capacidade.
func (s *Service) PublishEntryDelete(
	ctx context.Context,
	person identity.Person,
	cycle cultivation.Cycle,
	entry cultivation.Entry,
) error {
	return s.publishEntryDelete(ctx, person, cycle, entry)
}

func (s *Service) publishEntryDelete(
	ctx context.Context,
	person identity.Person,
	cycle cultivation.Cycle,
	entry cultivation.Entry,
) error {
	if _, err := s.ActorKey(ctx, person.ID); err != nil {
		return err
	}

	objectID := s.urls.Note(entry.ID)

	latest, err := s.store.LatestObjectActivity(ctx, person.ID, objectID)
	if err != nil {
		return err
	}
	if latest.Type != "Create" && latest.Type != "Update" {
		return nil
	}

	previous := visibilityValue(latest.Visibility)
	note := newEntryNote(s.urls, person, cycle, entry, nil, previous)

	id, err := newActivityID()
	if err != nil {
		return err
	}

	deletion := NewDelete(s.urls, person, note, id)
	payload, err := json.Marshal(deletion)
	if err != nil {
		return fmt.Errorf("codificar atividade Delete: %w", err)
	}

	activityDBID, inserted, err := s.store.InsertActivity(
		ctx, person.ID, deletion.ID, "Delete", objectID, payload, latest.Visibility, "outbound",
	)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	return s.enqueueFollowers(ctx, person.ID, activityDBID)
}

// publishCreate grava e distribui uma nova atividade Create.
//
// O identificador do objeto é gravado sem o token de capacidade, para que
// consultas posteriores encontrem a publicação em qualquer visibilidade.
func (s *Service) publishCreate(ctx context.Context, person identity.Person, note Note, objectID, visibility string) error {
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
		ctx, person.ID, create.ID, "Create", objectID, payload, visibility, "outbound",
	)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	return s.enqueueFollowers(ctx, person.ID, activityDBID)
}

// activityVisibility converte a visibilidade do diário para o valor guardado na
// federação. Visibilidade privada não gera atividade.
func activityVisibility(visibility cultivation.Visibility) (string, bool) {
	switch visibility {
	case cultivation.VisibilityPublic:
		return "public", true
	case cultivation.VisibilityFollowers:
		return "followers", true
	default:
		return "", false
	}
}

// visibilityValue interpreta a visibilidade guardada na federação.
func visibilityValue(value string) cultivation.Visibility {
	switch value {
	case "public":
		return cultivation.VisibilityPublic
	case "followers":
		return cultivation.VisibilityFollowers
	default:
		return cultivation.VisibilityPrivate
	}
}

// ReplyToInteraction publica uma resposta a uma mensagem recebida.
//
// A resposta é uma Note própria, com inReplyTo apontando para a mensagem remota
// e uma menção ao seu autor. A atividade Create é persistida e entregue ao inbox
// de quem respondeu e aos seguidores, seguindo o mesmo caminho das publicações
// de registros.
func (s *Service) ReplyToInteraction(
	ctx context.Context,
	person identity.Person,
	cycle cultivation.Cycle,
	entry cultivation.Entry,
	replyID, content string,
) (*Interaction, error) {
	content = strings.TrimSpace(content)
	if content == "" || utf8.RuneCountInString(content) > MaxReplyContent {
		return nil, ErrInvalidReply
	}
	effective := entry.EffectiveVisibility(&cycle)
	visibility, federated := activityVisibility(effective)
	if entry.AuthorID != person.ID || !federated {
		return nil, ErrNotFound
	}
	if _, err := s.ActorKey(ctx, person.ID); err != nil {
		return nil, err
	}

	parent, err := s.store.FindReply(ctx, person.ID, entry.ID, replyID)
	if err != nil {
		return nil, err
	}

	noteID, err := newActivityID()
	if err != nil {
		return nil, err
	}
	activityID, err := newActivityID()
	if err != nil {
		return nil, err
	}

	published := s.now().UTC()
	target := ReplyTarget{
		ActorID:     parent.ActorID,
		ActorHandle: parent.ActorHandle,
		ActorURL:    parent.ActorURL,
		ObjectID:    parent.ObjectID,
	}
	note := NewReplyNote(s.urls, person, entry, noteID, target, content, effective, published)
	create := NewCreate(s.urls, person, note, activityID)

	payload, err := json.Marshal(create)
	if err != nil {
		return nil, fmt.Errorf("codificar atividade Create: %w", err)
	}

	activityDBID, inserted, err := s.store.InsertActivity(
		ctx, person.ID, create.ID, "Create", note.ID, payload, visibility, "outbound",
	)
	if err != nil {
		return nil, err
	}
	if !inserted {
		return nil, ErrInvalidReply
	}

	local := Interaction{
		EntryID:     entry.ID,
		Kind:        InteractionReply,
		Direction:   "outbound",
		ActivityID:  create.ID,
		ObjectID:    note.ID,
		ActorID:     s.urls.Actor(person.ID),
		ActorHandle: person.Handle,
		ActorName:   person.DisplayName,
		ActorURL:    s.urls.Profile(person.Handle),
		InReplyTo:   parent.ObjectID,
		ParentID:    parent.ObjectID,
		Content:     content,
		PublishedAt: &published,
	}
	if err := s.store.InsertOutboundReply(ctx, person.ID, entry.ID, local); err != nil {
		return nil, err
	}

	inbox := parent.ActorInbox
	if inbox == "" {
		if actor, fetchErr := fetchActorDocument(ctx, s.client, parent.ActorID); fetchErr == nil {
			inbox = actor.InboxURL()
		} else {
			s.logger.Warn("fetch replied actor inbox", "actor", parent.ActorID, "error", fetchErr)
		}
	}
	if err := s.store.EnqueueDelivery(ctx, activityDBID, inbox); err != nil {
		s.logger.Warn("enqueue reply delivery", "actor", parent.ActorID, "error", err)
	}
	if err := s.enqueueFollowers(ctx, person.ID, activityDBID); err != nil {
		s.logger.Warn("enqueue reply followers", "entry", entry.ID, "error", err)
	}

	return &local, nil
}

// FindReplyNote devolve a Note de uma resposta publicada pelo Cultivar.
func (s *Service) FindReplyNote(ctx context.Context, replyID string) (Note, error) {
	payload, err := s.store.FindFederatedReplyObject(ctx, s.urls.ReplyNote(replyID))
	if err != nil {
		return Note{}, err
	}

	var create Create
	if err := json.Unmarshal(payload, &create); err != nil {
		return Note{}, fmt.Errorf("interpretar resposta publicada: %w", err)
	}
	return create.Object, nil
}

// enqueueFollowers agenda a atividade para todos os seguidores aceitos.
func (s *Service) enqueueFollowers(ctx context.Context, personID, activityDBID string) error {
	followers, err := s.store.ListFollowers(ctx, personID)
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

// Outbox devolve a coleção de publicações públicas de saída.
func (s *Service) Outbox(ctx context.Context, personID string) (OrderedCollection, error) {
	activities, err := s.store.ListOutbox(ctx, personID, 50)
	if err != nil {
		return OrderedCollection{}, err
	}

	items := outboxItems(activities)

	return OrderedCollection{
		Context:      activityStreamsContext,
		ID:           s.urls.Outbox(personID),
		Type:         "OrderedCollection",
		TotalItems:   len(items),
		OrderedItems: items,
	}, nil
}

// outboxItems converte apenas publicações Create em itens do outbox.
//
// Atividades de controle, como Accept, não representam publicações e não
// devem aparecer no totalItems que outras instâncias usam como contagem de
// publicações.
func outboxItems(activities []ActivityRecord) []any {
	items := make([]any, 0, len(activities))
	for _, activity := range activities {
		if activity.ActivityType != "Create" {
			continue
		}
		items = append(items, json.RawMessage(activity.Payload))
	}
	return items
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

// Following devolve a coleção de pessoas seguidas pelo Actor.
//
// A coleção inclui as relações locais, que são imediatas e não passam por
// aprovação. Contas remotas seguidas entram aqui quando o acompanhamento remoto
// estiver disponível.
func (s *Service) Following(ctx context.Context, personID string) (OrderedCollection, error) {
	ids, err := s.store.ListFollowing(ctx, personID)
	if err != nil {
		return OrderedCollection{}, err
	}

	items := make([]any, 0, len(ids))
	for _, id := range ids {
		items = append(items, s.urls.Actor(id))
	}

	return OrderedCollection{
		Context:      activityStreamsContext,
		ID:           s.urls.Following(personID),
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
	// Atividades públicas e restritas a seguidores usam identificadores opacos.
	// Quem recebeu a atividade pode buscá-la; o restante continua sem acesso.
	if activity.Visibility != "public" && activity.Visibility != "followers" {
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
	case "Create":
		handled, err := s.handleReply(ctx, personID, remote, activity, body)
		if handled || err != nil {
			return err
		}
	case "Update":
		handled, err := s.handleReplyUpdate(ctx, personID, remote, activity, body)
		if handled || err != nil {
			return err
		}
	case "Like":
		handled, err := s.handleReaction(ctx, personID, remote, activity, body, InteractionLike)
		if handled || err != nil {
			return err
		}
	case "Announce":
		handled, err := s.handleReaction(ctx, personID, remote, activity, body, InteractionAnnounce)
		if handled || err != nil {
			return err
		}
	case "Delete":
		if err := s.handleDelete(ctx, personID, remote, activity); err != nil {
			return err
		}
	}

	return s.recordInbound(ctx, personID, activity, body)
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

	if err := s.sendAccept(ctx, personID, remote, body); err != nil {
		return err
	}

	s.backfillFollow(ctx, personID, remote)
	return nil
}

func (s *Service) handleUndo(ctx context.Context, personID string, remote *RemoteActor, activity InboxActivity, body []byte) error {
	if kind := objectType(activity.Object); kind == "Follow" {
		if actor := embeddedActor(activity.Object); actor != "" && !sameURL(actor, remote.ID) {
			return fmt.Errorf("%w: Undo de Follow de outro Actor", ErrInvalidActivity)
		}
		if err := s.store.RemoveFollow(ctx, personID, remote.ID); err != nil {
			return err
		}
	} else if _, ok := reactionKind(kind); ok {
		if actor := embeddedActor(activity.Object); actor != "" && !sameURL(actor, remote.ID) {
			return fmt.Errorf("%w: Undo de reação de outro Actor", ErrInvalidActivity)
		}
		if err := s.undoReaction(ctx, remote, activity); err != nil {
			return err
		}
	} else if id := objectID(activity.Object); id != "" {
		activityType, err := s.store.ActivityType(ctx, id)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}

		switch activityType {
		case "Follow":
			if err := s.store.RemoveFollow(ctx, personID, remote.ID); err != nil {
				return err
			}
		case "Like", "Announce":
			if err := s.store.DeleteReactionByActivity(ctx, remote.ID, id); err != nil {
				return err
			}
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

// backfillFollow envia as publicações recentes ao novo seguidor.
//
// O Mastodon não importa itens do outbox ao abrir um perfil: ele mostra apenas
// as atividades que recebeu. Sem esse envio, quem segue depois da publicação
// não vê os registros anteriores. O backfill inclui conteúdo restrito a
// seguidores porque o Follow já foi aceito e o novo seguidor está autorizado.
func (s *Service) backfillFollow(ctx context.Context, personID string, remote *RemoteActor) {
	activities, err := s.store.ListBackfill(ctx, personID, backfillLimit)
	if err != nil {
		s.logger.Warn("list federation backfill", "person", personID, "error", err)
		return
	}

	inbox := remote.InboxURL()
	for _, activity := range activities {
		if err := s.store.EnqueueDelivery(ctx, activity.ID, inbox); err != nil {
			s.logger.Warn("enqueue federation backfill",
				"activity", activity.ActivityID,
				"inbox", inbox,
				"error", err,
			)
		}
	}
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
