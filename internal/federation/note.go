package federation

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
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
		Content:      entryContent(cycle, entry),
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

func entryContent(cycle cultivation.Cycle, entry cultivation.Entry) string {
	parts := make([]string, 0, 4)

	title := strings.TrimSpace(cycle.Name)
	if title != "" {
		line := "<p><strong>" + html.EscapeString(title) + "</strong>"
		if species := strings.TrimSpace(cycle.Species); species != "" {
			line += " · " + html.EscapeString(species)
		}
		parts = append(parts, line+"</p>")
	}

	body := strings.TrimSpace(entry.Body)
	if body != "" {
		escaped := html.EscapeString(body)
		escaped = strings.ReplaceAll(escaped, "\r\n", "\n")
		escaped = strings.ReplaceAll(escaped, "\n", "<br>")
		parts = append(parts, "<p>"+escaped+"</p>")
	}

	if entry.Stage != nil {
		parts = append(parts, "<p>Estágio: "+html.EscapeString(entry.Stage.Label())+"</p>")
	}

	if harvest := entry.HarvestLabel(); harvest != "" {
		parts = append(parts, "<p>Colheita: "+html.EscapeString(harvest)+"</p>")
	}

	if len(parts) == 0 {
		parts = append(parts, "<p>Novo registro de cultivo.</p>")
	}

	return strings.Join(parts, "")
}

// newActivityID gera um identificador opaco para atividades de saída.
func newActivityID() (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("gerar identificador de atividade: %w", err)
	}
	return hex.EncodeToString(random), nil
}
