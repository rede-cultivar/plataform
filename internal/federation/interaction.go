package federation

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// maxInteractionContent limita o texto guardado de uma resposta remota.
//
// O limite é aplicado depois da extração do HTML e serve para evitar que uma
// atividade malformada ocupe espaço desproporcional na página do registro.
const maxInteractionContent = 4000

// MaxReplyContent limita o texto de uma resposta publicada pelo Cultivar.
const MaxReplyContent = 2000

// ErrInvalidReply indica que o texto da resposta está vazio ou é longo demais.
var ErrInvalidReply = errors.New("resposta inválida")

// InteractionKind identifica o tipo de interação recebida.
type InteractionKind string

const (
	// InteractionReply é uma resposta publicada por outra pessoa.
	InteractionReply InteractionKind = "reply"
	// InteractionLike é uma curtida.
	InteractionLike InteractionKind = "like"
	// InteractionAnnounce é um impulso (boost).
	InteractionAnnounce InteractionKind = "announce"
)

// Interaction representa uma interação recebida do Fediverso.
//
// O conteúdo das respostas já vem em texto simples, pronto para ser escapado
// pelo template. Reações não têm conteúdo.
type Interaction struct {
	ID          string
	EntryID     string
	Kind        InteractionKind
	Direction   string
	ActivityID  string
	ObjectID    string
	ActorID     string
	ActorHandle string
	ActorName   string
	ActorURL    string
	ActorAvatar string
	ActorInbox  string
	InReplyTo   string
	ParentID    string
	Content     string
	PublishedAt *time.Time
	CreatedAt   time.Time
}

// Local informa se a resposta foi publicada a partir do Cultivar.
func (i Interaction) Local() bool {
	return i.Direction == "outbound"
}

// ActorLabel escolhe o nome exibível do autor da interação.
func (i Interaction) ActorLabel() string {
	if name := strings.TrimSpace(i.ActorName); name != "" {
		return name
	}
	if handle := strings.TrimSpace(i.ActorHandle); handle != "" {
		return "@" + handle
	}
	return i.ActorID
}

// PublishedLabel formata a data de publicação para a interface.
func (i Interaction) PublishedLabel() string {
	if i.PublishedAt == nil || i.PublishedAt.IsZero() {
		return ""
	}
	return i.PublishedAt.Format("02/01/2006 15:04")
}

// EntryInteractions reúne as interações de um registro.
type EntryInteractions struct {
	Replies   []Interaction
	Likes     []Interaction
	Announces []Interaction
}

// HasAny informa se o registro recebeu alguma interação.
func (e EntryInteractions) HasAny() bool {
	return len(e.Replies) > 0 || len(e.Likes) > 0 || len(e.Announces) > 0
}

// ReplySummary descreve a quantidade de respostas.
func (e EntryInteractions) ReplySummary() string {
	return countLabel(len(e.Replies), "resposta", "respostas")
}

// LikeSummary descreve a quantidade de curtidas.
func (e EntryInteractions) LikeSummary() string {
	return countLabel(len(e.Likes), "curtida", "curtidas")
}

// AnnounceSummary descreve a quantidade de impulsos.
func (e EntryInteractions) AnnounceSummary() string {
	return countLabel(len(e.Announces), "impulso", "impulsos")
}

func countLabel(count int, singular, plural string) string {
	switch count {
	case 0:
		return ""
	case 1:
		return "1 " + singular
	default:
		return fmt.Sprintf("%d %s", count, plural)
	}
}

// replyPayload reúne os campos de uma Note remota usados como resposta.
type replyPayload struct {
	ObjectID     string
	InReplyTo    string
	AttributedTo string
	Content      string
	PublishedAt  *time.Time
}

// parseReplyObject interpreta uma Note recebida.
func parseReplyObject(object any) (replyPayload, bool) {
	value, ok := object.(map[string]any)
	if !ok {
		return replyPayload{}, false
	}
	if kind, _ := value["type"].(string); kind != "Note" {
		return replyPayload{}, false
	}

	payload := replyPayload{
		ObjectID:     objectID(value),
		InReplyTo:    actorField(value["inReplyTo"]),
		AttributedTo: actorField(value["attributedTo"]),
		Content:      plainText(stringField(value, "content")),
	}
	if published := stringField(value, "published"); published != "" {
		if parsed, err := time.Parse(time.RFC3339, published); err == nil {
			payload.PublishedAt = &parsed
		}
	}

	return payload, true
}

func stringField(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return strings.TrimSpace(value)
}

// actorField extrai um identificador de Actor de string, objeto ou lista.
func actorField(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		id, _ := typed["id"].(string)
		return strings.TrimSpace(id)
	case []any:
		for _, item := range typed {
			if id := actorField(item); id != "" {
				return id
			}
		}
	}
	return ""
}

// entryIDFromNoteURL extrai o identificador de um registro de uma URL
// /notas/{id} pertencente a esta instância.
func entryIDFromNoteURL(urls URLs, noteURL string) (string, bool) {
	noteURL = strings.TrimSpace(noteURL)
	if noteURL == "" {
		return "", false
	}

	parsed, err := url.Parse(noteURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	base, err := url.Parse(urls.Base())
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", false
	}
	if !strings.EqualFold(parsed.Scheme, base.Scheme) || !strings.EqualFold(parsed.Host, base.Host) {
		return "", false
	}

	prefix := strings.TrimSuffix(base.Path, "/") + "/notas/"
	path := strings.TrimSuffix(parsed.Path, "/")
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}

	id := strings.ToLower(strings.TrimPrefix(path, prefix))
	if !isUUID(id) {
		return "", false
	}
	return id, true
}

// isUUID valida o formato canônico usado pelos identificadores do banco.
func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		switch index {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			if !((char >= '0' && char <= '9') ||
				(char >= 'a' && char <= 'f') ||
				(char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}
	return true
}

// plainText extrai texto legível de um conteúdo HTML remoto.
//
// O texto é normalizado, tem as tags removidas e é truncado para caber na
// página do registro. A interface escapa o resultado novamente antes de exibir.
func plainText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	document, err := html.Parse(strings.NewReader(value))
	if err != nil {
		return truncateText(normalizeText(html.UnescapeString(value)), maxInteractionContent)
	}

	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && (node.Data == "script" || node.Data == "style") {
			return
		}
		if node.Type == html.TextNode {
			builder.WriteString(node.Data)
		}
		if node.Type == html.ElementNode {
			switch node.Data {
			case "br":
				appendLineBreak(&builder)
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}

		if node.Type == html.ElementNode {
			switch node.Data {
			case "p", "div", "li", "blockquote":
				appendLineBreak(&builder)
			}
		}
	}
	walk(document)

	return truncateText(normalizeText(builder.String()), maxInteractionContent)
}

func appendLineBreak(builder *strings.Builder) {
	text := builder.String()
	if text != "" && !strings.HasSuffix(text, "\n") {
		builder.WriteString("\n")
	}
}

// normalizeText remove espaços repetidos e linhas vazias.
func normalizeText(value string) string {
	lines := strings.Split(value, "\n")
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			clean = append(clean, line)
		}
	}
	return strings.Join(clean, "\n")
}

// truncateText limita o texto por quantidade de caracteres, não de bytes.
func truncateText(value string, limit int) string {
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return strings.TrimSpace(string(runes[:limit])) + "…"
}

// EntryInteractions carrega as interações recebidas de vários registros.
//
// A interface web usa o resultado para exibir respostas e contadores na página
// do ciclo sem consultar o banco diretamente.
func (s *Service) EntryInteractions(ctx context.Context, entryIDs []string) (map[string]EntryInteractions, error) {
	return s.store.ListEntryInteractions(ctx, entryIDs)
}

// handleReply processa uma resposta recebida em Create.
//
// A atividade só é considerada uma interação quando responde a uma Note desta
// instância e quando o registro ainda pode ser federado. Caso contrário, ela é
// apenas registrada como atividade recebida.
func (s *Service) handleReply(ctx context.Context, personID string, remote *RemoteActor, activity InboxActivity, body []byte) (bool, error) {
	payload, ok := parseReplyObject(activity.Object)
	if !ok || payload.InReplyTo == "" || payload.ObjectID == "" {
		return false, nil
	}
	if payload.AttributedTo != "" && !sameURL(payload.AttributedTo, remote.ID) {
		return true, fmt.Errorf("%w: resposta atribuída a outro Actor", ErrInvalidActivity)
	}

	entryID, ok := entryIDFromNoteURL(s.urls, payload.InReplyTo)
	if !ok {
		return false, nil
	}
	owner, visibility, err := s.store.FederatedEntryOwner(ctx, entryID)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if owner != personID {
		return false, nil
	}
	allowed, err := s.allowedInteraction(ctx, personID, visibility, remote.ID)
	if err != nil {
		return false, err
	}
	if !allowed {
		return false, nil
	}

	reply := Interaction{
		EntryID:     entryID,
		Kind:        InteractionReply,
		Direction:   "inbound",
		ActivityID:  activity.ID,
		ObjectID:    payload.ObjectID,
		ActorID:     remote.ID,
		ActorHandle: remote.PreferredUsername,
		ActorName:   remote.Name,
		ActorURL:    remoteProfileURL(remote),
		ActorAvatar: remote.Icon.URL,
		ActorInbox:  remote.InboxURL(),
		InReplyTo:   payload.InReplyTo,
		ParentID:    payload.InReplyTo,
		Content:     payload.Content,
		PublishedAt: payload.PublishedAt,
	}
	if err := s.store.UpsertReply(ctx, personID, entryID, reply); err != nil {
		return true, err
	}

	return true, s.recordInbound(ctx, personID, activity, body)
}

// handleReplyUpdate atualiza o texto de uma resposta já conhecida.
func (s *Service) handleReplyUpdate(ctx context.Context, personID string, remote *RemoteActor, activity InboxActivity, body []byte) (bool, error) {
	payload, ok := parseReplyObject(activity.Object)
	if !ok || payload.ObjectID == "" {
		return false, nil
	}
	if payload.AttributedTo != "" && !sameURL(payload.AttributedTo, remote.ID) {
		return true, fmt.Errorf("%w: resposta atribuída a outro Actor", ErrInvalidActivity)
	}
	entryID, ok := entryIDFromNoteURL(s.urls, payload.InReplyTo)
	if !ok {
		return false, nil
	}
	owner, visibility, err := s.store.FederatedEntryOwner(ctx, entryID)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if owner != personID {
		return false, nil
	}
	allowed, err := s.allowedInteraction(ctx, personID, visibility, remote.ID)
	if err != nil {
		return false, err
	}
	if !allowed {
		return false, nil
	}

	if err := s.store.UpdateReply(ctx, remote.ID, payload.ObjectID, payload.Content, payload.PublishedAt); err != nil {
		return true, err
	}
	return true, s.recordInbound(ctx, personID, activity, body)
}

// handleReaction processa uma curtida ou um impulso.
func (s *Service) handleReaction(ctx context.Context, personID string, remote *RemoteActor, activity InboxActivity, body []byte, kind InteractionKind) (bool, error) {
	noteURL := objectID(activity.Object)
	entryID, ok := entryIDFromNoteURL(s.urls, noteURL)
	if !ok {
		return false, nil
	}
	owner, visibility, err := s.store.FederatedEntryOwner(ctx, entryID)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if owner != personID {
		return false, nil
	}
	allowed, err := s.allowedInteraction(ctx, personID, visibility, remote.ID)
	if err != nil {
		return false, err
	}
	if !allowed {
		return false, nil
	}

	reaction := Interaction{
		EntryID:     entryID,
		Kind:        kind,
		ActivityID:  activity.ID,
		ObjectID:    noteURL,
		ActorID:     remote.ID,
		ActorHandle: remote.PreferredUsername,
		ActorName:   remote.Name,
		ActorURL:    remoteProfileURL(remote),
		ActorAvatar: remote.Icon.URL,
	}
	if err := s.store.UpsertReaction(ctx, personID, entryID, reaction); err != nil {
		return true, err
	}

	return true, s.recordInbound(ctx, personID, activity, body)
}

// allowedInteraction informa se o Actor remoto pode interagir com o registro.
//
// Conteúdo público aceita qualquer Actor. Conteúdo restrito a seguidores só
// aceita quem segue a pessoa local e teve o Follow aceito.
func (s *Service) allowedInteraction(ctx context.Context, personID, visibility, actorID string) (bool, error) {
	if visibility != "followers" {
		return true, nil
	}
	return s.store.IsRemoteFollower(ctx, personID, actorID)
}

// handleDelete remove uma resposta ou os dados de uma conta excluída.
func (s *Service) handleDelete(ctx context.Context, personID string, remote *RemoteActor, activity InboxActivity) error {
	target := objectID(activity.Object)
	kind := objectType(activity.Object)

	if sameURL(target, remote.ID) || kind == "Person" || kind == "Service" || kind == "Application" {
		if err := s.store.RemoveFollow(ctx, personID, remote.ID); err != nil {
			return err
		}
		return s.store.RemoveActorInteractions(ctx, remote.ID)
	}
	if target == "" {
		return nil
	}

	return s.store.DeleteReply(ctx, remote.ID, target)
}

// undoReaction remove uma curtida ou impulso a partir de uma atividade Undo.
func (s *Service) undoReaction(ctx context.Context, remote *RemoteActor, activity InboxActivity) error {
	kind, ok := reactionKind(objectType(activity.Object))
	if !ok {
		if activityID := objectID(activity.Object); activityID != "" {
			return s.store.DeleteReactionByActivity(ctx, remote.ID, activityID)
		}
		return nil
	}

	if value, isObject := activity.Object.(map[string]any); isObject {
		if entryID, found := entryIDFromNoteURL(s.urls, objectID(value["object"])); found {
			return s.store.DeleteReaction(ctx, remote.ID, entryID, kind)
		}
	}
	if activityID := objectID(activity.Object); activityID != "" {
		return s.store.DeleteReactionByActivity(ctx, remote.ID, activityID)
	}
	return nil
}

// reactionKind converte o tipo ActivityPub em um tipo interno de reação.
func reactionKind(activityType string) (InteractionKind, bool) {
	switch activityType {
	case "Like":
		return InteractionLike, true
	case "Announce":
		return InteractionAnnounce, true
	default:
		return "", false
	}
}

// remoteProfileURL escolhe o endereço público do perfil remoto.
func remoteProfileURL(remote *RemoteActor) string {
	if remote == nil {
		return ""
	}
	if profile := strings.TrimSpace(remote.URL); profile != "" {
		return profile
	}
	return remote.ID
}
