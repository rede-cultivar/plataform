package federation

import (
	"html"
	"strings"
	"time"

	"github.com/rede-cultivar/plataform/internal/identity"
)

const (
	activityStreamsContext = "https://www.w3.org/ns/activitystreams"
	securityContext        = "https://w3id.org/security/v1"
)

// PersonActor representa uma pessoa como Actor ActivityPub.
type PersonActor struct {
	Context                   []string `json:"@context"`
	ID                        string   `json:"id"`
	Type                      string   `json:"type"`
	PreferredUsername         string   `json:"preferredUsername"`
	Name                      string   `json:"name,omitempty"`
	Summary                   string   `json:"summary,omitempty"`
	URL                       string   `json:"url"`
	Inbox                     string   `json:"inbox"`
	Outbox                    string   `json:"outbox"`
	Followers                 string   `json:"followers"`
	Following                 string   `json:"following"`
	ManuallyApprovesFollowers bool     `json:"manuallyApprovesFollowers"`
	Discoverable              bool     `json:"discoverable"`
	Published                 string   `json:"published"`
}

// NewPersonActor traduz uma pessoa do domínio para um documento Actor.
func NewPersonActor(urls URLs, person identity.Person) PersonActor {
	return PersonActor{
		Context:                   []string{activityStreamsContext, securityContext},
		ID:                        urls.Actor(person.ID),
		Type:                      "Person",
		PreferredUsername:         person.Handle,
		Name:                      person.DisplayName,
		Summary:                   summaryHTML(person.Bio),
		URL:                       urls.Profile(person.Handle),
		Inbox:                     urls.Inbox(person.ID),
		Outbox:                    urls.Outbox(person.ID),
		Followers:                 urls.Followers(person.ID),
		Following:                 urls.Following(person.ID),
		ManuallyApprovesFollowers: false,
		Discoverable:              true,
		Published:                 person.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// OrderedCollection representa uma coleção ActivityPub vazia.
//
// As coleções de outbox, seguidores e seguidos são publicadas como
// placeholders até que as publicações e os relacionamentos sejam
// implementados.
type OrderedCollection struct {
	Context      string `json:"@context"`
	ID           string `json:"id"`
	Type         string `json:"type"`
	TotalItems   int    `json:"totalItems"`
	OrderedItems []any  `json:"orderedItems"`
}

// NewOrderedCollection cria uma coleção vazia com o identificador informado.
func NewOrderedCollection(id string) OrderedCollection {
	return OrderedCollection{
		Context:      activityStreamsContext,
		ID:           id,
		Type:         "OrderedCollection",
		TotalItems:   0,
		OrderedItems: []any{},
	}
}

// summaryHTML converte a biografia em texto simples para o campo summary,
// que no ActivityPub aceita HTML.
func summaryHTML(bio string) string {
	bio = strings.TrimSpace(bio)
	if bio == "" {
		return ""
	}

	escaped := html.EscapeString(bio)
	escaped = strings.ReplaceAll(escaped, "\r\n", "\n")
	escaped = strings.ReplaceAll(escaped, "\n", "<br>")

	return "<p>" + escaped + "</p>"
}
