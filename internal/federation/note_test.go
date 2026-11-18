package federation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rede-cultivar/plataform/internal/cultivation"
	"github.com/rede-cultivar/plataform/internal/media"
)

func testCycle() cultivation.Cycle {
	return cultivation.Cycle{
		ID:         "cycle-1",
		PersonID:   testPerson().ID,
		Name:       "Tomates da varanda",
		Species:    "Solanum lycopersicum",
		Visibility: cultivation.VisibilityPublic,
	}
}

func testEntry() cultivation.Entry {
	return cultivation.Entry{
		ID:         "entry-1",
		CycleID:    "cycle-1",
		AuthorID:   testPerson().ID,
		OccurredAt: time.Date(2026, 9, 10, 9, 30, 0, 0, time.UTC),
		CreatedAt:  time.Date(2026, 9, 10, 9, 35, 0, 0, time.UTC),
		Body:       "Primeira flor observada.",
		Visibility: cultivation.EntryVisibilityInherit,
	}
}

func TestNewEntryNote(t *testing.T) {
	note := NewEntryNote(
		NewURLs("https://cultivar.example"),
		testPerson(),
		testCycle(),
		testEntry(),
		nil,
	)

	if note.ID != "https://cultivar.example/notas/entry-1" {
		t.Errorf("ID = %q", note.ID)
	}
	if note.Type != "Note" {
		t.Errorf("Type = %q, want Note", note.Type)
	}
	if note.AttributedTo != "https://cultivar.example/actors/"+testPerson().ID {
		t.Errorf("AttributedTo = %q", note.AttributedTo)
	}
	if note.URL != "https://cultivar.example/diario/cycle-1#registro-entry-1" {
		t.Errorf("URL = %q", note.URL)
	}
	if note.Published != "2026-09-10T09:35:00Z" {
		t.Errorf("Published = %q", note.Published)
	}
	if len(note.To) != 1 || note.To[0] != PublicCollection {
		t.Errorf("To = %v, want public collection", note.To)
	}
	if len(note.CC) != 1 || note.CC[0] != "https://cultivar.example/actors/"+testPerson().ID+"/followers" {
		t.Errorf("CC = %v, want followers collection", note.CC)
	}
	if !strings.Contains(note.Content, "<strong>Tomates da varanda</strong>") {
		t.Errorf("Content = %q, want cycle title", note.Content)
	}
	if !strings.Contains(note.Content, "Solanum lycopersicum") {
		t.Errorf("Content = %q, want species", note.Content)
	}
	if !strings.Contains(note.Content, "Primeira flor observada.") {
		t.Errorf("Content = %q, want entry body", note.Content)
	}
	if len(note.Attachment) != 0 {
		t.Errorf("Attachment = %v, want none", note.Attachment)
	}
}

func TestNewEntryNoteEscapesContent(t *testing.T) {
	cycle := testCycle()
	cycle.Name = `<script>alert("ciclo")</script>`
	cycle.Species = ""

	entry := testEntry()
	entry.Body = "Linha 1 <b>forte</b>\nLinha 2"

	note := NewEntryNote(NewURLs("https://cultivar.example"), testPerson(), cycle, entry, nil)

	if strings.Contains(note.Content, "<script>") || strings.Contains(note.Content, "<b>") {
		t.Errorf("Content = %q, want escaped HTML", note.Content)
	}
	if !strings.Contains(note.Content, "&lt;script&gt;") {
		t.Errorf("Content = %q, want escaped script tag", note.Content)
	}
	if !strings.Contains(note.Content, "&lt;b&gt;forte&lt;/b&gt;") {
		t.Errorf("Content = %q, want escaped bold tag", note.Content)
	}
	if !strings.Contains(note.Content, "Linha 1") || !strings.Contains(note.Content, "Linha 2") {
		t.Errorf("Content = %q, want both lines", note.Content)
	}
	if !strings.Contains(note.Content, "<br>") {
		t.Errorf("Content = %q, want line break", note.Content)
	}
}

func TestNewEntryNoteDefaultsToOccurredAt(t *testing.T) {
	entry := testEntry()
	entry.CreatedAt = time.Time{}

	note := NewEntryNote(NewURLs("https://cultivar.example"), testPerson(), testCycle(), entry, nil)

	if note.Published != "2026-09-10T09:30:00Z" {
		t.Errorf("Published = %q, want occurred_at fallback", note.Published)
	}
}

func TestNewEntryNoteIncludesStageAndHarvest(t *testing.T) {
	stage := cultivation.StageFlowering
	quantity := 2.5

	entry := testEntry()
	entry.Stage = &stage
	entry.HarvestQuantity = &quantity
	entry.HarvestUnit = "kg"

	note := NewEntryNote(NewURLs("https://cultivar.example"), testPerson(), testCycle(), entry, nil)

	if !strings.Contains(note.Content, "Estágio: Floração") {
		t.Errorf("Content = %q, want stage label", note.Content)
	}
	if !strings.Contains(note.Content, "Colheita: 2.5 kg") {
		t.Errorf("Content = %q, want harvest label", note.Content)
	}
}

func TestNewEntryNoteWithoutContent(t *testing.T) {
	cycle := cultivation.Cycle{ID: "cycle-1"}
	entry := cultivation.Entry{ID: "entry-1", CycleID: "cycle-1"}

	note := NewEntryNote(NewURLs("https://cultivar.example"), testPerson(), cycle, entry, nil)

	if !strings.Contains(note.Content, "Novo registro de cultivo.") {
		t.Errorf("Content = %q, want default paragraph", note.Content)
	}
}

func TestNewEntryNoteAttachesImages(t *testing.T) {
	assets := []media.Asset{
		{
			ID:           "asset-1",
			OriginalName: "tomate.png",
			ContentType:  "image/png",
			Width:        800,
			Height:       600,
		},
		{
			ID:           "asset-2",
			OriginalName: "imagem",
			ContentType:  "image/jpeg",
			Width:        640,
			Height:       480,
		},
		{
			ID: "asset-3",
		},
	}

	note := NewEntryNote(NewURLs("https://cultivar.example"), testPerson(), testCycle(), testEntry(), assets)

	if len(note.Attachment) != 2 {
		t.Fatalf("Attachment = %v, want two valid images", note.Attachment)
	}

	first := note.Attachment[0]
	if first.Type != "Image" || first.MediaType != "image/png" {
		t.Errorf("first attachment = %+v", first)
	}
	if first.URL != "https://cultivar.example/media/asset-1" {
		t.Errorf("first URL = %q", first.URL)
	}
	if first.Name != "tomate.png" {
		t.Errorf("first Name = %q, want original filename", first.Name)
	}
	if first.Width != 800 || first.Height != 600 {
		t.Errorf("first dimensions = %dx%d", first.Width, first.Height)
	}

	second := note.Attachment[1]
	if second.Name != "" {
		t.Errorf("second Name = %q, want empty name for generic label", second.Name)
	}
	if second.URL != "https://cultivar.example/media/asset-2" {
		t.Errorf("second URL = %q", second.URL)
	}
}

func TestNewCreate(t *testing.T) {
	urls := NewURLs("https://cultivar.example")
	note := NewEntryNote(urls, testPerson(), testCycle(), testEntry(), nil)

	create := NewCreate(urls, testPerson(), note, "abc123")

	if create.ID != "https://cultivar.example/activities/abc123" {
		t.Errorf("ID = %q", create.ID)
	}
	if create.Type != "Create" {
		t.Errorf("Type = %q, want Create", create.Type)
	}
	if create.Actor != "https://cultivar.example/actors/"+testPerson().ID {
		t.Errorf("Actor = %q", create.Actor)
	}
	if create.Published != note.Published {
		t.Errorf("Published = %q, want %q", create.Published, note.Published)
	}
	if len(create.To) != 1 || create.To[0] != PublicCollection {
		t.Errorf("To = %v", create.To)
	}
	if len(create.CC) != 1 || create.CC[0] != note.CC[0] {
		t.Errorf("CC = %v", create.CC)
	}
	if create.Object.ID != note.ID {
		t.Errorf("Object.ID = %q, want %q", create.Object.ID, note.ID)
	}
}

func TestNewEntryNoteIncludesCycleContext(t *testing.T) {
	cycle := testCycle()
	cycle.Variety = "Santa Cruz"
	cycle.Environment = cultivation.EnvironmentOutdoor
	cycle.Location = "Varanda"
	cycle.StartedOn = time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	entry := testEntry()
	entry.Measurements = map[string]string{"folhas": "8", "altura_cm": "30"}
	entry.Interventions = []string{"rega", "poda"}
	entry.Occurrences = []string{"solo seco"}
	entry.PracticeID = "practice-1"
	entry.PracticeTitle = "Compostagem rápida"

	note := NewEntryNote(NewURLs("https://cultivar.example"), testPerson(), cycle, entry, nil)

	for _, want := range []string{
		"Solanum lycopersicum · Santa Cruz",
		"Externo · Varanda · iniciado em 01/08/2026",
		"10/09/2026",
		"<strong>Medidas</strong>",
		"<li>altura_cm: 30</li><li>folhas: 8</li>",
		"<strong>Intervenções</strong>",
		"<li>rega</li><li>poda</li>",
		"<strong>Ocorrências</strong>",
		"<li>solo seco</li>",
		`<a href="https://cultivar.example/praticas/practice-1">Compostagem rápida</a>`,
		`<a href="https://cultivar.example/diario/cycle-1#registro-entry-1">Ver registro completo</a>`,
	} {
		if !strings.Contains(note.Content, want) {
			t.Errorf("Content = %q, want %q", note.Content, want)
		}
	}
}

func TestNewEntryNoteWithoutCycleNameUsesDefaultTitle(t *testing.T) {
	cycle := cultivation.Cycle{ID: "cycle-1"}
	entry := cultivation.Entry{ID: "entry-1", CycleID: "cycle-1", Body: "Registro simples."}

	note := NewEntryNote(NewURLs("https://cultivar.example"), testPerson(), cycle, entry, nil)

	if !strings.Contains(note.Content, "<strong>Registro de cultivo</strong>") {
		t.Errorf("Content = %q, want default title", note.Content)
	}
}

func TestNewUpdate(t *testing.T) {
	urls := NewURLs("https://cultivar.example")
	note := NewEntryNote(urls, testPerson(), testCycle(), testEntry(), nil)
	note.Updated = "2026-09-10T10:00:00Z"

	update := NewUpdate(urls, testPerson(), note, "abc123")

	if update.ID != "https://cultivar.example/activities/abc123" {
		t.Errorf("ID = %q", update.ID)
	}
	if update.Type != "Update" {
		t.Errorf("Type = %q, want Update", update.Type)
	}
	if update.Actor != urls.Actor(testPerson().ID) {
		t.Errorf("Actor = %q", update.Actor)
	}
	if len(update.To) != 1 || update.To[0] != PublicCollection {
		t.Errorf("To = %v", update.To)
	}
	if len(update.CC) != 1 || update.CC[0] != note.CC[0] {
		t.Errorf("CC = %v", update.CC)
	}
	if update.Object.ID != note.ID || update.Object.Updated != note.Updated {
		t.Errorf("Object = %+v, want note %+v", update.Object, note)
	}
}

func TestNewDelete(t *testing.T) {
	urls := NewURLs("https://cultivar.example")
	note := NewEntryNote(urls, testPerson(), testCycle(), testEntry(), nil)

	deletion := NewDelete(urls, testPerson(), note, "abc123")

	if deletion.ID != "https://cultivar.example/activities/abc123" {
		t.Errorf("ID = %q", deletion.ID)
	}
	if deletion.Type != "Delete" {
		t.Errorf("Type = %q, want Delete", deletion.Type)
	}
	if deletion.Actor != urls.Actor(testPerson().ID) {
		t.Errorf("Actor = %q", deletion.Actor)
	}
	if len(deletion.To) != 1 || deletion.To[0] != PublicCollection {
		t.Errorf("To = %v", deletion.To)
	}
	if deletion.Object.ID != note.ID || deletion.Object.Type != "Tombstone" {
		t.Errorf("Object = %+v, want tombstone of %q", deletion.Object, note.ID)
	}

	payload, err := json.Marshal(deletion)
	if err != nil {
		t.Fatalf("marshal delete: %v", err)
	}
	var decoded struct {
		Type   string `json:"type"`
		Object struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"object"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal delete: %v", err)
	}
	if decoded.Type != "Delete" || decoded.Object.Type != "Tombstone" || decoded.Object.ID != note.ID {
		t.Errorf("payload = %s, want Delete of a Tombstone", payload)
	}
}

func TestNewActivityIDIsOpaqueAndUnique(t *testing.T) {
	first, err := newActivityID()
	if err != nil {
		t.Fatalf("newActivityID() error = %v", err)
	}
	second, err := newActivityID()
	if err != nil {
		t.Fatalf("newActivityID() error = %v", err)
	}

	if first == second {
		t.Errorf("newActivityID() returned the same value twice: %q", first)
	}
	if len(first) != 32 {
		t.Errorf("newActivityID() length = %d, want 32 hex characters", len(first))
	}
}
