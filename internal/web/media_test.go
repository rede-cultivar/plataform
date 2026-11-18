package web

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rede-cultivar/plataform/internal/cultivation"
	"github.com/rede-cultivar/plataform/internal/identity"
	"github.com/rede-cultivar/plataform/internal/media"
)

// fakeMedia implementa MediaService em memória para os testes HTTP.
type fakeMedia struct {
	assets        map[string]*media.Asset
	data          map[string][]byte
	attachments   map[string][]string
	publicEntries map[string]bool
	uploadErr     error
	nextID        int
}

func newFakeMedia() *fakeMedia {
	return &fakeMedia{
		assets:        make(map[string]*media.Asset),
		data:          make(map[string][]byte),
		attachments:   make(map[string][]string),
		publicEntries: make(map[string]bool),
	}
}

func (f *fakeMedia) Upload(_ context.Context, ownerID, filename string, r io.Reader) (*media.Asset, error) {
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}

	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	f.nextID++
	asset := &media.Asset{
		ID:           fmt.Sprintf("asset-%d", f.nextID),
		OwnerID:      ownerID,
		OriginalName: filename,
		ContentType:  "image/png",
		ByteSize:     int64(len(content)),
		Width:        1,
		Height:       1,
		Status:       media.StatusReady,
	}
	f.assets[asset.ID] = asset
	f.data[asset.ID] = content
	return asset, nil
}

func (f *fakeMedia) AttachToEntry(_ context.Context, entryID, mediaID string, _ int) error {
	f.attachments[entryID] = append(f.attachments[entryID], mediaID)
	return nil
}

func (f *fakeMedia) ListByEntryIDs(_ context.Context, entryIDs []string) (map[string][]media.Asset, error) {
	result := make(map[string][]media.Asset)
	for _, entryID := range entryIDs {
		for _, mediaID := range f.attachments[entryID] {
			if asset, ok := f.assets[mediaID]; ok {
				result[entryID] = append(result[entryID], *asset)
			}
		}
	}
	return result, nil
}

func (f *fakeMedia) Open(_ context.Context, id string) (*media.Asset, io.ReadCloser, error) {
	asset, ok := f.assets[id]
	if !ok {
		return nil, nil, media.ErrNotFound
	}
	return asset, io.NopCloser(bytes.NewReader(f.data[id])), nil
}

func (f *fakeMedia) CanView(_ context.Context, id, viewerID string) (bool, error) {
	asset, ok := f.assets[id]
	if !ok {
		return false, nil
	}
	if viewerID != "" && asset.OwnerID == viewerID {
		return true, nil
	}

	for entryID, ids := range f.attachments {
		if !f.publicEntries[entryID] {
			continue
		}
		for _, mediaID := range ids {
			if mediaID == id {
				return true, nil
			}
		}
	}
	return false, nil
}

func (f *fakeMedia) Delete(_ context.Context, ownerID, id string) error {
	asset, ok := f.assets[id]
	if !ok || asset.OwnerID != ownerID {
		return media.ErrNotFound
	}

	delete(f.assets, id)
	delete(f.data, id)
	for entryID, ids := range f.attachments {
		filtered := ids[:0]
		for _, mediaID := range ids {
			if mediaID != id {
				filtered = append(filtered, mediaID)
			}
		}
		f.attachments[entryID] = filtered
	}
	return nil
}

// testPNG devolve uma imagem PNG válida de 1x1 pixel.
func testPNG(t *testing.T) []byte {
	t.Helper()

	data, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")
	if err != nil {
		t.Fatalf("decode test png: %v", err)
	}
	return data
}

type uploadFile struct {
	Name string
	Data []byte
}

func postMultipart(router http.Handler, target string, fields map[string]string, files []uploadFile, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for name, value := range fields {
		_ = writer.WriteField(name, value)
	}
	for _, file := range files {
		part, err := writer.CreateFormFile("photos", file.Name)
		if err != nil {
			panic(err)
		}
		_, _ = part.Write(file.Data)
	}
	_ = writer.Close()

	request := httptest.NewRequest(http.MethodPost, target, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestCreateEntryUploadsPhotoAndServesIt(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	mediaService := newFakeMedia()

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice(), mediaService)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/registros/novo", session)

	response := postMultipart(router, "/diario/cycle-1/registros", map[string]string{
		"csrf_token":  csrf.Value,
		"occurred_at": "2026-09-10T09:30",
		"body":        "Primeira flor observada.",
		"visibility":  "inherit",
	}, []uploadFile{{Name: "flor.png", Data: testPNG(t)}}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/diario/cycle-1#registro-entry-1" {
		t.Errorf("Location = %q, want timeline anchor", location)
	}

	if len(mediaService.assets) != 1 {
		t.Fatalf("assets = %d, want 1", len(mediaService.assets))
	}
	attached := mediaService.attachments["entry-1"]
	if len(attached) != 1 || attached[0] != "asset-1" {
		t.Fatalf("attachments = %v, want [asset-1]", attached)
	}

	// Um ciclo público permite que a imagem seja servida a visitantes.
	mediaService.publicEntries["entry-1"] = true

	timeline := getPage(router, "/diario/cycle-1")
	if timeline.Code != http.StatusOK {
		t.Fatalf("timeline status = %d, want %d", timeline.Code, http.StatusOK)
	}
	if body := timeline.Body.String(); !strings.Contains(body, `src="/media/asset-1"`) {
		t.Error("timeline does not render the uploaded photo")
	}

	image := getPage(router, "/media/asset-1")
	if image.Code != http.StatusOK {
		t.Fatalf("media status = %d, want %d; body = %s", image.Code, http.StatusOK, image.Body.String())
	}
	if contentType := image.Header().Get("Content-Type"); contentType != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", contentType)
	}
	if !bytes.Equal(image.Body.Bytes(), testPNG(t)) {
		t.Error("served image differs from the uploaded content")
	}
}

func TestEntryEditCanDeleteAndAddPhotos(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cycle := sampleCycle("cycle-1", cultivation.VisibilityPrivate)
	cultivationService.cycles["cycle-1"] = cycle
	stage := cultivation.StageVegetative
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro inicial.",
		Stage:        &stage,
		Visibility:   cultivation.EntryVisibilityInherit,
		CreatedAt:    time.Now(),
	}}
	mediaService := newFakeMedia()
	mediaService.nextID = 1
	mediaService.assets["asset-1"] = &media.Asset{ID: "asset-1", OwnerID: "person-1", OriginalName: "antiga.png", ContentType: "image/png", Width: 1, Height: 1, Status: media.StatusReady}
	mediaService.data["asset-1"] = testPNG(t)
	mediaService.attachments["entry-1"] = []string{"asset-1"}

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice(), mediaService)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/registros/entry-1/editar", session)

	response := postMultipart(router, "/diario/cycle-1/registros/entry-1/editar", map[string]string{
		"csrf_token":   csrf.Value,
		"occurred_at":  "2026-09-10T09:30",
		"body":         "Registro atualizado.",
		"visibility":   "public",
		"delete_photo": "asset-1",
	}, []uploadFile{{Name: "nova.png", Data: testPNG(t)}}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if _, ok := mediaService.assets["asset-1"]; ok {
		t.Fatal("old image still exists after deletion")
	}
	if len(mediaService.attachments["entry-1"]) != 1 {
		t.Fatalf("attachments after update = %v, want exactly 1 remaining image", mediaService.attachments["entry-1"])
	}
	if got := cultivationService.entries["cycle-1"][0].Body; got != "Registro atualizado." {
		t.Errorf("updated body = %q, want Registro atualizado.", got)
	}
}

func TestEntryPhotoDeleteRemovesAttachedPhoto(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPrivate)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro com foto.",
		Visibility:   cultivation.EntryVisibilityInherit,
		CreatedAt:    time.Now(),
	}}
	mediaService := newFakeMedia()
	mediaService.nextID = 1
	mediaService.assets["asset-1"] = &media.Asset{ID: "asset-1", OwnerID: "person-1", OriginalName: "antiga.png", ContentType: "image/png", Width: 1, Height: 1, Status: media.StatusReady}
	mediaService.data["asset-1"] = testPNG(t)
	mediaService.attachments["entry-1"] = []string{"asset-1"}

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice(), mediaService)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1", session)

	timeline := getPage(router, "/diario/cycle-1", session)
	if timeline.Code != http.StatusOK {
		t.Fatalf("timeline status = %d, want %d", timeline.Code, http.StatusOK)
	}
	if body := timeline.Body.String(); !strings.Contains(body, "/diario/cycle-1/registros/entry-1/fotos/asset-1/apagar") {
		t.Error("timeline does not render the photo delete action")
	}

	response := postForm(router, "/diario/cycle-1/registros/entry-1/fotos/asset-1/apagar", url.Values{
		"csrf_token": {csrf.Value},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/diario/cycle-1#registro-entry-1" {
		t.Errorf("Location = %q, want timeline anchor", location)
	}
	if _, ok := mediaService.assets["asset-1"]; ok {
		t.Error("photo still exists after deletion")
	}
	if attachments := mediaService.attachments["entry-1"]; len(attachments) != 0 {
		t.Errorf("attachments = %v, want none", attachments)
	}
}

func TestEntryPhotoDeleteFederatesUpdate(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPublic)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{{
		ID:           "entry-1",
		CycleID:      "cycle-1",
		AuthorID:     "person-1",
		AuthorHandle: "ana",
		OccurredAt:   time.Now(),
		Body:         "Registro com foto.",
		Visibility:   cultivation.EntryVisibilityInherit,
		CreatedAt:    time.Now(),
	}}
	mediaService := newFakeMedia()
	mediaService.nextID = 1
	mediaService.assets["asset-1"] = &media.Asset{ID: "asset-1", OwnerID: "person-1", OriginalName: "antiga.png", ContentType: "image/png", Width: 1, Height: 1, Status: media.StatusReady}
	mediaService.attachments["entry-1"] = []string{"asset-1"}
	federationService := newFakeFederation(t)

	router := newContentTestRouterWithFederation(
		t, identityService, cultivationService, newFakePractice(), mediaService, federationService,
	)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1", session)

	response := postForm(router, "/diario/cycle-1/registros/entry-1/fotos/asset-1/apagar", url.Values{
		"csrf_token": {csrf.Value},
	}, session, csrf)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}
	if federationService.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", federationService.updateCalls)
	}
	if federationService.lastUpdate.ID != "entry-1" {
		t.Errorf("updated entry = %q, want entry-1", federationService.lastUpdate.ID)
	}
	if len(federationService.lastUpdateAssets) != 0 {
		t.Errorf("update assets = %v, want none", federationService.lastUpdateAssets)
	}
}

func TestEntryPhotoDeleteRejectsNonOwner(t *testing.T) {
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
	mediaService := newFakeMedia()
	mediaService.assets["asset-1"] = &media.Asset{ID: "asset-1", OwnerID: "person-1", OriginalName: "foto.png", ContentType: "image/png", Width: 1, Height: 1, Status: media.StatusReady}
	mediaService.data["asset-1"] = testPNG(t)
	mediaService.attachments["entry-1"] = []string{"asset-1"}

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice(), mediaService)
	session := loginAsBruno(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1", session)

	response := postForm(router, "/diario/cycle-1/registros/entry-1/fotos/asset-1/apagar", url.Values{
		"csrf_token": {csrf.Value},
	}, session, csrf)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusForbidden, response.Body.String())
	}
	if _, ok := mediaService.assets["asset-1"]; !ok {
		t.Error("photo was deleted by a non-owner")
	}
}

func TestEntryPhotoDeleteRejectsPhotoFromAnotherEntry(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPrivate)
	cultivationService.entries["cycle-1"] = []cultivation.Entry{
		{ID: "entry-1", CycleID: "cycle-1", AuthorID: "person-1", AuthorHandle: "ana", OccurredAt: time.Now(), Visibility: cultivation.EntryVisibilityInherit, CreatedAt: time.Now()},
		{ID: "entry-2", CycleID: "cycle-1", AuthorID: "person-1", AuthorHandle: "ana", OccurredAt: time.Now(), Visibility: cultivation.EntryVisibilityInherit, CreatedAt: time.Now()},
	}
	mediaService := newFakeMedia()
	mediaService.assets["asset-1"] = &media.Asset{ID: "asset-1", OwnerID: "person-1", OriginalName: "foto.png", ContentType: "image/png", Width: 1, Height: 1, Status: media.StatusReady}
	mediaService.data["asset-1"] = testPNG(t)
	mediaService.attachments["entry-2"] = []string{"asset-1"}

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice(), mediaService)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1", session)

	response := postForm(router, "/diario/cycle-1/registros/entry-1/fotos/asset-1/apagar", url.Values{
		"csrf_token": {csrf.Value},
	}, session, csrf)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNotFound, response.Body.String())
	}
	if _, ok := mediaService.assets["asset-1"]; !ok {
		t.Error("photo from another entry was deleted")
	}
}

// loginAsBruno autentica uma segunda pessoa para testar permissões.
func loginAsBruno(t *testing.T, router http.Handler, fake *fakeIdentity) *http.Cookie {
	t.Helper()

	person := identity.Person{
		ID:          "person-2",
		Handle:      "bruno",
		DisplayName: "Bruno Souza",
		CreatedAt:   time.Now(),
	}
	account := &identity.Account{
		ID:        "account-2",
		Email:     "bruno@example.com",
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

func TestMediaHiddenForPrivateCycle(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPrivate)
	mediaService := newFakeMedia()

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice(), mediaService)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/registros/novo", session)

	response := postMultipart(router, "/diario/cycle-1/registros", map[string]string{
		"csrf_token":  csrf.Value,
		"occurred_at": "2026-09-10T09:30",
		"body":        "Registro privado.",
		"visibility":  "inherit",
	}, []uploadFile{{Name: "flor.png", Data: testPNG(t)}}, session, csrf)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusSeeOther, response.Body.String())
	}

	anonymous := getPage(router, "/media/asset-1")
	if anonymous.Code != http.StatusNotFound {
		t.Fatalf("anonymous status = %d, want %d", anonymous.Code, http.StatusNotFound)
	}

	owner := getPage(router, "/media/asset-1", session)
	if owner.Code != http.StatusOK {
		t.Fatalf("owner status = %d, want %d", owner.Code, http.StatusOK)
	}
}

func TestCreateEntryReportsInvalidPhoto(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPrivate)
	mediaService := newFakeMedia()
	mediaService.uploadErr = &media.ValidationError{Field: "photos", Message: "O arquivo enviado não é uma imagem."}

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice(), mediaService)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/registros/novo", session)

	response := postMultipart(router, "/diario/cycle-1/registros", map[string]string{
		"csrf_token":  csrf.Value,
		"occurred_at": "2026-09-10T09:30",
		"body":        "Registro com arquivo inválido.",
		"visibility":  "inherit",
	}, []uploadFile{{Name: "arquivo.txt", Data: []byte("não é uma imagem")}}, session, csrf)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if body := response.Body.String(); !strings.Contains(body, "O arquivo enviado não é uma imagem.") {
		t.Error("response does not contain the media validation message")
	}
	if entries := cultivationService.entries["cycle-1"]; len(entries) != 0 {
		t.Errorf("entries = %d, want 0", len(entries))
	}
}

func TestCreateEntryCleansUpPhotosWhenEntryFails(t *testing.T) {
	identityService := newFakeIdentity()
	cultivationService := newFakeCultivation()
	cultivationService.cycles["cycle-1"] = sampleCycle("cycle-1", cultivation.VisibilityPrivate)
	cultivationService.entryErr = &cultivation.ValidationError{Field: "body", Message: "O registro deve ter no máximo 5000 caracteres."}
	mediaService := newFakeMedia()

	router := newContentTestRouter(t, identityService, cultivationService, newFakePractice(), mediaService)
	session := loginAsAna(t, router, identityService)
	csrf := csrfFor(t, router, "/diario/cycle-1/registros/novo", session)

	response := postMultipart(router, "/diario/cycle-1/registros", map[string]string{
		"csrf_token":  csrf.Value,
		"occurred_at": "2026-09-10T09:30",
		"body":        "Registro longo.",
		"visibility":  "inherit",
	}, []uploadFile{{Name: "flor.png", Data: testPNG(t)}}, session, csrf)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if len(mediaService.assets) != 0 {
		t.Errorf("assets = %d, want 0 after cleanup", len(mediaService.assets))
	}
}
