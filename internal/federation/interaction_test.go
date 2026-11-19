package federation

import (
	"strings"
	"testing"
	"time"
)

func TestEntryIDFromNoteURL(t *testing.T) {
	urls := NewURLs("https://cultivar.example")
	const entryID = "e026bd4f-d80d-466c-80e4-d2704905373e"

	cases := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{"canônica", urls.Note(entryID), entryID, true},
		{"barra final", urls.Note(entryID) + "/", entryID, true},
		{"outro host", "https://other.example/notas/" + entryID, "", false},
		{"outro caminho", "https://cultivar.example/actors/" + entryID, "", false},
		{"identificador inválido", "https://cultivar.example/notas/abc", "", false},
		{"travessia de caminho", "https://cultivar.example/notas/../" + entryID, "", false},
		{"vazio", "", "", false},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, ok := entryIDFromNoteURL(urls, test.input)
			if ok != test.ok || got != test.want {
				t.Fatalf("entryIDFromNoteURL(%q) = %q, %v; want %q, %v", test.input, got, ok, test.want, test.ok)
			}
		})
	}
}

func TestPracticeIDFromArticleURL(t *testing.T) {
	urls := NewURLs("https://cultivar.example")
	const practiceID = "e026bd4f-d80d-466c-80e4-d2704905373e"

	cases := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{"canônica", urls.Article(practiceID), practiceID, true},
		{"barra final", urls.Article(practiceID) + "/", practiceID, true},
		{"outro host", "https://other.example/artigos/" + practiceID, "", false},
		{"outro caminho", "https://cultivar.example/notas/" + practiceID, "", false},
		{"identificador inválido", "https://cultivar.example/artigos/abc", "", false},
		{"vazio", "", "", false},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, ok := practiceIDFromArticleURL(urls, test.input)
			if ok != test.ok || got != test.want {
				t.Fatalf("practiceIDFromArticleURL(%q) = %q, %v; want %q, %v", test.input, got, ok, test.want, test.ok)
			}
		})
	}
}

func TestPlainText(t *testing.T) {
	input := `<p>Olá, <strong>mundo</strong>!</p><p>Segunda linha &amp; mais.</p>`
	want := "Olá, mundo!\nSegunda linha & mais."

	if got := plainText(input); got != want {
		t.Fatalf("plainText() = %q, want %q", got, want)
	}
}

func TestPlainTextTruncatesByRunes(t *testing.T) {
	got := plainText(strings.Repeat("á", maxInteractionContent+10))

	runes := []rune(got)
	if len(runes) != maxInteractionContent+1 {
		t.Fatalf("runas = %d, want %d", len(runes), maxInteractionContent+1)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("texto truncado não termina em reticências: %q", got)
	}
}

func TestParseReplyObject(t *testing.T) {
	published := "2026-11-18T10:30:00Z"
	object := map[string]any{
		"id":           "https://remote.example/notes/1",
		"type":         "Note",
		"inReplyTo":    "https://cultivar.example/notas/e026bd4f-d80d-466c-80e4-d2704905373e",
		"attributedTo": map[string]any{"id": "https://remote.example/users/bob"},
		"content":      "<p>Que tomate bonito!</p>",
		"published":    published,
	}

	payload, ok := parseReplyObject(object)
	if !ok {
		t.Fatal("parseReplyObject() não reconheceu a Note")
	}
	if payload.ObjectID != "https://remote.example/notes/1" {
		t.Errorf("ObjectID = %q", payload.ObjectID)
	}
	if payload.AttributedTo != "https://remote.example/users/bob" {
		t.Errorf("AttributedTo = %q", payload.AttributedTo)
	}
	if payload.Content != "Que tomate bonito!" {
		t.Errorf("Content = %q", payload.Content)
	}
	if payload.PublishedAt == nil || !payload.PublishedAt.Equal(mustParseTime(t, published)) {
		t.Errorf("PublishedAt = %v", payload.PublishedAt)
	}
}

func TestParseReplyObjectRejectsOtherTypes(t *testing.T) {
	if _, ok := parseReplyObject(map[string]any{"type": "Article"}); ok {
		t.Fatal("parseReplyObject() aceitou um Article")
	}
	if _, ok := parseReplyObject("https://remote.example/notes/1"); ok {
		t.Fatal("parseReplyObject() aceitou uma string")
	}
}

func TestEntryInteractionSummaries(t *testing.T) {
	empty := EntryInteractions{}
	if empty.HasAny() {
		t.Error("HasAny() = true sem interações")
	}
	if empty.LikeSummary() != "" || empty.AnnounceSummary() != "" || empty.ReplySummary() != "" {
		t.Error("resumos deveriam ficar vazios sem interações")
	}

	value := EntryInteractions{
		Replies:   []Interaction{{}, {}},
		Likes:     []Interaction{{}},
		Announces: []Interaction{{}, {}, {}},
	}
	if !value.HasAny() {
		t.Error("HasAny() = false com interações")
	}
	if got := value.ReplySummary(); got != "2 respostas" {
		t.Errorf("ReplySummary() = %q", got)
	}
	if got := value.LikeSummary(); got != "1 curtida" {
		t.Errorf("LikeSummary() = %q", got)
	}
	if got := value.AnnounceSummary(); got != "3 impulsos" {
		t.Errorf("AnnounceSummary() = %q", got)
	}
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}
	return parsed
}
