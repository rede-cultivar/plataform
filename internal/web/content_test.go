package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rede-cultivar/plataform/internal/cultivation"
	"github.com/rede-cultivar/plataform/internal/federation"
	"github.com/rede-cultivar/plataform/internal/platform/config"
	"github.com/rede-cultivar/plataform/internal/practices"
)

// fakeCultivation implementa CultivationService em memória para os testes HTTP.
type fakeCultivation struct {
	cycles    map[string]*cultivation.Cycle
	entries   map[string][]cultivation.Entry
	follows   map[string]bool
	createErr error
	updateErr error
	entryErr  error
	nextID    int
}

func newFakeCultivation() *fakeCultivation {
	return &fakeCultivation{
		cycles:  make(map[string]*cultivation.Cycle),
		entries: make(map[string][]cultivation.Entry),
		follows: make(map[string]bool),
	}
}

// setFollowing registra no fake que followerID segue followedID.
func (f *fakeCultivation) setFollowing(followerID, followedID string) {
	f.follows[followerID+"->"+followedID] = true
}

func (f *fakeCultivation) CreateCycle(_ context.Context, personID string, input cultivation.CycleInput) (*cultivation.Cycle, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}

	f.nextID++
	cycle := &cultivation.Cycle{
		ID:           "cycle-" + strconv.Itoa(f.nextID),
		PersonID:     personID,
		PersonHandle: "ana",
		Name:         input.Name,
		Species:      input.Species,
		Variety:      input.Variety,
		Category:     input.Category,
		Environment:  input.Environment,
		Location:     input.Location,
		Substrate:    input.Substrate,
		Container:    input.Container,
		Technique:    input.Technique,
		Stage:        input.Stage,
		Status:       cultivation.StatusActive,
		Visibility:   input.Visibility,
		StartedOn:    input.StartedOn,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	f.cycles[cycle.ID] = cycle
	return cycle, nil
}

func (f *fakeCultivation) UpdateCycle(_ context.Context, personID, cycleID string, input cultivation.CycleUpdateInput) (*cultivation.Cycle, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}

	cycle, ok := f.cycles[cycleID]
	if !ok {
		return nil, cultivation.ErrNotFound
	}
	if !cycle.IsOwner(personID) {
		return nil, cultivation.ErrForbidden
	}

	cycle.Name = input.Name
	cycle.Species = input.Species
	cycle.Variety = input.Variety
	cycle.Category = input.Category
	cycle.Environment = input.Environment
	cycle.Location = input.Location
	cycle.Substrate = input.Substrate
	cycle.Container = input.Container
	cycle.Technique = input.Technique
	cycle.Stage = input.Stage
	cycle.Status = input.Status
	cycle.Visibility = input.Visibility
	cycle.StartedOn = input.StartedOn
	cycle.FinishedOn = input.FinishedOn
	cycle.Outcome = input.Outcome
	return cycle, nil
}

func (f *fakeCultivation) FindCycle(_ context.Context, viewerID, cycleID string) (*cultivation.Cycle, []cultivation.Entry, error) {
	cycle, ok := f.cycles[cycleID]
	if !ok {
		return nil, nil, cultivation.ErrNotFound
	}

	viewer := cultivation.Viewer{
		PersonID:  viewerID,
		Following: f.follows[viewerID+"->"+cycle.PersonID],
	}
	if !cycle.CanView(viewer) {
		return nil, nil, cultivation.ErrNotFound
	}

	visible := make([]cultivation.Entry, 0, len(f.entries[cycleID]))
	for _, entry := range f.entries[cycleID] {
		if entry.CanView(viewer, cycle) {
			visible = append(visible, entry)
		}
	}
	return cycle, visible, nil
}

func (f *fakeCultivation) FindEntry(_ context.Context, entryID string) (*cultivation.Entry, error) {
	for _, entries := range f.entries {
		for _, entry := range entries {
			if entry.ID == entryID {
				found := entry
				return &found, nil
			}
		}
	}
	return nil, cultivation.ErrNotFound
}

func (f *fakeCultivation) ListPersonCycles(_ context.Context, personID string) ([]cultivation.Cycle, error) {
	result := make([]cultivation.Cycle, 0)
	for _, cycle := range f.cycles {
		if cycle.PersonID == personID {
			result = append(result, *cycle)
		}
	}
	return result, nil
}

func (f *fakeCultivation) ListVisibleCycles(_ context.Context, viewerID, personID string) ([]cultivation.Cycle, error) {
	viewer := cultivation.Viewer{
		PersonID:  viewerID,
		Following: f.follows[viewerID+"->"+personID],
	}

	result := make([]cultivation.Cycle, 0)
	for _, cycle := range f.cycles {
		if cycle.PersonID != personID {
			continue
		}
		if !cycle.CanView(viewer) {
			continue
		}
		if viewerID != personID && cycle.Status == cultivation.StatusArchived {
			continue
		}
		result = append(result, *cycle)
	}
	return result, nil
}

func (f *fakeCultivation) ListPublicCycles(_ context.Context, filter cultivation.CycleFilter) ([]cultivation.Cycle, error) {
	result := make([]cultivation.Cycle, 0)
	for _, cycle := range f.cycles {
		if cycle.Visibility != cultivation.VisibilityPublic || cycle.Status == cultivation.StatusArchived {
			continue
		}
		if filter.Species != "" && !strings.Contains(strings.ToLower(cycle.Species), strings.ToLower(filter.Species)) {
			continue
		}
		result = append(result, *cycle)
	}
	return result, nil
}

func (f *fakeCultivation) CreateEntry(_ context.Context, personID, cycleID string, input cultivation.EntryInput) (*cultivation.Entry, error) {
	if f.entryErr != nil {
		return nil, f.entryErr
	}

	cycle, ok := f.cycles[cycleID]
	if !ok {
		return nil, cultivation.ErrNotFound
	}
	if !cycle.IsOwner(personID) {
		return nil, cultivation.ErrForbidden
	}

	entry := cultivation.Entry{
		ID:              "entry-1",
		CycleID:         cycleID,
		AuthorID:        personID,
		AuthorHandle:    "ana",
		OccurredAt:      input.OccurredAt,
		Body:            input.Body,
		Stage:           input.Stage,
		Measurements:    input.Measurements,
		Interventions:   input.Interventions,
		Occurrences:     input.Occurrences,
		HarvestQuantity: input.HarvestQuantity,
		HarvestUnit:     input.HarvestUnit,
		PracticeID:      input.PracticeID,
		Visibility:      input.Visibility,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	f.entries[cycleID] = append(f.entries[cycleID], entry)
	cycle.EntryCount = len(f.entries[cycleID])
	return &entry, nil
}

func (f *fakeCultivation) UpdateEntry(_ context.Context, personID, cycleID, entryID string, input cultivation.EntryInput) (*cultivation.Entry, error) {
	if f.entryErr != nil {
		return nil, f.entryErr
	}

	cycle, ok := f.cycles[cycleID]
	if !ok {
		return nil, cultivation.ErrNotFound
	}
	if !cycle.IsOwner(personID) {
		return nil, cultivation.ErrForbidden
	}

	for i := range f.entries[cycleID] {
		if f.entries[cycleID][i].ID != entryID {
			continue
		}
		f.entries[cycleID][i].OccurredAt = input.OccurredAt
		f.entries[cycleID][i].Body = input.Body
		f.entries[cycleID][i].Stage = input.Stage
		f.entries[cycleID][i].Measurements = input.Measurements
		f.entries[cycleID][i].Interventions = input.Interventions
		f.entries[cycleID][i].Occurrences = input.Occurrences
		f.entries[cycleID][i].HarvestQuantity = input.HarvestQuantity
		f.entries[cycleID][i].HarvestUnit = input.HarvestUnit
		f.entries[cycleID][i].PracticeID = input.PracticeID
		f.entries[cycleID][i].Visibility = input.Visibility
		f.entries[cycleID][i].UpdatedAt = time.Now()
		return &f.entries[cycleID][i], nil
	}
	return nil, cultivation.ErrNotFound
}

// fakePractice implementa PracticeService em memória para os testes HTTP.
type fakePractice struct {
	practices  map[string]*practices.Practice
	createErr  error
	updateErr  error
	publishErr error
	archiveErr error
	nextID     int
}

func newFakePractice() *fakePractice {
	return &fakePractice{practices: make(map[string]*practices.Practice)}
}

func (f *fakePractice) CreatePractice(_ context.Context, authorID string, input practices.Input) (*practices.Practice, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}

	f.nextID++
	practice := &practices.Practice{
		ID:           "practice-" + strconv.Itoa(f.nextID),
		AuthorID:     authorID,
		AuthorHandle: "ana",
		Title:        input.Title,
		Summary:      input.Summary,
		Category:     input.Category,
		Difficulty:   input.Difficulty,
		TimeEstimate: input.TimeEstimate,
		CostEstimate: input.CostEstimate,
		Materials:    input.Materials,
		Steps:        input.Steps,
		Safety:       input.Safety,
		Adaptations:  input.Adaptations,
		References:   input.References,
		License:      input.License,
		Version:      1,
		Status:       practices.StatusDraft,
		Visibility:   practices.VisibilityPrivate,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if input.Publish {
		published := time.Now()
		practice.Status = practices.StatusPublished
		practice.Visibility = practices.VisibilityPublic
		practice.PublishedAt = &published
	}
	f.practices[practice.ID] = practice
	return practice, nil
}

func (f *fakePractice) UpdatePractice(_ context.Context, authorID, practiceID string, input practices.Input) (*practices.Practice, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}

	practice, ok := f.practices[practiceID]
	if !ok {
		return nil, practices.ErrNotFound
	}
	if !practice.IsOwner(authorID) {
		return nil, practices.ErrForbidden
	}

	practice.Title = input.Title
	practice.Summary = input.Summary
	practice.Category = input.Category
	practice.Difficulty = input.Difficulty
	practice.TimeEstimate = input.TimeEstimate
	practice.CostEstimate = input.CostEstimate
	practice.Materials = input.Materials
	practice.Steps = input.Steps
	practice.Safety = input.Safety
	practice.Adaptations = input.Adaptations
	practice.References = input.References
	practice.License = input.License
	practice.Version++
	if input.Publish {
		published := time.Now()
		practice.Status = practices.StatusPublished
		practice.Visibility = practices.VisibilityPublic
		if practice.PublishedAt == nil {
			practice.PublishedAt = &published
		}
	}
	return practice, nil
}

func (f *fakePractice) FindPractice(_ context.Context, viewerID, practiceID string) (*practices.Practice, error) {
	practice, ok := f.practices[practiceID]
	if !ok {
		return nil, practices.ErrNotFound
	}
	if !practice.CanView(viewerID) {
		return nil, practices.ErrNotFound
	}
	return practice, nil
}

func (f *fakePractice) ListPublicPractices(_ context.Context, filter practices.Filter) ([]practices.Practice, error) {
	result := make([]practices.Practice, 0)
	for _, practice := range f.practices {
		if practice.Status != practices.StatusPublished || practice.Visibility != practices.VisibilityPublic {
			continue
		}
		if filter.Category != "" && !strings.Contains(strings.ToLower(practice.Category), strings.ToLower(filter.Category)) {
			continue
		}
		result = append(result, *practice)
	}
	return result, nil
}

func (f *fakePractice) ListAuthorPractices(_ context.Context, authorID string) ([]practices.Practice, error) {
	result := make([]practices.Practice, 0)
	for _, practice := range f.practices {
		if practice.AuthorID == authorID {
			result = append(result, *practice)
		}
	}
	return result, nil
}

func (f *fakePractice) PublishPractice(_ context.Context, authorID, practiceID string) (*practices.Practice, error) {
	if f.publishErr != nil {
		return nil, f.publishErr
	}

	practice, ok := f.practices[practiceID]
	if !ok {
		return nil, practices.ErrNotFound
	}
	if !practice.IsOwner(authorID) {
		return nil, practices.ErrForbidden
	}

	published := time.Now()
	practice.Status = practices.StatusPublished
	practice.Visibility = practices.VisibilityPublic
	if practice.PublishedAt == nil {
		practice.PublishedAt = &published
	}
	return practice, nil
}

func (f *fakePractice) ArchivePractice(_ context.Context, authorID, practiceID string) (*practices.Practice, error) {
	if f.archiveErr != nil {
		return nil, f.archiveErr
	}

	practice, ok := f.practices[practiceID]
	if !ok {
		return nil, practices.ErrNotFound
	}
	if !practice.IsOwner(authorID) {
		return nil, practices.ErrForbidden
	}
	practice.Status = practices.StatusArchived
	return practice, nil
}

func newContentTestRouter(t *testing.T, identityService IdentityService, cultivationService CultivationService, practiceService PracticeService, mediaServices ...MediaService) http.Handler {
	t.Helper()

	deps := contentTestDeps(identityService, cultivationService, practiceService)
	if len(mediaServices) > 0 {
		deps.Media = mediaServices[0]
	}

	return NewRouter(deps)
}

// newContentTestRouterWithFederation monta o roteador com federação simulada.
func newContentTestRouterWithFederation(t *testing.T, identityService IdentityService, cultivationService CultivationService, practiceService PracticeService, mediaService MediaService, federationService FederationService) http.Handler {
	t.Helper()

	deps := contentTestDeps(identityService, cultivationService, practiceService)
	deps.Media = mediaService
	deps.Federation = federationService

	return NewRouter(deps)
}

func contentTestDeps(identityService IdentityService, cultivationService CultivationService, practiceService PracticeService) Deps {
	return Deps{
		Config: &config.Config{
			Env:                    config.EnvDevelopment,
			InstanceName:           "Cultivar Test",
			LocalDomain:            "localhost:8080",
			LogLevel:               "error",
			SecretKey:              "segredo-de-teste",
			SessionIdleTimeout:     time.Hour,
			SessionAbsoluteTimeout: 24 * time.Hour,
		},
		Identity:    identityService,
		Cultivation: cultivationService,
		Practices:   practiceService,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func sampleCycle(id string, visibility cultivation.Visibility) *cultivation.Cycle {
	return &cultivation.Cycle{
		ID:           id,
		PersonID:     "person-1",
		PersonHandle: "ana",
		Name:         "Tomates da varanda",
		Species:      "Solanum lycopersicum",
		Environment:  cultivation.EnvironmentOutdoor,
		Stage:        cultivation.StageSeedling,
		Status:       cultivation.StatusActive,
		Visibility:   visibility,
		StartedOn:    time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func samplePractice(id string, status practices.Status, visibility practices.Visibility) *practices.Practice {
	practice := &practices.Practice{
		ID:           id,
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		Title:        "Adubo de casca de banana",
		Summary:      "Adubo simples para hortas domésticas.",
		Category:     "Adubação",
		Difficulty:   practices.DifficultyBeginner,
		Materials:    []string{"Cascas de banana", "Água"},
		Steps:        []string{"Ferva as cascas", "Deixe descansar", "Dilua e aplique"},
		License:      "CC BY-SA 4.0",
		Version:      1,
		Status:       status,
		Visibility:   visibility,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if status == practices.StatusPublished {
		published := time.Now()
		practice.PublishedAt = &published
	}
	return practice
}

// csrfFor busca o token CSRF em uma página autenticada.
func csrfFor(t *testing.T, router http.Handler, target string, cookies ...*http.Cookie) *http.Cookie {
	t.Helper()

	response := getPage(router, target, cookies...)
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == csrfCookieName {
			return cookie
		}
	}

	t.Fatalf("GET %s did not set the CSRF cookie; status = %d", target, response.Code)
	return nil
}

func getPage(router http.Handler, target string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, target, nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestDiaryListsOnlyPublicCycles(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["public"] = sampleCycle("public", cultivation.VisibilityPublic)
	cultivationService.cycles["private"] = sampleCycle("private", cultivation.VisibilityPrivate)

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice())
	response := getPage(router, "/diario")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	body := response.Body.String()
	if !strings.Contains(body, "Tomates da varanda") {
		t.Error("diary does not list the public cycle")
	}
	if strings.Contains(body, `href="/diario/private"`) {
		t.Error("diary lists a private cycle")
	}
}

func TestMyDiaryRequiresAuthentication(t *testing.T) {
	router := newContentTestRouter(t, newFakeIdentity(), newFakeCultivation(), newFakePractice())
	response := getPage(router, "/diario/meu")

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	if location := response.Header().Get("Location"); !strings.HasPrefix(location, "/entrar") {
		t.Errorf("Location = %q, want /entrar", location)
	}
}

func TestMyDiaryListsPrivateCycles(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["private"] = sampleCycle("private", cultivation.VisibilityPrivate)

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice())
	session := loginAsAna(t, router, identityService)

	response := getPage(router, "/diario/meu", session)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if body := response.Body.String(); !strings.Contains(body, "Tomates da varanda") {
		t.Error("own diary does not list the private cycle")
	}
}

func TestCreateCycleRedirectsToCycle(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice())
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/novo", session)

	response := postForm(router, "/diario", url.Values{
		"csrf_token":  {csrf.Value},
		"name":        {"Tomates da varanda"},
		"species":     {"Solanum lycopersicum"},
		"environment": {"outdoor"},
		"stage":       {"seedling"},
		"visibility":  {"private"},
		"started_on":  {"2026-09-01"},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/diario/cycle-1" {
		t.Errorf("Location = %q, want /diario/cycle-1", location)
	}

	cycle, ok := cultivationService.cycles["cycle-1"]
	if !ok {
		t.Fatal("cycle was not stored")
	}
	if cycle.Visibility != cultivation.VisibilityPrivate {
		t.Errorf("visibility = %q, want private", cycle.Visibility)
	}
	if cycle.PersonID != "person-1" {
		t.Errorf("person = %q, want person-1", cycle.PersonID)
	}
}

func TestCreateCycleRejectsMissingCSRF(t *testing.T) {
	identityService := newFakeIdentity()
	router := newContentTestRouter(t, identityService, newFakeCultivation(), newFakePractice())
	session := loginAsAna(t, router, identityService)

	response := postForm(router, "/diario", url.Values{
		"name":       {"Tomates"},
		"species":    {"Solanum lycopersicum"},
		"started_on": {"2026-09-01"},
	}, session)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestCreateCycleShowsValidationError(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.createErr = &cultivation.ValidationError{Field: "name", Message: "Informe um nome de ciclo com até 120 caracteres."}

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice())
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/novo", session)

	response := postForm(router, "/diario", url.Values{
		"csrf_token": {csrf.Value},
		"name":       {""},
		"species":    {"Solanum lycopersicum"},
		"started_on": {"2026-09-01"},
	}, session, csrf)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if body := response.Body.String(); !strings.Contains(body, "Informe um nome de ciclo") {
		t.Error("response does not contain the validation message")
	}
}

func TestPrivateCycleHiddenFromAnonymousVisitors(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["private"] = sampleCycle("private", cultivation.VisibilityPrivate)
	cultivationService.entries["private"] = []cultivation.Entry{{
		ID:         "entry-1",
		CycleID:    "private",
		AuthorID:   "person-1",
		OccurredAt: time.Now(),
		Body:       "Registro privado",
		Visibility: cultivation.EntryVisibilityInherit,
	}}

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice())

	anonymous := getPage(router, "/diario/private")
	if anonymous.Code != http.StatusNotFound {
		t.Fatalf("anonymous status = %d, want %d", anonymous.Code, http.StatusNotFound)
	}

	session := loginAsAna(t, router, identityService)
	owner := getPage(router, "/diario/private", session)
	if owner.Code != http.StatusOK {
		t.Fatalf("owner status = %d, want %d; body = %s", owner.Code, http.StatusOK, owner.Body.String())
	}
	if body := owner.Body.String(); !strings.Contains(body, "Registro privado") {
		t.Error("owner cannot see the private entry")
	}
}

func TestCreateEntryRedirectsToTimeline(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPrivate)

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice())
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/registros/novo", session)

	response := postForm(router, "/diario/cycle-1/registros", url.Values{
		"csrf_token":    {csrf.Value},
		"occurred_at":   {"2026-09-10T09:30"},
		"body":          {"Primeira flor observada."},
		"stage":         {"flowering"},
		"measurements":  {"altura_cm=25\ntemperatura_c=22"},
		"interventions": {"rega, poda"},
		"visibility":    {"inherit"},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/diario/cycle-1#registro-entry-1" {
		t.Errorf("Location = %q, want timeline anchor", location)
	}

	entries := cultivationService.entries["cycle-1"]
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].Measurements["altura_cm"] != "25" {
		t.Errorf("measurements = %v, want altura_cm=25", entries[0].Measurements)
	}
	if len(entries[0].Interventions) != 2 {
		t.Errorf("interventions = %v, want 2 items", entries[0].Interventions)
	}
}

func TestPracticeLibraryListsOnlyPublished(t *testing.T) {
	practiceService := newFakePractice()
	practiceService.practices["published"] = samplePractice("published", practices.StatusPublished, practices.VisibilityPublic)
	practiceService.practices["draft"] = samplePractice("draft", practices.StatusDraft, practices.VisibilityPrivate)

	router := newContentTestRouter(t, newFakeIdentity(), newFakeCultivation(), practiceService)
	response := getPage(router, "/praticas")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	body := response.Body.String()
	if !strings.Contains(body, "Adubo de casca de banana") {
		t.Error("library does not list the published practice")
	}
	if strings.Contains(body, `href="/praticas/draft"`) {
		t.Error("library lists a draft practice")
	}
}

func TestPracticeDraftHiddenFromAnonymousVisitors(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	practiceService.practices["draft"] = samplePractice("draft", practices.StatusDraft, practices.VisibilityPrivate)

	router := newContentTestRouter(t, identityService, newFakeCultivation(), practiceService)

	anonymous := getPage(router, "/praticas/draft")
	if anonymous.Code != http.StatusNotFound {
		t.Fatalf("anonymous status = %d, want %d", anonymous.Code, http.StatusNotFound)
	}

	session := loginAsAna(t, router, identityService)
	owner := getPage(router, "/praticas/draft", session)
	if owner.Code != http.StatusOK {
		t.Fatalf("owner status = %d, want %d; body = %s", owner.Code, http.StatusOK, owner.Body.String())
	}
	if body := owner.Body.String(); !strings.Contains(body, "Adubo de casca de banana") {
		t.Error("owner cannot see the draft practice")
	}
}

func TestCreatePracticeCanPublish(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	router := newContentTestRouter(t, identityService, newFakeCultivation(), practiceService)
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
	if location := response.Header().Get("Location"); location != "/praticas/practice-1" {
		t.Errorf("Location = %q, want /praticas/practice-1", location)
	}

	practice, ok := practiceService.practices["practice-1"]
	if !ok {
		t.Fatal("practice was not stored")
	}
	if practice.Status != practices.StatusPublished || practice.Visibility != practices.VisibilityPublic {
		t.Errorf("practice = %s/%s, want published/public", practice.Status, practice.Visibility)
	}
	if len(practice.Steps) != 3 {
		t.Errorf("steps = %v, want 3 items", practice.Steps)
	}
}

func TestCreatePracticeRejectsMissingCSRF(t *testing.T) {
	identityService := newFakeIdentity()
	router := newContentTestRouter(t, identityService, newFakeCultivation(), newFakePractice())
	session := loginAsAna(t, router, identityService)

	response := postForm(router, "/praticas", url.Values{
		"title": {"Adubo"},
	}, session)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestPublishPracticeRedirectsToPractice(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	practiceService.practices["draft"] = samplePractice("draft", practices.StatusDraft, practices.VisibilityPrivate)

	router := newContentTestRouter(t, identityService, newFakeCultivation(), practiceService)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/praticas/draft", session)

	response := postForm(router, "/praticas/draft/publicar", url.Values{
		"csrf_token": {csrf.Value},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/praticas/draft" {
		t.Errorf("Location = %q, want /praticas/draft", location)
	}
	if practiceService.practices["draft"].Status != practices.StatusPublished {
		t.Error("practice was not published")
	}
}

func TestArchivePracticeRedirectsToPractice(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	practiceService.practices["published"] = samplePractice("published", practices.StatusPublished, practices.VisibilityPublic)

	router := newContentTestRouter(t, identityService, newFakeCultivation(), practiceService)
	session := loginAsAna(t, router, identityService)
	csrf := getCSRFCookie(t, router, "/praticas/published")

	response := postForm(router, "/praticas/published/arquivar", url.Values{
		"csrf_token": {csrf.Value},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if practiceService.practices["published"].Status != practices.StatusArchived {
		t.Error("practice was not archived")
	}
}

func TestCycleEditFormAndUpdate(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPrivate)

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice())
	session := loginAsAna(t, router, identityService)

	formPage := getPage(router, "/diario/cycle-1/editar", session)
	if formPage.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body = %s", formPage.Code, http.StatusOK, formPage.Body.String())
	}
	if body := formPage.Body.String(); !strings.Contains(body, `action="/diario/cycle-1/editar"`) {
		t.Error("edit form does not point to the cycle")
	}
	csrf := responseCSRFCookie(formPage)
	if csrf == nil {
		t.Fatal("edit form did not set the CSRF cookie")
	}

	response := postForm(router, "/diario/cycle-1/editar", url.Values{
		"csrf_token":  {csrf.Value},
		"name":        {"Tomates da varanda"},
		"species":     {"Solanum lycopersicum"},
		"environment": {"outdoor"},
		"stage":       {"flowering"},
		"status":      {"active"},
		"visibility":  {"private"},
		"started_on":  {"2026-09-01"},
		"finished_on": {""},
		"outcome":     {"Floração vigorosa."},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if got := cultivationService.cycles["cycle-1"].Stage; got != cultivation.StageFlowering {
		t.Errorf("stage = %q, want flowering", got)
	}
}

func TestCycleEditValidationErrorKeepsFormTarget(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPrivate)
	cultivationService.updateErr = &cultivation.ValidationError{Field: "outcome", Message: "O resultado deve ter no máximo 2000 caracteres."}

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice())
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/editar", session)

	response := postForm(router, "/diario/cycle-1/editar", url.Values{
		"csrf_token":  {csrf.Value},
		"name":        {"Tomates da varanda"},
		"species":     {"Solanum lycopersicum"},
		"environment": {"outdoor"},
		"stage":       {"flowering"},
		"status":      {"active"},
		"visibility":  {"private"},
		"started_on":  {"2026-09-01"},
	}, session, csrf)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	body := response.Body.String()
	if !strings.Contains(body, "O resultado deve ter no máximo 2000 caracteres.") {
		t.Error("response does not contain the validation message")
	}
	if !strings.Contains(body, `action="/diario/cycle-1/editar"`) {
		t.Error("validation response lost the form target")
	}
}

// cycleEditValues monta o formulário de edição de ciclo com os campos padrão.
func cycleEditValues(csrf, name, visibility string) url.Values {
	return url.Values{
		"csrf_token":  {csrf},
		"name":        {name},
		"species":     {"Solanum lycopersicum"},
		"environment": {"outdoor"},
		"stage":       {"flowering"},
		"status":      {"active"},
		"visibility":  {visibility},
		"started_on":  {"2026-09-01"},
	}
}

func TestCycleEditToPrivateFederatesDelete(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro público.",
		Visibility:   cultivation.EntryVisibilityInherit,
		CreatedAt:    time.Now(),
	}}
	federationService := newFakeFederation(t)

	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), nil, federationService,
	)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/editar", session)

	response := postForm(router, "/diario/cycle-1/editar",
		cycleEditValues(csrf.Value, "Tomates da varanda", "private"), session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.deleteCalls != 1 {
		t.Fatalf("deleteCalls = %d, want 1", federationService.deleteCalls)
	}
	if federationService.updateCalls != 0 {
		t.Errorf("updateCalls = %d, want 0", federationService.updateCalls)
	}
	if federationService.lastDelete.ID != "entry-1" {
		t.Errorf("deleted entry = %q, want entry-1", federationService.lastDelete.ID)
	}
}

func TestCycleEditCardChangeFederatesUpdate(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro público.",
		Visibility:   cultivation.EntryVisibilityInherit,
		CreatedAt:    time.Now(),
	}}
	federationService := newFakeFederation(t)

	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), nil, federationService,
	)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/editar", session)

	response := postForm(router, "/diario/cycle-1/editar",
		cycleEditValues(csrf.Value, "Tomates da varanda renovada", "public"), session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", federationService.updateCalls)
	}
	if federationService.deleteCalls != 0 {
		t.Errorf("deleteCalls = %d, want 0", federationService.deleteCalls)
	}
	if federationService.lastUpdate.ID != "entry-1" {
		t.Errorf("updated entry = %q, want entry-1", federationService.lastUpdate.ID)
	}
	if federationService.lastUpdateCycle.Name != "Tomates da varanda renovada" {
		t.Errorf("updated cycle = %q, want new name", federationService.lastUpdateCycle.Name)
	}
}

func TestCycleEditWithoutCardChangeSkipsFederation(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro público.",
		Visibility:   cultivation.EntryVisibilityInherit,
		CreatedAt:    time.Now(),
	}}
	federationService := newFakeFederation(t)

	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), nil, federationService,
	)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/editar", session)

	response := postForm(router, "/diario/cycle-1/editar",
		cycleEditValues(csrf.Value, "Tomates da varanda", "public"), session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.updateCalls != 0 {
		t.Errorf("updateCalls = %d, want 0", federationService.updateCalls)
	}
	if federationService.deleteCalls != 0 {
		t.Errorf("deleteCalls = %d, want 0", federationService.deleteCalls)
	}
}

func TestCycleEditToPublicFederatesEntries(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPrivate)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro privado.",
		Visibility:   cultivation.EntryVisibilityInherit,
		CreatedAt:    time.Now(),
	}}
	federationService := newFakeFederation(t)

	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), nil, federationService,
	)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/editar", session)

	response := postForm(router, "/diario/cycle-1/editar",
		cycleEditValues(csrf.Value, "Tomates da varanda", "public"), session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", federationService.updateCalls)
	}
	if federationService.deleteCalls != 0 {
		t.Errorf("deleteCalls = %d, want 0", federationService.deleteCalls)
	}
}

func TestCycleEditToFollowersFederatesEntries(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPrivate)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro privado.",
		Visibility:   cultivation.EntryVisibilityInherit,
		CreatedAt:    time.Now(),
	}}
	federationService := newFakeFederation(t)

	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), nil, federationService,
	)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/editar", session)

	response := postForm(router, "/diario/cycle-1/editar",
		cycleEditValues(csrf.Value, "Tomates da varanda", "followers"), session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", federationService.updateCalls)
	}
	if federationService.deleteCalls != 0 {
		t.Errorf("deleteCalls = %d, want 0", federationService.deleteCalls)
	}
}

func TestCycleEditFromFollowersToPrivateDeletesEntries(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityFollowers)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro para seguidores.",
		Visibility:   cultivation.EntryVisibilityInherit,
		CreatedAt:    time.Now(),
	}}
	federationService := newFakeFederation(t)

	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), nil, federationService,
	)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/editar", session)

	response := postForm(router, "/diario/cycle-1/editar",
		cycleEditValues(csrf.Value, "Tomates da varanda", "private"), session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.deleteCalls != 1 {
		t.Fatalf("deleteCalls = %d, want 1", federationService.deleteCalls)
	}
	if federationService.updateCalls != 0 {
		t.Errorf("updateCalls = %d, want 0", federationService.updateCalls)
	}
}

func TestPracticeEditFormAndUpdate(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	practiceService.practices["draft"] = samplePractice("draft", practices.StatusDraft, practices.VisibilityPrivate)

	router := newContentTestRouter(t, identityService, newFakeCultivation(), practiceService)
	session := loginAsAna(t, router, identityService)

	formPage := getPage(router, "/praticas/draft/editar", session)
	if formPage.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body = %s", formPage.Code, http.StatusOK, formPage.Body.String())
	}
	if body := formPage.Body.String(); !strings.Contains(body, `action="/praticas/draft/editar"`) {
		t.Error("edit form does not point to the practice")
	}
	csrf := responseCSRFCookie(formPage)
	if csrf == nil {
		t.Fatal("edit form did not set the CSRF cookie")
	}

	response := postForm(router, "/praticas/draft/editar", url.Values{
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
	if got := practiceService.practices["draft"].Version; got != 2 {
		t.Errorf("version = %d, want 2", got)
	}
}

func TestPracticeEditValidationErrorKeepsFormTarget(t *testing.T) {
	identityService := newFakeIdentity()
	practiceService := newFakePractice()
	practiceService.practices["draft"] = samplePractice("draft", practices.StatusDraft, practices.VisibilityPrivate)
	practiceService.updateErr = &practices.ValidationError{Field: "steps", Message: "Uma prática publicada precisa ter pelo menos uma etapa."}

	router := newContentTestRouter(t, identityService, newFakeCultivation(), practiceService)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/praticas/draft/editar", session)

	response := postForm(router, "/praticas/draft/editar", url.Values{
		"csrf_token": {csrf.Value},
		"title":      {"Adubo de casca de banana"},
		"difficulty": {"beginner"},
		"action":     {"save"},
	}, session, csrf)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	body := response.Body.String()
	if !strings.Contains(body, "Uma prática publicada precisa ter pelo menos uma etapa.") {
		t.Error("response does not contain the validation message")
	}
	if !strings.Contains(body, `action="/praticas/draft/editar"`) {
		t.Error("validation response lost the form target")
	}
}

func TestEntryFormRendersAllFields(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPrivate)
	practiceService := newFakePractice()
	practiceService.practices["practice-1"] = samplePractice("practice-1", practices.StatusPublished, practices.VisibilityPublic)

	router := newContentTestRouter(t, identityService, cultivationService, practiceService)
	session := loginAsAna(t, router, identityService)

	response := getPage(router, "/diario/cycle-1/registros/novo", session)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()
	for _, want := range []string{
		`name="occurred_at"`,
		"Prática utilizada",
		"Adubo de casca de banana",
		`name="harvest_quantity"`,
		"Herdar do ciclo",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("entry form does not contain %q", want)
		}
	}
}

func TestEntryEditFormAndUpdate(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cycle := sampleCycle("cycle-1", cultivation.VisibilityPrivate)
	cultivationService.cycles["cycle-1"] = cycle
	quantity := 1.25
	stage := cultivation.StageVegetative
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:              "entry-1",
		CycleID:         "cycle-1",
		AuthorID:        "person-1",
		AuthorHandle:    "ana",
		OccurredAt:      time.Now(),
		Body:            "Registro inicial.",
		Stage:           &stage,
		Measurements:    map[string]string{"altura_cm": "20"},
		Interventions:   []string{"rega"},
		Occurrences:     []string{"solo seco"},
		HarvestQuantity: &quantity,
		HarvestUnit:     "kg",
		Visibility:      cultivation.EntryVisibilityInherit,
		CreatedAt:       time.Now(),
	}}

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice())
	session := loginAsAna(t, router, identityService)

	formPage := getPage(router, "/diario/cycle-1/registros/entry-1/editar", session)
	if formPage.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body = %s", formPage.Code, http.StatusOK, formPage.Body.String())
	}
	if body := formPage.Body.String(); !strings.Contains(body, `action="/diario/cycle-1/registros/entry-1/editar"`) {
		t.Error("edit form does not point to the entry")
	}
	csrf := responseCSRFCookie(formPage)
	if csrf == nil {
		t.Fatal("edit form did not set the CSRF cookie")
	}

	response := postForm(router, "/diario/cycle-1/registros/entry-1/editar", url.Values{
		"csrf_token":       {csrf.Value},
		"occurred_at":      {time.Now().Add(1 * time.Hour).Format("2006-01-02T15:04")},
		"body":             {"Registro atualizado."},
		"stage":            {"flowering"},
		"measurements":     {"altura_cm=30\nfolhas=8"},
		"interventions":    {"rega, poda"},
		"occurrences":      {"sem ocorrência"},
		"harvest_quantity": {"2"},
		"harvest_unit":     {"kg"},
		"visibility":       {"public"},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if got := cultivationService.entries["cycle-1"][0].Body; got != "Registro atualizado." {
		t.Errorf("body = %q, want Registro atualizado.", got)
	}
	if got := cultivationService.entries["cycle-1"][0].Visibility; got != cultivation.EntryVisibilityPublic {
		t.Errorf("visibility = %q, want public", got)
	}
}

func TestEntryEditFederatesUpdate(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro inicial.",
		Visibility:   cultivation.EntryVisibilityInherit,
		CreatedAt:    time.Now(),
	}}
	federationService := newFakeFederation(t)

	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), nil, federationService,
	)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/registros/entry-1/editar", session)

	response := postForm(router, "/diario/cycle-1/registros/entry-1/editar", url.Values{
		"csrf_token":  {csrf.Value},
		"occurred_at": {time.Now().Format("2006-01-02T15:04")},
		"body":        {"Registro atualizado."},
		"visibility":  {"public"},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", federationService.updateCalls)
	}
	if federationService.lastUpdate.Body != "Registro atualizado." {
		t.Errorf("updated body = %q, want Registro atualizado.", federationService.lastUpdate.Body)
	}
}

func TestEntryEditToPrivateFederatesDelete(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro público.",
		Visibility:   cultivation.EntryVisibilityPublic,
		CreatedAt:    time.Now(),
	}}
	federationService := newFakeFederation(t)

	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), nil, federationService,
	)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/registros/entry-1/editar", session)

	response := postForm(router, "/diario/cycle-1/registros/entry-1/editar", url.Values{
		"csrf_token":  {csrf.Value},
		"occurred_at": {time.Now().Format("2006-01-02T15:04")},
		"body":        {"Registro privado."},
		"visibility":  {"private"},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.deleteCalls != 1 {
		t.Fatalf("deleteCalls = %d, want 1", federationService.deleteCalls)
	}
	if federationService.updateCalls != 0 {
		t.Errorf("updateCalls = %d, want 0", federationService.updateCalls)
	}
	if federationService.lastDelete.ID != "entry-1" {
		t.Errorf("deleted entry = %q, want entry-1", federationService.lastDelete.ID)
	}
}

func TestEntryEditToFollowersFederatesUpdate(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro público.",
		Visibility:   cultivation.EntryVisibilityPublic,
		CreatedAt:    time.Now(),
	}}
	federationService := newFakeFederation(t)

	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), nil, federationService,
	)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/registros/entry-1/editar", session)

	response := postForm(router, "/diario/cycle-1/registros/entry-1/editar", url.Values{
		"csrf_token":  {csrf.Value},
		"occurred_at": {time.Now().Format("2006-01-02T15:04")},
		"body":        {"Registro para seguidores."},
		"visibility":  {"followers"},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", federationService.updateCalls)
	}
	if federationService.deleteCalls != 0 {
		t.Errorf("deleteCalls = %d, want 0", federationService.deleteCalls)
	}
	if federationService.lastUpdate.Visibility != cultivation.EntryVisibilityFollowers {
		t.Errorf("updated visibility = %q, want followers", federationService.lastUpdate.Visibility)
	}
}

func TestCycleTimelineRendersStructuredEntry(t *testing.T) {
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	stage := cultivation.StageFlowering
	quantity := 1.5
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:              "entry-1",
		CycleID:         "cycle-1",
		AuthorID:        "person-1",
		AuthorHandle:    "ana",
		OccurredAt:      time.Now(),
		Body:            "Primeira colheita.",
		Stage:           &stage,
		Measurements:    map[string]string{"altura_cm": "25"},
		Interventions:   []string{"rega", "poda"},
		Occurrences:     []string{"mosca-branca"},
		HarvestQuantity: &quantity,
		HarvestUnit:     "kg",
		Visibility:      cultivation.EntryVisibilityInherit,
		CreatedAt:       time.Now(),
	}}

	router := newContentTestRouter(t, newFakeIdentity(), cultivationService, newFakePractice())
	response := getPage(router, "/diario/cycle-1")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()
	for _, want := range []string{"Primeira colheita.", "altura_cm", "25", "rega", "mosca-branca", "1.5 kg", "Floração"} {
		if !strings.Contains(body, want) {
			t.Errorf("timeline does not contain %q", want)
		}
	}
}

func TestCycleTimelineRendersFederatedInteractions(t *testing.T) {
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro público.",
		Visibility:   cultivation.EntryVisibilityInherit,
		CreatedAt:    time.Now(),
	}}

	published := time.Date(2026, 11, 18, 10, 30, 0, 0, time.UTC)
	federationService := newFakeFederation(t)
	federationService.interactions = map[string]federation.EntryInteractions{
		"entry-1": {
			Replies: []federation.Interaction{{
				ActorHandle: "bob",
				ActorName:   "Bob Cultivador",
				ActorURL:    "https://remote.example/@bob",
				Content:     "Que tomate bonito!",
				PublishedAt: &published,
			}},
			Likes:     []federation.Interaction{{ActorHandle: "carol"}},
			Announces: []federation.Interaction{{ActorHandle: "dave"}},
		},
	}

	router := newContentTestRouterWithFederation(
		t, newFakeIdentity(), cultivationService, newFakePractice(), nil, federationService,
	)
	response := getPage(router, "/diario/cycle-1")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()
	for _, want := range []string{
		"1 resposta",
		"1 curtida",
		"1 impulso",
		"Respostas",
		"Bob Cultivador",
		"Que tomate bonito!",
		"https://remote.example/@bob",
		"18/11/2026 10:30",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("timeline does not contain %q", want)
		}
	}
}

func TestCycleTimelineRendersLocalReplyAndReplyForm(t *testing.T) {
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro público.",
		Visibility:   cultivation.EntryVisibilityInherit,
		CreatedAt:    time.Now(),
	}}

	published := time.Date(2026, 11, 18, 10, 30, 0, 0, time.UTC)
	federationService := newFakeFederation(t)
	federationService.interactions = map[string]federation.EntryInteractions{
		"entry-1": {
			Replies: []federation.Interaction{
				{
					ID:          "reply-1",
					Direction:   "inbound",
					ActorHandle: "bob",
					ActorName:   "Bob Cultivador",
					ActorURL:    "https://remote.example/@bob",
					Content:     "Que tomate bonito!",
					PublishedAt: &published,
				},
				{
					ID:          "reply-2",
					Direction:   "outbound",
					ActorHandle: "ana",
					ActorName:   "Ana Silva",
					Content:     "Obrigada, Bob!",
					PublishedAt: &published,
				},
			},
		},
	}

	identityService := newFakeIdentity()
	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), nil, federationService,
	)
	session := loginAsAna(t, router, identityService)

	response := getPage(router, "/diario/cycle-1", session)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()
	for _, want := range []string{
		"Sua resposta",
		"Obrigada, Bob!",
		"Responder",
		`action="/diario/cycle-1/registros/entry-1/respostas"`,
		`name="reply_id" value="reply-1"`,
		`maxlength="2000"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("timeline does not contain %q", want)
		}
	}
	if strings.Contains(body, `name="reply_id" value="reply-2"`) {
		t.Error("local reply should not offer another reply form")
	}
}

func TestEntryReplyPublishesResponse(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:         "entry-1",
		CycleID:    "cycle-1",
		AuthorID:   "person-1",
		OccurredAt: time.Now(),
		Body:       "Registro público.",
		Visibility: cultivation.EntryVisibilityInherit,
		CreatedAt:  time.Now(),
	}}

	federationService := newFakeFederation(t)
	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), nil, federationService,
	)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1", session)

	response := postForm(router, "/diario/cycle-1/registros/entry-1/respostas", url.Values{
		"csrf_token": {csrf.Value},
		"reply_id":   {"reply-1"},
		"content":    {"Obrigada, Bob!"},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/diario/cycle-1#registro-entry-1" {
		t.Errorf("Location = %q", location)
	}
	if federationService.replyCalls != 1 {
		t.Fatalf("replyCalls = %d, want 1", federationService.replyCalls)
	}
	if federationService.lastReplyID != "reply-1" {
		t.Errorf("reply id = %q", federationService.lastReplyID)
	}
	if federationService.lastReplyContent != "Obrigada, Bob!" {
		t.Errorf("reply content = %q", federationService.lastReplyContent)
	}
}

func TestEntryReplyRejectsEmptyContent(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:         "entry-1",
		CycleID:    "cycle-1",
		AuthorID:   "person-1",
		OccurredAt: time.Now(),
		Visibility: cultivation.EntryVisibilityInherit,
		CreatedAt:  time.Now(),
	}}

	federationService := newFakeFederation(t)
	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), nil, federationService,
	)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1", session)

	response := postForm(router, "/diario/cycle-1/registros/entry-1/respostas", url.Values{
		"csrf_token": {csrf.Value},
		"reply_id":   {"reply-1"},
		"content":    {"   "},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	if location := response.Header().Get("Location"); location != "/diario/cycle-1?erro_resposta=vazia#registro-entry-1" {
		t.Errorf("Location = %q", location)
	}
	if federationService.replyCalls != 0 {
		t.Errorf("replyCalls = %d, want 0", federationService.replyCalls)
	}
}
