package federation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rede-cultivar/plataform/internal/practices"
)

func testPractice() practices.Practice {
	published := time.Date(2026, 9, 12, 8, 0, 0, 0, time.UTC)
	return practices.Practice{
		ID:           "22222222-2222-2222-2222-222222222222",
		AuthorID:     testPerson().ID,
		Title:        "Composteira de balde",
		Summary:      "Uma composteira simples para apartamentos.",
		Category:     "Compostagem",
		Difficulty:   practices.DifficultyBeginner,
		TimeEstimate: "1 hora",
		CostEstimate: "R$ 40",
		Materials:    []string{"Balde de 20 L", "Terra"},
		Steps:        []string{"Faça furos no balde.", "Alterne camadas."},
		Safety:       "Evite carne e laticínios.",
		Adaptations:  "Funciona em varandas.",
		References:   "Manual de compostagem.",
		License:      "CC BY-SA 4.0",
		Version:      1,
		Status:       practices.StatusPublished,
		Visibility:   practices.VisibilityPublic,
		PublishedAt:  &published,
		CreatedAt:    published,
	}
}

func TestNewPracticeArticle(t *testing.T) {
	practice := testPractice()
	article := NewPracticeArticle(NewURLs("https://cultivar.example"), testPerson(), practice)

	if article.ID != "https://cultivar.example/artigos/"+practice.ID {
		t.Errorf("ID = %q", article.ID)
	}
	if article.Type != "Article" {
		t.Errorf("Type = %q, want Article", article.Type)
	}
	if article.AttributedTo != "https://cultivar.example/actors/"+testPerson().ID {
		t.Errorf("AttributedTo = %q", article.AttributedTo)
	}
	if article.Name != practice.Title {
		t.Errorf("Name = %q, want %q", article.Name, practice.Title)
	}
	if article.URL != "https://cultivar.example/praticas/"+practice.ID {
		t.Errorf("URL = %q", article.URL)
	}
	if article.Published != "2026-09-12T08:00:00Z" {
		t.Errorf("Published = %q", article.Published)
	}
	if len(article.To) != 1 || article.To[0] != PublicCollection {
		t.Errorf("To = %v, want public collection", article.To)
	}
	if len(article.CC) != 1 || article.CC[0] != "https://cultivar.example/actors/"+testPerson().ID+"/followers" {
		t.Errorf("CC = %v, want followers collection", article.CC)
	}

	content := article.Content
	for _, want := range []string{
		practice.Summary,
		"Compostagem",
		"Dificuldade: Iniciante",
		"Tempo estimado: 1 hora",
		"Custo aproximado: R$ 40",
		"Materiais e ferramentas",
		"Balde de 20 L",
		"Passo a passo",
		"<ol>",
		"Faça furos no balde.",
		"Avisos de segurança e riscos",
		practice.Safety,
		"Adaptações e variações",
		"Referências",
		"Licença: CC BY-SA 4.0",
		`href="https://cultivar.example/praticas/` + practice.ID + `"`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("Content = %q, want substring %q", content, want)
		}
	}
}

func TestNewPracticeArticleEscapesContent(t *testing.T) {
	practice := testPractice()
	practice.Title = `<script>alert("prática")</script>`
	practice.Summary = "Linha 1 <b>forte</b>\nLinha 2"
	practice.Steps = []string{`<img src=x onerror="alert(1)">`}

	article := NewPracticeArticle(NewURLs("https://cultivar.example"), testPerson(), practice)

	if article.Name != practice.Title {
		t.Errorf("Name = %q, want raw title text", article.Name)
	}
	if strings.Contains(article.Content, "<script>") || strings.Contains(article.Content, "<b>") {
		t.Errorf("Content = %q, want escaped HTML", article.Content)
	}
	if !strings.Contains(article.Content, "&lt;b&gt;forte&lt;/b&gt;") {
		t.Errorf("Content = %q, want escaped bold tag", article.Content)
	}
	if !strings.Contains(article.Content, "<br>") {
		t.Errorf("Content = %q, want line break", article.Content)
	}
	if strings.Contains(article.Content, "<img") {
		t.Errorf("Content = %q, want escaped step", article.Content)
	}
	if !strings.Contains(article.Content, "&lt;img") {
		t.Errorf("Content = %q, want escaped image tag", article.Content)
	}
}

func TestNewPracticeArticleDefaultTitle(t *testing.T) {
	practice := testPractice()
	practice.Title = "  "

	article := NewPracticeArticle(NewURLs("https://cultivar.example"), testPerson(), practice)

	if article.Name != "Prática de cultivo" {
		t.Errorf("Name = %q, want default title", article.Name)
	}
}

func TestNewPracticeArticleWithoutPublishedAtUsesCreatedAt(t *testing.T) {
	practice := testPractice()
	practice.PublishedAt = nil

	article := NewPracticeArticle(NewURLs("https://cultivar.example"), testPerson(), practice)

	if article.Published != "2026-09-12T08:00:00Z" {
		t.Errorf("Published = %q, want CreatedAt fallback", article.Published)
	}
}

func TestNewArticleActivities(t *testing.T) {
	practice := testPractice()
	urls := NewURLs("https://cultivar.example")
	article := NewPracticeArticle(urls, testPerson(), practice)

	create := NewArticleCreate(urls, testPerson(), article, "activity-1")
	if create.Type != "Create" || create.ID != urls.Activity("activity-1") {
		t.Errorf("Create = %+v", create)
	}
	if create.Object.ID != article.ID {
		t.Errorf("Create.Object.ID = %q, want %q", create.Object.ID, article.ID)
	}

	update := NewArticleUpdate(urls, testPerson(), article, "activity-2")
	if update.Type != "Update" || update.Object.ID != article.ID {
		t.Errorf("Update = %+v", update)
	}

	deletion := NewArticleDelete(urls, testPerson(), article, "activity-3")
	if deletion.Type != "Delete" {
		t.Errorf("Delete.Type = %q", deletion.Type)
	}
	if deletion.Object.ID != article.ID || deletion.Object.Type != "Tombstone" {
		t.Errorf("Delete.Object = %+v", deletion.Object)
	}

	payload, err := json.Marshal(create)
	if err != nil {
		t.Fatalf("marshal create: %v", err)
	}
	if !strings.Contains(string(payload), `"type":"Create"`) {
		t.Errorf("payload = %s, want Create type", payload)
	}
}
