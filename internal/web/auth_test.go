package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rede-cultivar/plataform/internal/identity"
	"github.com/rede-cultivar/plataform/internal/platform/config"
)

const testPassword = "senha-correta-123"

type fakeIdentity struct {
	accounts    map[string]*identity.Account
	people      map[string]*identity.Person
	sessions    map[string]*identity.Account
	registerErr error
	authErr     error
	nextID      int
}

func newFakeIdentity() *fakeIdentity {
	return &fakeIdentity{
		accounts: make(map[string]*identity.Account),
		people:   make(map[string]*identity.Person),
		sessions: make(map[string]*identity.Account),
	}
}

func (f *fakeIdentity) Register(_ context.Context, input identity.RegisterInput) (*identity.Account, error) {
	if f.registerErr != nil {
		return nil, f.registerErr
	}

	f.nextID++
	person := identity.Person{
		ID:          fmt.Sprintf("person-%d", f.nextID),
		Handle:      input.Handle,
		DisplayName: input.DisplayName,
		CreatedAt:   time.Now(),
	}
	account := &identity.Account{
		ID:        fmt.Sprintf("account-%d", f.nextID),
		Email:     input.Email,
		Status:    "active",
		CreatedAt: time.Now(),
		Person:    person,
	}

	f.accounts[input.Email] = account
	f.people[input.Handle] = &person

	return account, nil
}

func (f *fakeIdentity) Authenticate(_ context.Context, email, password string) (*identity.Account, error) {
	if f.authErr != nil {
		return nil, f.authErr
	}

	account, ok := f.accounts[email]
	if !ok || password != testPassword {
		return nil, identity.ErrInvalidCredentials
	}

	return account, nil
}

func (f *fakeIdentity) StartSession(_ context.Context, accountID, _, _ string) (string, error) {
	for _, account := range f.accounts {
		if account.ID == accountID {
			token := "token-" + accountID
			f.sessions[token] = account
			return token, nil
		}
	}

	return "", errors.New("account not found")
}

func (f *fakeIdentity) AuthenticateSession(_ context.Context, token string) (*identity.Account, error) {
	account, ok := f.sessions[token]
	if !ok {
		return nil, identity.ErrSessionExpired
	}
	return account, nil
}

func (f *fakeIdentity) EndSession(_ context.Context, token string) error {
	delete(f.sessions, token)
	return nil
}

func (f *fakeIdentity) FindPersonByHandle(_ context.Context, handle string) (*identity.Person, error) {
	person, ok := f.people[handle]
	if !ok {
		return nil, identity.ErrNotFound
	}
	return person, nil
}

func newAuthTestRouter(t *testing.T, identityService IdentityService) http.Handler {
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
		Identity: identityService,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func getCSRFCookie(t *testing.T, router http.Handler, target string) *http.Cookie {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, target, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == csrfCookieName {
			return cookie
		}
	}

	t.Fatalf("GET %s did not set the CSRF cookie", target)
	return nil
}

func postForm(router http.Handler, target string, values url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}

func sessionCookie(response *httptest.ResponseRecorder) *http.Cookie {
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			return cookie
		}
	}
	return nil
}

func TestRegisterCreatesSessionAndShowsAccount(t *testing.T) {
	fake := newFakeIdentity()
	router := newAuthTestRouter(t, fake)
	csrf := getCSRFCookie(t, router, "/cadastro")

	response := postForm(router, "/cadastro", url.Values{
		"csrf_token":       {csrf.Value},
		"display_name":     {"Ana Silva"},
		"handle":           {"ana"},
		"email":            {"ana@example.com"},
		"password":         {testPassword},
		"password_confirm": {testPassword},
	}, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/conta" {
		t.Fatalf("Location = %q, want %q", location, "/conta")
	}

	cookie := sessionCookie(response)
	if cookie == nil {
		t.Fatal("registration did not set the session cookie")
	}
	if !cookie.HttpOnly {
		t.Error("session cookie is not HttpOnly")
	}

	request := httptest.NewRequest(http.MethodGet, "/conta", nil)
	request.AddCookie(cookie)
	accountPage := httptest.NewRecorder()
	router.ServeHTTP(accountPage, request)

	if accountPage.Code != http.StatusOK {
		t.Fatalf("GET /conta status = %d, want %d", accountPage.Code, http.StatusOK)
	}
	if body := accountPage.Body.String(); !strings.Contains(body, "ana@example.com") {
		t.Errorf("GET /conta body does not contain the account e-mail")
	}
}

func TestLoginAuthenticatesExistingAccount(t *testing.T) {
	fake := newFakeIdentity()
	person := identity.Person{ID: "person-1", Handle: "ana", DisplayName: "Ana Silva", CreatedAt: time.Now()}
	fake.accounts["ana@example.com"] = &identity.Account{
		ID:        "account-1",
		Email:     "ana@example.com",
		Status:    "active",
		CreatedAt: time.Now(),
		Person:    person,
	}
	fake.people["ana"] = &person

	router := newAuthTestRouter(t, fake)
	csrf := getCSRFCookie(t, router, "/entrar")

	response := postForm(router, "/entrar", url.Values{
		"csrf_token": {csrf.Value},
		"email":      {"ana@example.com"},
		"password":   {testPassword},
		"next":       {"/conta"},
	}, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/conta" {
		t.Fatalf("Location = %q, want %q", location, "/conta")
	}
	if sessionCookie(response) == nil {
		t.Fatal("login did not set the session cookie")
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	router := newAuthTestRouter(t, newFakeIdentity())
	csrf := getCSRFCookie(t, router, "/entrar")

	response := postForm(router, "/entrar", url.Values{
		"csrf_token": {csrf.Value},
		"email":      {"ana@example.com"},
		"password":   {"senha-errada-123"},
	}, csrf)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if body := response.Body.String(); !strings.Contains(body, "E-mail ou senha inválidos") {
		t.Errorf("response does not contain the generic credentials message")
	}
	if sessionCookie(response) != nil {
		t.Error("failed login set a session cookie")
	}
}

func TestLoginRequiresCSRF(t *testing.T) {
	router := newAuthTestRouter(t, newFakeIdentity())

	response := postForm(router, "/entrar", url.Values{
		"email":    {"ana@example.com"},
		"password": {testPassword},
	})

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	fake := newFakeIdentity()
	person := identity.Person{ID: "person-1", Handle: "ana", DisplayName: "Ana Silva", CreatedAt: time.Now()}
	account := &identity.Account{ID: "account-1", Email: "ana@example.com", Status: "active", Person: person}
	fake.accounts[account.Email] = account
	fake.people[person.Handle] = &person

	router := newAuthTestRouter(t, fake)
	csrf := getCSRFCookie(t, router, "/entrar")

	login := postForm(router, "/entrar", url.Values{
		"csrf_token": {csrf.Value},
		"email":      {account.Email},
		"password":   {testPassword},
	}, csrf)
	cookie := sessionCookie(login)
	if cookie == nil {
		t.Fatal("login did not set the session cookie")
	}

	logout := postForm(router, "/sair", url.Values{
		"csrf_token": {csrf.Value},
	}, cookie, csrf)

	if logout.Code != http.StatusSeeOther {
		t.Fatalf("logout status = %d, want %d", logout.Code, http.StatusSeeOther)
	}
	if _, ok := fake.sessions[cookie.Value]; ok {
		t.Error("logout did not revoke the session")
	}
}

func TestProfileShowsPerson(t *testing.T) {
	fake := newFakeIdentity()
	fake.people["ana"] = &identity.Person{
		ID:          "person-1",
		Handle:      "ana",
		DisplayName: "Ana Silva",
		Bio:         "Cultivo hortas comunitárias.",
		CreatedAt:   time.Now(),
	}
	router := newAuthTestRouter(t, fake)

	request := httptest.NewRequest(http.MethodGet, "/perfil/ana", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if body := response.Body.String(); !strings.Contains(body, "Ana Silva") {
		t.Error("profile page does not contain the display name")
	}
}

func TestProfileNotFound(t *testing.T) {
	router := newAuthTestRouter(t, newFakeIdentity())

	request := httptest.NewRequest(http.MethodGet, "/perfil/desconhecido", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestAccountRequiresAuthentication(t *testing.T) {
	router := newAuthTestRouter(t, newFakeIdentity())

	request := httptest.NewRequest(http.MethodGet, "/conta", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	if location := response.Header().Get("Location"); !strings.HasPrefix(location, "/entrar") {
		t.Fatalf("Location = %q, want /entrar", location)
	}
}

func TestSafeNext(t *testing.T) {
	cases := map[string]string{
		"":                 "/conta",
		"/conta":           "/conta",
		"/perfil/ana":      "/perfil/ana",
		"//evil.example":   "/conta",
		"/\\evil.example":  "/conta",
		"https://evil.com": "/conta",
		"perfil":           "/conta",
	}

	for next, want := range cases {
		if got := safeNext(next); got != want {
			t.Errorf("safeNext(%q) = %q, want %q", next, got, want)
		}
	}
}
