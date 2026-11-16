package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rede-cultivar/plataform/internal/cultivation"
	"github.com/rede-cultivar/plataform/internal/federation"
	"github.com/rede-cultivar/plataform/internal/identity"
	"github.com/rede-cultivar/plataform/internal/media"
	"github.com/rede-cultivar/plataform/internal/platform/config"
	"github.com/rede-cultivar/plataform/internal/sitesettings"
)

// fakeFederation implementa FederationService em memória para os testes HTTP.
type fakeFederation struct {
	base        string
	key         *federation.KeyPair
	outbox      federation.OrderedCollection
	followers   federation.OrderedCollection
	profiles    []federation.Follower
	profilesErr error
	activities  map[string]json.RawMessage
	inboxErr    error
	inboxCalls  int
}

func newFakeFederation(t *testing.T) *fakeFederation {
	t.Helper()

	key, err := federation.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	return &fakeFederation{
		base:       "http://localhost:8080",
		key:        key,
		activities: make(map[string]json.RawMessage),
	}
}

func (f *fakeFederation) ActorKey(_ context.Context, _ string) (*federation.KeyPair, error) {
	return f.key, nil
}

func (f *fakeFederation) PublishEntry(_ context.Context, _ identity.Person, _ cultivation.Cycle, _ cultivation.Entry, _ []media.Asset) error {
	return nil
}

func (f *fakeFederation) Outbox(_ context.Context, personID string) (federation.OrderedCollection, error) {
	if f.outbox.ID != "" {
		return f.outbox, nil
	}
	return federation.NewOrderedCollection(federation.NewURLs(f.base).Outbox(personID)), nil
}

func (f *fakeFederation) Followers(_ context.Context, personID string) (federation.OrderedCollection, error) {
	if f.followers.ID != "" {
		return f.followers, nil
	}
	return federation.NewOrderedCollection(federation.NewURLs(f.base).Followers(personID)), nil
}

func (f *fakeFederation) FollowerProfiles(_ context.Context, _ string) ([]federation.Follower, error) {
	if f.profilesErr != nil {
		return nil, f.profilesErr
	}
	return f.profiles, nil
}

func (f *fakeFederation) HandleInbox(_ context.Context, _ string, _ *http.Request) error {
	f.inboxCalls++
	return f.inboxErr
}

func (f *fakeFederation) FindActivity(_ context.Context, activityID string) (json.RawMessage, error) {
	payload, ok := f.activities[activityID]
	if !ok {
		return nil, federation.ErrNotFound
	}
	return payload, nil
}

func newFederationTestRouterWithService(t *testing.T) (http.Handler, *fakeIdentity, *fakeFederation) {
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

	service := newFakeFederation(t)
	return newAuthTestRouter(t, fake, service), fake, service
}

func newFederationTestRouter(t *testing.T) (http.Handler, *fakeIdentity) {
	t.Helper()

	router, fake, _ := newFederationTestRouterWithService(t)
	return router, fake
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
	if actor.PublicKey == nil {
		t.Fatal("PublicKey = nil, want actor key")
	}
	if actor.PublicKey.ID != "http://localhost:8080/actors/person-1#main-key" {
		t.Errorf("PublicKey.ID = %q", actor.PublicKey.ID)
	}
	if actor.PublicKey.Owner != actor.ID {
		t.Errorf("PublicKey.Owner = %q, want %q", actor.PublicKey.Owner, actor.ID)
	}
	if !strings.Contains(actor.PublicKey.PublicKeyPEM, "BEGIN PUBLIC KEY") {
		t.Errorf("PublicKey.PublicKeyPEM = %q, want PEM", actor.PublicKey.PublicKeyPEM)
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

func TestInboxDelegatesToFederation(t *testing.T) {
	router, _, service := newFederationTestRouterWithService(t)

	request := httptest.NewRequest(http.MethodPost, "/actors/person-1/inbox", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusAccepted, response.Body.String())
	}
	if service.inboxCalls != 1 {
		t.Errorf("inbox calls = %d, want 1", service.inboxCalls)
	}
}

func TestInboxRejectsInvalidSignature(t *testing.T) {
	router, _, service := newFederationTestRouterWithService(t)
	service.inboxErr = federation.ErrInvalidSignature

	request := httptest.NewRequest(http.MethodPost, "/actors/person-1/inbox", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestInboxRequiresFederationService(t *testing.T) {
	fake := newFakeIdentity()
	fake.people["ana"] = &identity.Person{ID: "person-1", Handle: "ana"}
	router := newAuthTestRouter(t, fake)

	request := httptest.NewRequest(http.MethodPost, "/actors/person-1/inbox", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func newFederationContentRouter(
	t *testing.T,
	identityService IdentityService,
	cultivationService CultivationService,
	mediaService MediaService,
	federationService FederationService,
) http.Handler {
	t.Helper()

	return NewRouter(Deps{
		Config: &config.Config{
			Env:                    config.EnvDevelopment,
			InstanceName:           "Cultivar Test",
			LocalDomain:            "localhost:8080",
			LogLevel:               "error",
			SessionIdleTimeout:     time.Hour,
			SessionAbsoluteTimeout: 24 * time.Hour,
		},
		Identity:    identityService,
		Cultivation: cultivationService,
		Media:       mediaService,
		Federation:  federationService,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func TestNotePublishesPublicEntry(t *testing.T) {
	identityService := newFakeIdentity()
	identityService.people["ana"] = &identity.Person{
		ID:          "person-1",
		Handle:      "ana",
		DisplayName: "Ana Silva",
		CreatedAt:   time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC),
	}

	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:         "entry-1",
		CycleID:    "cycle-1",
		AuthorID:   "person-1",
		OccurredAt: time.Date(2026, 9, 10, 9, 30, 0, 0, time.UTC),
		Body:       "Primeira flor observada.",
		Visibility: cultivation.EntryVisibilityInherit,
		CreatedAt:  time.Date(2026, 9, 10, 9, 30, 0, 0, time.UTC),
	}}

	mediaService := newFakeMedia()
	mediaService.assets["asset-1"] = &media.Asset{
		ID:          "asset-1",
		OwnerID:     "person-1",
		ContentType: "image/png",
		Width:       1,
		Height:      1,
		Status:      media.StatusReady,
	}
	mediaService.attachments["entry-1"] = []string{"asset-1"}

	router := newFederationContentRouter(t, identityService, cultivationService, mediaService, newFakeFederation(t))

	request := httptest.NewRequest(http.MethodGet, "/notas/entry-1", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/activity+json") {
		t.Errorf("Content-Type = %q, want application/activity+json", contentType)
	}

	var note federation.Note
	decodeJSON(t, response, &note)

	if note.ID != "http://localhost:8080/notas/entry-1" {
		t.Errorf("ID = %q", note.ID)
	}
	if note.AttributedTo != "http://localhost:8080/actors/person-1" {
		t.Errorf("AttributedTo = %q", note.AttributedTo)
	}
	if !strings.Contains(note.Content, "Primeira flor observada.") {
		t.Errorf("Content = %q", note.Content)
	}
	if len(note.Attachment) != 1 {
		t.Fatalf("Attachment = %v, want one image", note.Attachment)
	}
	if note.Attachment[0].URL != "http://localhost:8080/media/asset-1" {
		t.Errorf("Attachment URL = %q", note.Attachment[0].URL)
	}
}

func TestNoteHidesPrivateEntry(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPrivate)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:         "entry-1",
		CycleID:    "cycle-1",
		AuthorID:   "person-1",
		OccurredAt: time.Date(2026, 9, 10, 9, 30, 0, 0, time.UTC),
		Body:       "Registro privado.",
		Visibility: cultivation.EntryVisibilityInherit,
	}}

	router := newFederationContentRouter(t, identityService, cultivationService, newFakeMedia(), newFakeFederation(t))

	request := httptest.NewRequest(http.MethodGet, "/notas/entry-1", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestActivityServesStoredPayload(t *testing.T) {
	router, _, service := newFederationTestRouterWithService(t)
	service.activities["http://localhost:8080/activities/abc"] = json.RawMessage(`{"id":"http://localhost:8080/activities/abc","type":"Create"}`)

	request := httptest.NewRequest(http.MethodGet, "/activities/abc", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if body := response.Body.String(); !strings.Contains(body, `"type":"Create"`) {
		t.Errorf("body = %s, want stored activity", body)
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

// authenticateFake cria uma sessão válida para a pessoa "ana" nos testes.
func TestNodeInfoReportsClosedRegistrations(t *testing.T) {
	router := newAuthTestRouterWithSettings(t, newFakeIdentity(), &fakeSiteSettings{
		settings: &sitesettings.Settings{RegistrationsMode: sitesettings.ModeNone},
	})
	request := httptest.NewRequest(http.MethodGet, "/nodeinfo/2.0", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var document federation.NodeInfo
	decodeJSON(t, response, &document)
	if document.OpenRegistrations {
		t.Error("OpenRegistrations = true, want false for closed signups")
	}
}

// authenticateFake cria uma sessão válida para a pessoa "ana" nos testes.
func authenticateFake(fake *fakeIdentity) *http.Cookie {
	person := *fake.people["ana"]
	account := &identity.Account{
		ID:        "account-1",
		Email:     "ana@example.com",
		Status:    "active",
		CreatedAt: time.Now(),
		Person:    person,
	}
	fake.sessions["token-account-1"] = account

	return &http.Cookie{Name: sessionCookieName, Value: "token-account-1"}
}

func TestAccountFollowersListsRemoteProfiles(t *testing.T) {
	router, fake, service := newFederationTestRouterWithService(t)
	service.profiles = []federation.Follower{
		{
			ActorID:     "https://remote.example/users/bob",
			Handle:      "bob",
			Domain:      "remote.example",
			DisplayName: "Bob Remoto",
			ProfileURL:  "https://remote.example/@bob",
			AvatarURL:   "https://remote.example/avatars/bob.png",
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/conta/seguidores", nil)
	request.AddCookie(authenticateFake(fake))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	body := response.Body.String()
	for _, expected := range []string{
		"Seguidores",
		"Bob Remoto",
		"@bob@remote.example",
		"https://remote.example/@bob",
		"https://remote.example/avatars/bob.png",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("body does not contain %q", expected)
		}
	}
}

func TestAccountFollowersEmptyState(t *testing.T) {
	router, fake, _ := newFederationTestRouterWithService(t)

	request := httptest.NewRequest(http.MethodGet, "/conta/seguidores", nil)
	request.AddCookie(authenticateFake(fake))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if body := response.Body.String(); !strings.Contains(body, "Nenhum seguidor ainda.") {
		t.Errorf("body does not contain the empty state")
	}
}

func TestAccountFollowersRequiresLogin(t *testing.T) {
	router, _, _ := newFederationTestRouterWithService(t)

	request := httptest.NewRequest(http.MethodGet, "/conta/seguidores", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	if location := response.Header().Get("Location"); location != "/entrar?next=%2Fconta%2Fseguidores" {
		t.Errorf("Location = %q", location)
	}
}

func TestAccountFollowersReportsServiceError(t *testing.T) {
	router, fake, service := newFederationTestRouterWithService(t)
	service.profilesErr = errors.New("falha remota")

	request := httptest.NewRequest(http.MethodGet, "/conta/seguidores", nil)
	request.AddCookie(authenticateFake(fake))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if body := response.Body.String(); !strings.Contains(body, "Não foi possível carregar os seguidores.") {
		t.Errorf("body does not contain the error message")
	}
}
