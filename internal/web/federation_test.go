package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rede-cultivar/plataform/internal/federation"
	"github.com/rede-cultivar/plataform/internal/identity"
)

func newFederationTestRouter(t *testing.T) (http.Handler, *fakeIdentity) {
	t.Helper()

	fake := newFakeIdentity()
	person := identity.Person{
		ID:          "person-1",
		Handle:      "ana",
		DisplayName: "Ana Silva",
		Bio:         "Cultivo hortas.",
		CreatedAt:   time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC),
	}
	fake.people["ana"] = &person

	return newAuthTestRouter(t, fake), fake
}

func decodeJSON(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode JSON: %v; body = %s", err, response.Body.String())
	}
}

func webfingerTarget(resource string) string {
	return "/.well-known/webfinger?resource=" + url.QueryEscape(resource)
}

func TestWebFingerResolvesLocalAccount(t *testing.T) {
	router, _ := newFederationTestRouter(t)

	request := httptest.NewRequest(http.MethodGet, webfingerTarget("acct:ana@localhost:8080"), nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/jrd+json") {
		t.Errorf("Content-Type = %q, want application/jrd+json", contentType)
	}
	if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", origin)
	}

	var document federation.WebFinger
	decodeJSON(t, response, &document)

	if document.Subject != "acct:ana@localhost:8080" {
		t.Errorf("Subject = %q", document.Subject)
	}

	var self *federation.WebFingerLink
	for i := range document.Links {
		if document.Links[i].Rel == "self" {
			self = &document.Links[i]
		}
	}
	if self == nil {
		t.Fatal("WebFinger response has no self link")
	}
	if self.Type != "application/activity+json" {
		t.Errorf("self link type = %q", self.Type)
	}
	if self.Href != "http://localhost:8080/actors/person-1" {
		t.Errorf("self link href = %q", self.Href)
	}
}

func TestWebFingerRejectsInvalidRequests(t *testing.T) {
	router, _ := newFederationTestRouter(t)

	cases := []struct {
		target string
		status int
	}{
		{"/.well-known/webfinger", http.StatusBadRequest},
		{webfingerTarget("https://example.com/actors/abc"), http.StatusBadRequest},
		{webfingerTarget("acct:ana@example.com"), http.StatusNotFound},
		{webfingerTarget("acct:desconhecido@localhost:8080"), http.StatusNotFound},
	}

	for _, tc := range cases {
		request := httptest.NewRequest(http.MethodGet, tc.target, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != tc.status {
			t.Errorf("GET %s status = %d, want %d", tc.target, response.Code, tc.status)
		}
	}
}

func TestActorPublishesPerson(t *testing.T) {
	router, _ := newFederationTestRouter(t)

	request := httptest.NewRequest(http.MethodGet, "/actors/person-1", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/activity+json") {
		t.Errorf("Content-Type = %q, want application/activity+json", contentType)
	}

	var actor federation.PersonActor
	decodeJSON(t, response, &actor)

	if actor.ID != "http://localhost:8080/actors/person-1" {
		t.Errorf("ID = %q", actor.ID)
	}
	if actor.Type != "Person" {
		t.Errorf("Type = %q, want Person", actor.Type)
	}
	if actor.PreferredUsername != "ana" {
		t.Errorf("PreferredUsername = %q, want ana", actor.PreferredUsername)
	}
	if actor.Name != "Ana Silva" {
		t.Errorf("Name = %q", actor.Name)
	}
	if actor.Summary != "<p>Cultivo hortas.</p>" {
		t.Errorf("Summary = %q", actor.Summary)
	}
	if actor.Inbox != "http://localhost:8080/actors/person-1/inbox" {
		t.Errorf("Inbox = %q", actor.Inbox)
	}
	if actor.Published != "2026-11-05T12:00:00Z" {
		t.Errorf("Published = %q", actor.Published)
	}
}

func TestActorNotFound(t *testing.T) {
	router, _ := newFederationTestRouter(t)

	request := httptest.NewRequest(http.MethodGet, "/actors/desconhecido", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestActorCollectionsAreEmpty(t *testing.T) {
	router, _ := newFederationTestRouter(t)

	for _, name := range []string{"outbox", "followers", "following"} {
		request := httptest.NewRequest(http.MethodGet, "/actors/person-1/"+name, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Errorf("GET /actors/person-1/%s status = %d, want %d", name, response.Code, http.StatusOK)
			continue
		}

		var collection federation.OrderedCollection
		decodeJSON(t, response, &collection)

		if collection.ID != "http://localhost:8080/actors/person-1/"+name {
			t.Errorf("%s ID = %q", name, collection.ID)
		}
		if collection.Type != "OrderedCollection" {
			t.Errorf("%s Type = %q", name, collection.Type)
		}
		if collection.TotalItems != 0 || len(collection.OrderedItems) != 0 {
			t.Errorf("%s is not empty: %+v", name, collection)
		}
	}
}

func TestInboxReturnsNotImplemented(t *testing.T) {
	router, _ := newFederationTestRouter(t)

	request := httptest.NewRequest(http.MethodPost, "/actors/person-1/inbox", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotImplemented)
	}
}

func TestNodeInfoDiscovery(t *testing.T) {
	router, _ := newFederationTestRouter(t)

	request := httptest.NewRequest(http.MethodGet, "/.well-known/nodeinfo", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/jrd+json") {
		t.Errorf("Content-Type = %q, want application/jrd+json", contentType)
	}

	var index federation.NodeInfoIndex
	decodeJSON(t, response, &index)

	if len(index.Links) != 1 {
		t.Fatalf("Links = %v, want one entry", index.Links)
	}
	if index.Links[0].Rel != federation.NodeInfoSchemaRel {
		t.Errorf("Rel = %q", index.Links[0].Rel)
	}
	if index.Links[0].Href != "http://localhost:8080/nodeinfo/2.0" {
		t.Errorf("Href = %q", index.Links[0].Href)
	}
}

func TestNodeInfoDocument(t *testing.T) {
	router, fake := newFederationTestRouter(t)
	fake.stats = identity.Stats{TotalAccounts: 4, ActiveMonth: 2, ActiveHalfyear: 3}

	request := httptest.NewRequest(http.MethodGet, "/nodeinfo/2.0", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); contentType != federation.NodeInfoContentType {
		t.Errorf("Content-Type = %q, want %q", contentType, federation.NodeInfoContentType)
	}

	var document federation.NodeInfo
	decodeJSON(t, response, &document)

	if document.Version != "2.0" {
		t.Errorf("Version = %q, want 2.0", document.Version)
	}
	if document.Software.Name != "cultivar" {
		t.Errorf("Software.Name = %q, want cultivar", document.Software.Name)
	}
	if len(document.Protocols) != 1 || document.Protocols[0] != "activitypub" {
		t.Errorf("Protocols = %v", document.Protocols)
	}
	if !document.OpenRegistrations {
		t.Error("OpenRegistrations = false, want true")
	}
	if document.Usage.Users.Total != 4 || document.Usage.Users.ActiveMonth != 2 || document.Usage.Users.ActiveHalfyear != 3 {
		t.Errorf("Usage.Users = %+v", document.Usage.Users)
	}
	if document.Metadata.NodeName != "Cultivar Test" {
		t.Errorf("Metadata.NodeName = %q, want Cultivar Test", document.Metadata.NodeName)
	}
}
