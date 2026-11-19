package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rede-cultivar/plataform/internal/federation"
	"github.com/rede-cultivar/plataform/internal/practices"
)

func practiceFederationRouter(t *testing.T, identityService *fakeIdentity, practiceService *fakePractice, federationService *fakeFederation) http.Handler {
	t.Helper()

	return newContentTestRouterWithFederation(
		t,
		identityService,
		newFakeCultivation(),
		practiceService,
		nil,
		federationService,
	)
}

func TestCreatePublishedPracticeFederates(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	federationService := newFakeFederation(t)
	router := practiceFederationRouter(t, identityService, practiceService, federationService)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/praticas/nova", session)

	response := postForm(router, "/praticas", url.Values{
		"csrf_token": {csrf.Value},
		"title":      {"Adubo de casca de banana"},
		"summary":    {"Adubo simples para hortas domésticas."},
		"difficulty": {"beginner"},
		"materials":  {"Cascas de banana\nÁgua"},
		"steps":      {"Ferva as cascas\nDeixe descansar\nDilua e aplique"},
		"license":    {"CC BY-SA 4.0"},
		"action":     {"publish"},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.practiceCalls != 1 {
		t.Fatalf("practiceCalls = %d, want 1", federationService.practiceCalls)
	}
	if federationService.lastPractice.ID != "practice-1" {
		t.Errorf("federated practice = %q, want practice-1", federationService.lastPractice.ID)
	}
}

func TestCreateDraftPracticeDoesNotFederate(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	federationService := newFakeFederation(t)
	router := practiceFederationRouter(t, identityService, practiceService, federationService)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/praticas/nova", session)

	response := postForm(router, "/praticas", url.Values{
		"csrf_token": {csrf.Value},
		"title":      {"Adubo de casca de banana"},
		"difficulty": {"beginner"},
		"action":     {"save"},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.practiceCalls != 0 {
		t.Errorf("practiceCalls = %d, want 0", federationService.practiceCalls)
	}
}

func TestPublishPracticeFederatesUpdate(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	practiceService.practices["draft"] = samplePractice("draft", practices.StatusDraft, practices.VisibilityPrivate)
	federationService := newFakeFederation(t)
	router := practiceFederationRouter(t, identityService, practiceService, federationService)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/praticas/draft", session)

	response := postForm(router, "/praticas/draft/publicar", url.Values{
		"csrf_token": {csrf.Value},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.practiceUpdateCalls != 1 {
		t.Fatalf("practiceUpdateCalls = %d, want 1", federationService.practiceUpdateCalls)
	}
	if federationService.lastPracticeUpdate.Status != practices.StatusPublished {
		t.Errorf("federated status = %q, want published", federationService.lastPracticeUpdate.Status)
	}
}

func TestEditPublishedPracticeFederatesUpdate(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	practiceService.practices["published"] = samplePractice("published", practices.StatusPublished, practices.VisibilityPublic)
	federationService := newFakeFederation(t)
	router := practiceFederationRouter(t, identityService, practiceService, federationService)
	session := loginAsAna(t, router, identityService)

	formPage := getPage(router, "/praticas/published/editar", session)
	csrf := responseCSRFCookie(formPage)
	if csrf == nil {
		t.Fatal("edit form did not set the CSRF cookie")
	}

	response := postForm(router, "/praticas/published/editar", url.Values{
		"csrf_token": {csrf.Value},
		"title":      {"Adubo de casca de banana"},
		"summary":    {"Adubo simples para hortas domésticas."},
		"difficulty": {"beginner"},
		"materials":  {"Cascas de banana\nÁgua"},
		"steps":      {"Ferva as cascas\nDeixe descansar\nDilua e aplique"},
		"license":    {"CC BY-SA 4.0"},
		"action":     {"save"},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.practiceUpdateCalls != 1 {
		t.Fatalf("practiceUpdateCalls = %d, want 1", federationService.practiceUpdateCalls)
	}
	if federationService.practiceDeleteCalls != 0 {
		t.Errorf("practiceDeleteCalls = %d, want 0", federationService.practiceDeleteCalls)
	}
}

func TestArchivePracticeFederatesDelete(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	practiceService.practices["published"] = samplePractice("published", practices.StatusPublished, practices.VisibilityPublic)
	federationService := newFakeFederation(t)
	router := practiceFederationRouter(t, identityService, practiceService, federationService)
	session := loginAsAna(t, router, identityService)
	csrf := getCSRFCookie(t, router, "/praticas/published")

	response := postForm(router, "/praticas/published/arquivar", url.Values{
		"csrf_token": {csrf.Value},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.practiceDeleteCalls != 1 {
		t.Fatalf("practiceDeleteCalls = %d, want 1", federationService.practiceDeleteCalls)
	}
	if federationService.lastPracticeDelete.ID != "published" {
		t.Errorf("deleted practice = %q, want published", federationService.lastPracticeDelete.ID)
	}
}

func TestArticleEndpointServesPublishedPractice(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	practiceService.practices["published"] = samplePractice("published", practices.StatusPublished, practices.VisibilityPublic)
	router := practiceFederationRouter(t, identityService, practiceService, newFakeFederation(t))
	loginAsAna(t, router, identityService)

	response := getPage(router, "/artigos/published")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "activity+json") {
		t.Errorf("Content-Type = %q, want activity+json", contentType)
	}

	body := response.Body.String()
	for _, want := range []string{
		`"type":"Article"`,
		`"name":"Adubo de casca de banana"`,
		`/artigos/published`,
		`/praticas/published`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body = %s, want substring %q", body, want)
		}
	}
}

func TestArticleEndpointHidesUnpublishedPractice(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	practiceService.practices["draft"] = samplePractice("draft", practices.StatusDraft, practices.VisibilityPrivate)
	practiceService.practices["archived"] = samplePractice("archived", practices.StatusArchived, practices.VisibilityPublic)
	router := practiceFederationRouter(t, identityService, practiceService, newFakeFederation(t))
	loginAsAna(t, router, identityService)

	for _, id := range []string{"draft", "archived", "missing"} {
		response := getPage(router, "/artigos/"+id)
		if response.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want %d", id, response.Code, http.StatusNotFound)
		}
	}
}

func TestPracticePageShowsFederatedInteractions(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	practiceService.practices["published"] = samplePractice("published", practices.StatusPublished, practices.VisibilityPublic)

	publishedAt := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	federationService := newFakeFederation(t)
	federationService.practiceInteractions = map[string]federation.EntryInteractions{
		"published": {
			Replies: []federation.Interaction{{
				Kind:        federation.InteractionReply,
				ActorName:   "Bruno",
				ActorHandle: "bruno",
				ActorURL:    "https://remote.example/@bruno",
				Content:     "Funcionou bem aqui.",
				PublishedAt: &publishedAt,
			}},
			Likes: []federation.Interaction{{
				Kind:        federation.InteractionLike,
				ActorName:   "Carla",
				ActorHandle: "carla",
				ActorURL:    "https://remote.example/@carla",
			}},
		},
	}

	router := practiceFederationRouter(t, identityService, practiceService, federationService)
	response := getPage(router, "/praticas/published")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	body := response.Body.String()
	for _, want := range []string{
		"Interações no Fediverso",
		"1 resposta",
		"1 curtida",
		"Bruno",
		"Funcionou bem aqui.",
		"Carla",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %q", want)
		}
	}
}

func TestPracticePageHidesInteractionsForDraft(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	practiceService.practices["draft"] = samplePractice("draft", practices.StatusDraft, practices.VisibilityPrivate)

	federationService := newFakeFederation(t)
	federationService.practiceInteractions = map[string]federation.EntryInteractions{
		"draft": {
			Replies: []federation.Interaction{{
				Kind:      federation.InteractionReply,
				ActorName: "Bruno",
				Content:   "Comentário inesperado.",
			}},
		},
	}

	router := practiceFederationRouter(t, identityService, practiceService, federationService)
	session := loginAsAna(t, router, identityService)
	response := getPage(router, "/praticas/draft", session)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if body := response.Body.String(); strings.Contains(body, "Interações no Fediverso") {
		t.Error("draft page shows federated interactions")
	}
}
