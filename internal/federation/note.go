package federation

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/rede-cultivar/plataform/internal/cultivation"
	"github.com/rede-cultivar/plataform/internal/identity"
	"github.com/rede-cultivar/plataform/internal/media"
)

// PublicCollection identifica o destinatário público do ActivityPub.
const PublicCollection = "https://www.w3.org/ns/activitystreams#Public"

// Attachment representa uma imagem anexada a uma publicação.
type Attachment struct {
	Type      string `json:"type"`
	MediaType string `json:"mediaType"`
	URL       string `json:"url"`
	Name      string `json:"name,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
}

// Note representa um registro de cultivo publicado no Fediverso.
type Note struct {
	Context      []string     `json:"@context"`
	ID           string       `json:"id"`
	Type         string       `json:"type"`
	AttributedTo string       `json:"attributedTo"`
	Content      string       `json:"content"`
	Published    string       `json:"published"`
	Updated      string       `json:"updated,omitempty"`
	URL          string       `json:"url"`
	To           []string     `json:"to"`
	CC           []string     `json:"cc,omitempty"`
	Attachment   []Attachment `json:"attachment,omitempty"`
}

// Create representa a atividade que distribui uma Note aos seguidores.
type Create struct {
	Context   []string `json:"@context"`
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	Actor     string   `json:"actor"`
	Published string   `json:"published"`
	To        []string `json:"to"`
	CC        []string `json:"cc,omitempty"`
	Object    Note     `json:"object"`
}

// Update representa a atividade que redistribui uma Note alterada.
//
// O identificador da Note permanece o mesmo; a instância remota substitui o
// conteúdo e os anexos da publicação original.
type Update struct {
	Context []string `json:"@context"`
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Actor   string   `json:"actor"`
	To      []string `json:"to"`
	CC      []string `json:"cc,omitempty"`
	Object  Note     `json:"object"`
}

// Tombstone representa um objeto removido do Fediverso.
type Tombstone struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// Delete representa a remoção de uma publicação federada.
//
// O Mastodon aceita o objeto como Tombstone e remove o status correspondente
// quando o Actor é o mesmo que publicou.
type Delete struct {
	Context []string  `json:"@context"`
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Actor   string    `json:"actor"`
	To      []string  `json:"to"`
	CC      []string  `json:"cc,omitempty"`
	Object  Tombstone `json:"object"`
}

// NewEntryNote traduz um registro de cultivo público para uma Note.
func NewEntryNote(urls URLs, person identity.Person, cycle cultivation.Cycle, entry cultivation.Entry, assets []media.Asset) Note {
	published := entry.CreatedAt
	if published.IsZero() {
		published = entry.OccurredAt
	}

	note := Note{
		Context:      []string{activityStreamsContext},
		ID:           urls.Note(entry.ID),
		Type:         "Note",
		AttributedTo: urls.Actor(person.ID),
		Content:      entryContent(urls, cycle, entry),
		Published:    published.UTC().Format(time.RFC3339),
		URL:          urls.Entry(cycle.ID, entry.ID),
		To:           []string{PublicCollection},
		CC:           []string{urls.Followers(person.ID)},
	}

	for _, asset := range assets {
		if asset.ID == "" || asset.ContentType == "" {
			continue
		}
		attachment := Attachment{
			Type:      "Image",
			MediaType: asset.ContentType,
			URL:       urls.Media(asset.ID),
			Width:     asset.Width,
			Height:    asset.Height,
		}
		if name := strings.TrimSpace(asset.OriginalName); name != "" && name != "imagem" {
			attachment.Name = name
		}
		note.Attachment = append(note.Attachment, attachment)
	}

	return note
}

// NewCreate monta a atividade Create que distribui a Note.
func NewCreate(urls URLs, person identity.Person, note Note, activityID string) Create {
	return Create{
		Context:   []string{activityStreamsContext},
		ID:        urls.Activity(activityID),
		Type:      "Create",
		Actor:     urls.Actor(person.ID),
		Published: note.Published,
		To:        note.To,
		CC:        note.CC,
		Object:    note,
	}
}

// NewUpdate monta a atividade Update que redistribui a Note alterada.
func NewUpdate(urls URLs, person identity.Person, note Note, activityID string) Update {
	return Update{
		Context: []string{activityStreamsContext},
		ID:      urls.Activity(activityID),
		Type:    "Update",
		Actor:   urls.Actor(person.ID),
		To:      note.To,
		CC:      note.CC,
		Object:  note,
	}
}

// NewDelete monta a atividade que remove uma publicação do Fediverso.
func NewDelete(urls URLs, person identity.Person, note Note, activityID string) Delete {
	return Delete{
		Context: []string{activityStreamsContext},
		ID:      urls.Activity(activityID),
		Type:    "Delete",
		Actor:   urls.Actor(person.ID),
		To:      note.To,
		CC:      note.CC,
		Object: Tombstone{
			ID:   note.ID,
			Type: "Tombstone",
		},
	}
}

func entryContent(urls URLs, cycle cultivation.Cycle, entry cultivation.Entry) string {
	parts := make([]string, 0, 8)

	title := strings.TrimSpace(cycle.Name)
	if title == "" {
		title = "Registro de cultivo"
	}

	header := "<p><strong>" + html.EscapeString(title) + "</strong>"
	subtitle := make([]string, 0, 2)
	if species := strings.TrimSpace(cycle.Species); species != "" {
		subtitle = append(subtitle, html.EscapeString(species))
	}
	if variety := strings.TrimSpace(cycle.Variety); variety != "" {
		subtitle = append(subtitle, html.EscapeString(variety))
	}
	if len(subtitle) > 0 {
		header += " · " + strings.Join(subtitle, " · ")
	}
	parts = append(parts, header+"</p>")

	context := make([]string, 0, 4)
	if cycle.Environment != "" {
		context = append(context, html.EscapeString(cycle.Environment.Label()))
	}
	if location := strings.TrimSpace(cycle.Location); location != "" {
		context = append(context, html.EscapeString(location))
	}
	if !cycle.StartedOn.IsZero() {
		context = append(context, "iniciado em "+cycle.StartedOn.Format("02/01/2006"))
	}
	if len(context) > 0 {
		parts = append(parts, "<p>"+strings.Join(context, " · ")+"</p>")
	}

	contentStart := len(parts)

	meta := make([]string, 0, 3)
	if !entry.OccurredAt.IsZero() {
		meta = append(meta, entry.OccurredAt.Format("02/01/2006"))
	}
	if entry.Stage != nil {
		meta = append(meta, "Estágio: "+html.EscapeString(entry.Stage.Label()))
	}
	if harvest := entry.HarvestLabel(); harvest != "" {
		meta = append(meta, "Colheita: "+html.EscapeString(harvest))
	}
	if len(meta) > 0 {
		parts = append(parts, "<p>"+strings.Join(meta, " · ")+"</p>")
	}

	body := strings.TrimSpace(entry.Body)
	if body != "" {
		escaped := html.EscapeString(body)
		escaped = strings.ReplaceAll(escaped, "\r\n", "\n")
		escaped = strings.ReplaceAll(escaped, "\n", "<br>")
		parts = append(parts, "<p>"+escaped+"</p>")
	}

	if list := entryList("Medidas", measurements(entry.Measurements)); list != "" {
		parts = append(parts, list)
	}
	if list := entryList("Intervenções", entry.Interventions); list != "" {
		parts = append(parts, list)
	}
	if list := entryList("Ocorrências", entry.Occurrences); list != "" {
		parts = append(parts, list)
	}

	if practice := strings.TrimSpace(entry.PracticeTitle); practice != "" {
		line := "Prática: "
		if entry.PracticeID != "" {
			line += `<a href="` + html.EscapeString(urls.Practice(entry.PracticeID)) + `">` +
				html.EscapeString(practice) + `</a>`
		} else {
			line += html.EscapeString(practice)
		}
		parts = append(parts, "<p>"+line+"</p>")
	}

	if len(parts) == contentStart {
		parts = append(parts, "<p>Novo registro de cultivo.</p>")
	}

	parts = append(parts,
		`<p><a href="`+html.EscapeString(urls.Entry(cycle.ID, entry.ID))+`">Ver registro completo</a></p>`,
	)

	return strings.Join(parts, "")
}

// entryList monta uma lista com título, ignorando itens vazios.
func entryList(title string, items []string) string {
	clean := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			clean = append(clean, item)
		}
	}
	if len(clean) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("<p><strong>" + html.EscapeString(title) + "</strong></p><ul>")
	for _, item := range clean {
		builder.WriteString("<li>" + html.EscapeString(item) + "</li>")
	}
	builder.WriteString("</ul>")
	return builder.String()
}

// measurements converte as medidas em linhas estáveis, ordenadas pela chave.
func measurements(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.TrimSpace(values[key])
		if value == "" {
			continue
		}
		lines = append(lines, strings.TrimSpace(key)+": "+value)
	}
	return lines
}

// newActivityID gera um identificador opaco para atividades de saída.
func newActivityID() (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("gerar identificador de atividade: %w", err)
	}
	return hex.EncodeToString(random), nil
}
