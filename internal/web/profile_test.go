package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rede-cultivar/plataform/internal/identity"
)

func loginAsAna(t *testing.T, router http.Handler, fake *fakeIdentity) *http.Cookie {
	t.Helper()

	person := identity.Person{
		ID:          "person-1",
		Handle:      "ana",
		DisplayName: "Ana Silva",
		Bio:         "Biografia antiga.",
		CreatedAt:   time.Now(),
	}
	account := &identity.Account{
		ID:        "account-1",
		Email:     "ana@example.com",
		Status:    "active",
		CreatedAt: time.Now(),
		Person:    person,
	}
	fake.accounts[account.Email] = account
	fake.people[person.Handle] = &person

	csrf := getCSRFCookie(t, router, "/entrar")
	response := postForm(router, "/entrar", url.Values{
		"csrf_token": {csrf.Value},
		"email":      {account.Email},
		"password":   {testPassword},
	}, csrf)

	cookie := sessionCookie(response)
	if cookie == nil {
		t.Fatalf("login failed: status = %d; body = %s", response.Code, response.Body.String())
	}

	return cookie
}

func responseCSRFCookie(response *httptest.ResponseRecorder) *http.Cookie {
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == csrfCookieName {
			return cookie
		}
	}
	return nil
}

func openProfileEditForm(t *testing.T, router http.Handler, session *http.Cookie) (*httptest.ResponseRecorder, *http.Cookie) {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/conta/perfil", nil)
	request.AddCookie(session)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /conta/perfil status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	csrf := responseCSRFCookie(response)
	if csrf == nil {
		t.Fatal("GET /conta/perfil did not set the CSRF cookie")
	}

	return response, csrf
}

func TestProfileEditFormShowsCurrentProfile(t *testing.T) {
	fake := newFakeIdentity()
	router := newAuthTestRouter(t, fake)
	session := loginAsAna(t, router, fake)

	response, _ := openProfileEditForm(t, router, session)
	body := response.Body.String()

	if !strings.Contains(body, `value="Ana Silva"`) {
		t.Error("profile edit form does not contain the current display name")
	}
	if !strings.Contains(body, "Biografia antiga.") {
		t.Error("profile edit form does not contain the current bio")
	}
}

func TestProfileEditUpdatesProfile(t *testing.T) {
	fake := newFakeIdentity()
	router := newAuthTestRouter(t, fake)
	session := loginAsAna(t, router, fake)
	_, csrf := openProfileEditForm(t, router, session)

	response := postForm(router, "/conta/perfil", url.Values{
		"csrf_token":   {csrf.Value},
		"display_name": {"Ana Oliveira"},
		"bio":          {"Cultivo hortas e compartilho sementes."},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/conta/perfil?atualizado=1" {
		t.Fatalf("Location = %q, want /conta/perfil?atualizado=1", location)
	}

	updatedRequest := httptest.NewRequest(http.MethodGet, "/conta/perfil?atualizado=1", nil)
	updatedRequest.AddCookie(session)
	updated := httptest.NewRecorder()
	router.ServeHTTP(updated, updatedRequest)

	if updated.Code != http.StatusOK {
		t.Fatalf("GET /conta/perfil?atualizado=1 status = %d, want %d", updated.Code, http.StatusOK)
	}

	body := updated.Body.String()
	if !strings.Contains(body, "Perfil atualizado.") {
		t.Error("updated page does not contain the success notice")
	}
	if !strings.Contains(body, "Ana Oliveira") {
		t.Error("updated page does not contain the new display name")
	}
	if !strings.Contains(body, "Cultivo hortas e compartilho sementes.") {
		t.Error("updated page does not contain the new bio")
	}

	if fake.people["ana"].DisplayName != "Ana Oliveira" {
		t.Errorf("stored display name = %q", fake.people["ana"].DisplayName)
	}
	if fake.people["ana"].Bio != "Cultivo hortas e compartilho sementes." {
		t.Errorf("stored bio = %q", fake.people["ana"].Bio)
	}
}

func TestProfileEditReportsValidationError(t *testing.T) {
	fake := newFakeIdentity()
	router := newAuthTestRouter(t, fake)
	session := loginAsAna(t, router, fake)
	_, csrf := openProfileEditForm(t, router, session)

	fake.updateErr = identity.ErrInvalidBio

	response := postForm(router, "/conta/perfil", url.Values{
		"csrf_token":   {csrf.Value},
		"display_name": {"Ana Silva"},
		"bio":          {strings.Repeat("a", 501)},
	}, session, csrf)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if body := response.Body.String(); !strings.Contains(body, "A biografia deve ter no máximo 500 caracteres.") {
		t.Error("response does not contain the bio validation message")
	}
}

func TestProfileEditRequiresCSRF(t *testing.T) {
	fake := newFakeIdentity()
	router := newAuthTestRouter(t, fake)
	session := loginAsAna(t, router, fake)

	response := postForm(router, "/conta/perfil", url.Values{
		"display_name": {"Ana Silva"},
		"bio":          {"x"},
	}, session)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestProfileEditRequiresAuthentication(t *testing.T) {
	router := newAuthTestRouter(t, newFakeIdentity())

	request := httptest.NewRequest(http.MethodGet, "/conta/perfil", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusSeeOther)
	}

	location := response.Header().Get("Location")
	if !strings.HasPrefix(location, "/entrar") {
		t.Errorf("Location = %q, want /entrar", location)
	}
	if !strings.Contains(location, "next=") {
		t.Errorf("Location = %q, want next parameter", location)
	}
}
