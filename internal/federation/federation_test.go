package federation

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rede-cultivar/plataform/internal/identity"
)

func testPerson() identity.Person {
	return identity.Person{
		ID:          "11111111-1111-1111-1111-111111111111",
		Handle:      "ana",
		DisplayName: "Ana Silva",
		Bio:         "Cultivo hortas.\nCompartilho sementes.",
		CreatedAt:   time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC),
	}
}

func TestURLs(t *testing.T) {
	urls := NewURLs("http://localhost:8080/")

	if got := urls.Base(); got != "http://localhost:8080" {
		t.Errorf("Base() = %q, want %q", got, "http://localhost:8080")
	}
	if got := urls.Host(); got != "localhost:8080" {
		t.Errorf("Host() = %q, want %q", got, "localhost:8080")
	}
	if got := urls.Actor("abc"); got != "http://localhost:8080/actors/abc" {
		t.Errorf("Actor() = %q", got)
	}
	if got := urls.Inbox("abc"); got != "http://localhost:8080/actors/abc/inbox" {
		t.Errorf("Inbox() = %q", got)
	}
	if got := urls.Outbox("abc"); got != "http://localhost:8080/actors/abc/outbox" {
		t.Errorf("Outbox() = %q", got)
	}
	if got := urls.Followers("abc"); got != "http://localhost:8080/actors/abc/followers" {
		t.Errorf("Followers() = %q", got)
	}
	if got := urls.Following("abc"); got != "http://localhost:8080/actors/abc/following" {
		t.Errorf("Following() = %q", got)
	}
	if got := urls.Profile("ana"); got != "http://localhost:8080/perfil/ana" {
		t.Errorf("Profile() = %q", got)
	}
	if got := urls.Practice("abc"); got != "http://localhost:8080/praticas/abc" {
		t.Errorf("Practice() = %q", got)
	}
}

func TestParseSignatureHeaderReportsParameterNames(t *testing.T) {
	_, err := ParseSignatureHeader(`sig1=:signature-value:`)
	if err == nil {
		t.Fatal("ParseSignatureHeader() error = nil, want missing legacy parameters")
	}
	if !strings.Contains(err.Error(), "campos recebidos: sig1") {
		t.Errorf("error = %q, want received parameter name", err)
	}
	if strings.Contains(err.Error(), "signature-value") {
		t.Errorf("error exposes signature value: %q", err)
	}
}

func TestSignedHeaderValueUsesRequestHost(t *testing.T) {
	request := httptest.NewRequest("POST", "https://cultivar.example/actors/test/inbox", nil)

	if got := signedHeaderValue(request, "host"); got != "cultivar.example" {
		t.Fatalf("signedHeaderValue(host) = %q, want %q", got, "cultivar.example")
	}
}

func TestNewPersonActor(t *testing.T) {
	urls := NewURLs("https://cultivar.example")
	actor := NewPersonActor(urls, testPerson())

	if actor.ID != "https://cultivar.example/actors/11111111-1111-1111-1111-111111111111" {
		t.Errorf("ID = %q", actor.ID)
	}
	if actor.Type != "Person" {
		t.Errorf("Type = %q, want Person", actor.Type)
	}
	if actor.PreferredUsername != "ana" {
		t.Errorf("PreferredUsername = %q, want ana", actor.PreferredUsername)
	}
	if actor.Name != "Ana Silva" {
		t.Errorf("Name = %q, want Ana Silva", actor.Name)
	}
	if actor.Summary != "<p>Cultivo hortas.<br>Compartilho sementes.</p>" {
		t.Errorf("Summary = %q", actor.Summary)
	}
	if actor.URL != "https://cultivar.example/perfil/ana" {
		t.Errorf("URL = %q", actor.URL)
	}
	if actor.Inbox != actor.ID+"/inbox" {
		t.Errorf("Inbox = %q", actor.Inbox)
	}
	if actor.Outbox != actor.ID+"/outbox" {
		t.Errorf("Outbox = %q", actor.Outbox)
	}
	if actor.Followers != actor.ID+"/followers" {
		t.Errorf("Followers = %q", actor.Followers)
	}
	if actor.Following != actor.ID+"/following" {
		t.Errorf("Following = %q", actor.Following)
	}
	if actor.Published != "2026-11-05T12:00:00Z" {
		t.Errorf("Published = %q", actor.Published)
	}
	if !actor.Discoverable {
		t.Error("Discoverable = false, want true")
	}
	if actor.ManuallyApprovesFollowers {
		t.Error("ManuallyApprovesFollowers = true, want false")
	}
	if len(actor.Context) != 2 {
		t.Errorf("Context = %v, want two entries", actor.Context)
	}
}

func TestNewPersonActorEscapesSummary(t *testing.T) {
	person := testPerson()
	person.Bio = `<script>alert("x")</script>`

	actor := NewPersonActor(NewURLs("https://cultivar.example"), person)

	if strings.Contains(actor.Summary, "<script>") {
		t.Errorf("Summary = %q, want escaped HTML", actor.Summary)
	}
	if !strings.Contains(actor.Summary, "&lt;script&gt;") {
		t.Errorf("Summary = %q, want escaped script tag", actor.Summary)
	}
}

func TestNewOrderedCollection(t *testing.T) {
	collection := NewOrderedCollection("https://cultivar.example/actors/abc/outbox")

	if collection.Type != "OrderedCollection" {
		t.Errorf("Type = %q, want OrderedCollection", collection.Type)
	}
	if collection.TotalItems != 0 {
		t.Errorf("TotalItems = %d, want 0", collection.TotalItems)
	}
	if collection.OrderedItems == nil {
		t.Fatal("OrderedItems = nil, want empty slice")
	}

	payload, err := json.Marshal(collection)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(payload), `"orderedItems":[]`) {
		t.Errorf("payload = %s, want empty orderedItems array", payload)
	}
}

func TestNewWebFinger(t *testing.T) {
	document := NewWebFinger(NewURLs("https://cultivar.example"), "cultivar.example", testPerson())

	if document.Subject != "acct:ana@cultivar.example" {
		t.Errorf("Subject = %q", document.Subject)
	}
	if len(document.Aliases) != 2 {
		t.Fatalf("Aliases = %v, want two entries", document.Aliases)
	}
	if len(document.Links) != 2 {
		t.Fatalf("Links = %v, want two entries", document.Links)
	}
	if document.Links[0].Rel != "http://webfinger.net/rel/profile-page" {
		t.Errorf("profile link rel = %q", document.Links[0].Rel)
	}
	if document.Links[1].Rel != "self" {
		t.Errorf("self link rel = %q", document.Links[1].Rel)
	}
	if document.Links[1].Type != "application/activity+json" {
		t.Errorf("self link type = %q", document.Links[1].Type)
	}
	if document.Links[1].Href != "https://cultivar.example/actors/11111111-1111-1111-1111-111111111111" {
		t.Errorf("self link href = %q", document.Links[1].Href)
	}
}

func TestParseAcctResource(t *testing.T) {
	cases := []struct {
		resource string
		handle   string
		host     string
		ok       bool
	}{
		{"acct:ana@localhost:8080", "ana", "localhost:8080", true},
		{"acct:@Ana@LocalHost:8080", "ana", "localhost:8080", true},
		{"ACCT:ana@cultivar.example", "ana", "cultivar.example", true},
		{"acct:ana", "", "", false},
		{"acct:@localhost:8080", "", "", false},
		{"https://localhost:8080/actors/abc", "", "", false},
		{"", "", "", false},
	}

	for _, tc := range cases {
		handle, host, ok := ParseAcctResource(tc.resource)
		if ok != tc.ok || handle != tc.handle || host != tc.host {
			t.Errorf("ParseAcctResource(%q) = (%q, %q, %t), want (%q, %q, %t)",
				tc.resource, handle, host, ok, tc.handle, tc.host, tc.ok)
		}
	}
}

func TestNewNodeInfoIndex(t *testing.T) {
	index := NewNodeInfoIndex(NewURLs("https://cultivar.example"))

	if len(index.Links) != 1 {
		t.Fatalf("Links = %v, want one entry", index.Links)
	}
	if index.Links[0].Rel != NodeInfoSchemaRel {
		t.Errorf("Rel = %q, want %q", index.Links[0].Rel, NodeInfoSchemaRel)
	}
	if index.Links[0].Href != "https://cultivar.example/nodeinfo/2.0" {
		t.Errorf("Href = %q", index.Links[0].Href)
	}
}

func TestNewNodeInfo(t *testing.T) {
	document := NewNodeInfo("Cultivar", "dev", identity.Stats{
		TotalAccounts:  4,
		ActiveMonth:    2,
		ActiveHalfyear: 3,
	})

	if document.Version != NodeInfoVersion {
		t.Errorf("Version = %q, want %q", document.Version, NodeInfoVersion)
	}
	if document.Software.Name != "cultivar" || document.Software.Version != "dev" {
		t.Errorf("Software = %+v", document.Software)
	}
	if len(document.Protocols) != 1 || document.Protocols[0] != "activitypub" {
		t.Errorf("Protocols = %v", document.Protocols)
	}
	if document.Services.Inbound == nil || document.Services.Outbound == nil {
		t.Error("Services arrays must be non-nil")
	}
	if !document.OpenRegistrations {
		t.Error("OpenRegistrations = false, want true")
	}
	if document.Usage.Users.Total != 4 || document.Usage.Users.ActiveMonth != 2 || document.Usage.Users.ActiveHalfyear != 3 {
		t.Errorf("Usage.Users = %+v", document.Usage.Users)
	}
	if document.Metadata.NodeName != "Cultivar" {
		t.Errorf("Metadata.NodeName = %q", document.Metadata.NodeName)
	}
}
