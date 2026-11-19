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
	"github.com/rede-cultivar/plataform/internal/sitesettings"
)

const testPassword = "senha-correta-123"

type fakeIdentity struct {
	accounts    map[string]*identity.Account
	people      map[string]*identity.Person
	sessions    map[string]*identity.Account
	follows     map[string]bool
	registerErr error
	authErr     error
	updateErr   error
	stats       identity.Stats
	nextID      int
}

func newFakeIdentity() *fakeIdentity {
	return &fakeIdentity{
		accounts: make(map[string]*identity.Account),
		people:   make(map[string]*identity.Person),
		sessions: make(map[string]*identity.Account),
		follows:  make(map[string]bool),
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
		Status:    input.Status,
		Role:      identity.RoleMember,
		CreatedAt: time.Now(),
		Person:    person,
	}
	if account.Status == "" {
		account.Status = identity.StatusActive
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
	if account.Status == identity.StatusPending {
		return nil, identity.ErrAccountPending
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

func (f *fakeIdentity) FindPersonByID(_ context.Context, id string) (*identity.Person, error) {
	for _, person := range f.people {
		if person.ID == id {
			return person, nil
		}
	}
	return nil, identity.ErrNotFound
}

func (f *fakeIdentity) Follow(_ context.Context, followerID, followedID string) error {
	if followerID == followedID {
		return identity.ErrCannotFollowSelf
	}
	if _, err := f.FindPersonByID(context.Background(), followedID); err != nil {
		return err
	}
	f.follows[followerID+"->"+followedID] = true
	return nil
}

func (f *fakeIdentity) Unfollow(_ context.Context, followerID, followedID string) error {
	delete(f.follows, followerID+"->"+followedID)
	return nil
}

func (f *fakeIdentity) IsFollowing(_ context.Context, followerID, followedID string) (bool, error) {
	return f.follows[followerID+"->"+followedID], nil
}

func (f *fakeIdentity) ListFollowers(_ context.Context, personID string) ([]identity.Person, error) {
	people := make([]identity.Person, 0)
	for key := range f.follows {
		followerID, followedID, _ := strings.Cut(key, "->")
		if followedID != personID {
			continue
		}
		if person, err := f.FindPersonByID(context.Background(), followerID); err == nil {
			people = append(people, *person)
		}
	}
	return people, nil
}

func (f *fakeIdentity) ListFollowing(_ context.Context, personID string) ([]identity.Person, error) {
	people := make([]identity.Person, 0)
	for key := range f.follows {
		followerID, followedID, _ := strings.Cut(key, "->")
		if followerID != personID {
			continue
		}
		if person, err := f.FindPersonByID(context.Background(), followedID); err == nil {
			people = append(people, *person)
		}
	}
	return people, nil
}

func (f *fakeIdentity) FollowStats(_ context.Context, personID string) (identity.FollowStats, error) {
	var stats identity.FollowStats
	for key := range f.follows {
		followerID, followedID, _ := strings.Cut(key, "->")
		if followedID == personID {
			stats.Followers++
		}
		if followerID == personID {
			stats.Following++
		}
	}
	return stats, nil
}

func (f *fakeIdentity) UpdateProfile(_ context.Context, personID, displayName, bio string) (*identity.Person, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}

	for _, person := range f.people {
		if person.ID == personID {
			person.DisplayName = displayName
			person.Bio = bio

			for _, account := range f.accounts {
				if account.Person.ID == personID {
					account.Person.DisplayName = displayName
					account.Person.Bio = bio
				}
			}

			return person, nil
		}
	}

	return nil, identity.ErrNotFound
}

func (f *fakeIdentity) Stats(_ context.Context) (identity.Stats, error) {
	return f.stats, nil
}

func (f *fakeIdentity) ListPendingAccounts(_ context.Context) ([]identity.Account, error) {
	var accounts []identity.Account
	for _, account := range f.accounts {
		if account.Status == identity.StatusPending {
			accounts = append(accounts, *account)
		}
	}
	return accounts, nil
}

func (f *fakeIdentity) ApproveAccount(_ context.Context, accountID string) error {
	for _, account := range f.accounts {
		if account.ID == accountID && account.Status == identity.StatusPending {
			account.Status = identity.StatusActive
			return nil
		}
	}
	return identity.ErrNotFound
}

func (f *fakeIdentity) RejectAccount(_ context.Context, accountID string) error {
	for email, account := range f.accounts {
		if account.ID == accountID && account.Status == identity.StatusPending {
			delete(f.accounts, email)
			delete(f.people, account.Person.Handle)
			return nil
		}
	}
	return identity.ErrNotFound
}

type fakeSiteSettings struct {
	settings  *sitesettings.Settings
	updateErr error
}

func (f *fakeSiteSettings) Get(_ context.Context) (*sitesettings.Settings, error) {
	if f.settings == nil {
		return sitesettings.Default(), nil
	}
	settings := *f.settings
	return &settings, nil
}

func (f *fakeSiteSettings) Update(_ context.Context, input sitesettings.UpdateInput) (*sitesettings.Settings, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if err := sitesettings.Validate(input); err != nil {
		return nil, err
	}
	f.settings = &sitesettings.Settings{
		RegistrationsMode:          input.RegistrationsMode,
		ClosedRegistrationsMessage: input.ClosedRegistrationsMessage,
	}
	return f.Get(context.Background())
}

func newAuthTestRouter(t *testing.T, identityService IdentityService, federationServices ...FederationService) http.Handler {
	return newAuthTestRouterWithSettings(t, identityService, nil, federationServices...)
}

func newAuthTestRouterWithSettings(t *testing.T, identityService IdentityService, settings SiteSettingsService, federationServices ...FederationService) http.Handler {
	t.Helper()

	deps := Deps{
		Config: &config.Config{
			Env:                    config.EnvDevelopment,
			InstanceName:           "Cultivar Test",
			LocalDomain:            "localhost:8080",
			LogLevel:               "error",
			SessionIdleTimeout:     time.Hour,
			SessionAbsoluteTimeout: 24 * time.Hour,
		},
		Identity:     identityService,
		SiteSettings: settings,
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	if len(federationServices) > 0 {
		deps.Federation = federationServices[0]
	}

	return NewRouter(deps)
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

func TestClosedRegistrationsHideAndRejectSignup(t *testing.T) {
	identityService := newFakeIdentity()
	settings := &fakeSiteSettings{settings: &sitesettings.Settings{
		RegistrationsMode:          sitesettings.ModeNone,
		ClosedRegistrationsMessage: "Voltamos em breve.",
	}}
	router := newAuthTestRouterWithSettings(t, identityService, settings)
	csrf := getCSRFCookie(t, router, "/cadastro")

	page := getPage(router, "/cadastro")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Voltamos em breve.") {
		t.Fatalf("GET /cadastro status/body = %d/%s, want closed message", page.Code, page.Body.String())
	}
	if strings.Contains(page.Body.String(), `action="/cadastro"`) {
		t.Error("closed registration page still renders the signup form")
	}

	response := postForm(router, "/cadastro", url.Values{
		"csrf_token":       {csrf.Value},
		"display_name":     {"Nova pessoa"},
		"handle":           {"novapessoa"},
		"email":            {"nova@example.com"},
		"password":         {testPassword},
		"password_confirm": {testPassword},
	}, csrf)
	if response.Code != http.StatusForbidden {
		t.Fatalf("POST /cadastro status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if len(identityService.accounts) != 0 {
		t.Fatal("closed registration created an account")
	}
}

func TestApprovedRegistrationCreatesPendingAccount(t *testing.T) {
	identityService := newFakeIdentity()
	settings := &fakeSiteSettings{settings: &sitesettings.Settings{RegistrationsMode: sitesettings.ModeApproved}}
	router := newAuthTestRouterWithSettings(t, identityService, settings)
	csrf := getCSRFCookie(t, router, "/cadastro")

	response := postForm(router, "/cadastro", url.Values{
		"csrf_token":       {csrf.Value},
		"display_name":     {"Nova pessoa"},
		"handle":           {"novapessoa"},
		"email":            {"nova@example.com"},
		"password":         {testPassword},
		"password_confirm": {testPassword},
	}, csrf)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), "aguardando aprovação") {
		t.Fatalf("status/body = %d/%s, want pending approval confirmation", response.Code, response.Body.String())
	}
	if account := identityService.accounts["nova@example.com"]; account == nil || account.Status != identity.StatusPending {
		t.Fatalf("created account = %+v, want pending", account)
	}
	if sessionCookie(response) != nil {
		t.Fatal("pending registration unexpectedly started a session")
	}
}

func TestAdminCanUpdateRegistrationSettings(t *testing.T) {
	identityService := newFakeIdentity()
	member := &identity.Account{ID: "member-1", Role: identity.RoleMember}
	admin := &identity.Account{ID: "admin-1", Role: identity.RoleAdmin}
	identityService.sessions["member-session"] = member
	identityService.sessions["admin-session"] = admin
	settings := &fakeSiteSettings{settings: sitesettings.Default()}
	router := newAuthTestRouterWithSettings(t, identityService, settings)

	memberResponse := getPage(router, "/admin/settings", &http.Cookie{Name: sessionCookieName, Value: "member-session"})
	if memberResponse.Code != http.StatusForbidden {
		t.Fatalf("non-admin status = %d, want %d", memberResponse.Code, http.StatusForbidden)
	}

	adminCookie := &http.Cookie{Name: sessionCookieName, Value: "admin-session"}
	csrf := csrfFor(t, router, "/", adminCookie)
	response := postForm(router, "/admin/settings", url.Values{
		"csrf_token":                   {csrf.Value},
		"registrations_mode":           {sitesettings.ModeNone},
		"closed_registrations_message": {"Inscrições temporariamente fechadas."},
	}, adminCookie, csrf)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/settings?updated=1" {
		t.Fatalf("status/location = %d/%q, want redirect after update", response.Code, response.Header().Get("Location"))
	}
	if settings.settings.RegistrationsMode != sitesettings.ModeNone {
		t.Fatalf("registration mode = %q, want none", settings.settings.RegistrationsMode)
	}
}

func TestAdminCanApproveAndRejectPendingAccounts(t *testing.T) {
	identityService := newFakeIdentity()
	admin := &identity.Account{ID: "admin-1", Role: identity.RoleAdmin}
	identityService.sessions["admin-session"] = admin
	approved, err := identityService.Register(context.Background(), identity.RegisterInput{
		Email: "aprovar@example.com", Handle: "aprovar", DisplayName: "Aprovar", Status: identity.StatusPending,
	})
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := identityService.Register(context.Background(), identity.RegisterInput{
		Email: "rejeitar@example.com", Handle: "rejeitar", DisplayName: "Rejeitar", Status: identity.StatusPending,
	})
	if err != nil {
		t.Fatal(err)
	}
	router := newAuthTestRouterWithSettings(t, identityService, &fakeSiteSettings{settings: sitesettings.Default()})
	adminCookie := &http.Cookie{Name: sessionCookieName, Value: "admin-session"}
	csrf := csrfFor(t, router, "/", adminCookie)

	approveResponse := postForm(router, "/admin/accounts/"+approved.ID+"/approve", url.Values{
		"csrf_token": {csrf.Value},
	}, adminCookie, csrf)
	if approveResponse.Code != http.StatusSeeOther {
		t.Fatalf("approve status = %d, want %d", approveResponse.Code, http.StatusSeeOther)
	}
	if identityService.accounts[approved.Email].Status != identity.StatusActive {
		t.Error("approved account is not active")
	}

	rejectResponse := postForm(router, "/admin/accounts/"+rejected.ID+"/reject", url.Values{
		"csrf_token": {csrf.Value},
	}, adminCookie, csrf)
	if rejectResponse.Code != http.StatusSeeOther {
		t.Fatalf("reject status = %d, want %d", rejectResponse.Code, http.StatusSeeOther)
	}
	if _, exists := identityService.accounts[rejected.Email]; exists {
		t.Error("rejected account still exists")
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
