package federation

import (
	"html"
	"strings"
	"time"

	"github.com/rede-cultivar/plataform/internal/identity"
	"github.com/rede-cultivar/plataform/internal/practices"
)

// Article representa uma prática publicada no Fediverso.
//
// Diferente da Note de um registro, o Article descreve conhecimento
// reproduzível: título, resumo, materiais e passo a passo. O campo Name leva o
// título e o corpo HTML reúne o restante. Summary é omitido de propósito: no
// Mastodon ele é interpretado como aviso de conteúdo (CW), não como resumo.
type Article struct {
	Context      []string `json:"@context"`
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	AttributedTo string   `json:"attributedTo"`
	Name         string   `json:"name"`
	Content      string   `json:"content"`
	Published    string   `json:"published"`
	Updated      string   `json:"updated,omitempty"`
	URL          string   `json:"url"`
	To           []string `json:"to"`
	CC           []string `json:"cc,omitempty"`
}

// ArticleCreate distribui um Article aos seguidores.
type ArticleCreate struct {
	Context   []string `json:"@context"`
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	Actor     string   `json:"actor"`
	Published string   `json:"published"`
	To        []string `json:"to"`
	CC        []string `json:"cc,omitempty"`
	Object    Article  `json:"object"`
}

// ArticleUpdate redistribui um Article já federado.
type ArticleUpdate struct {
	Context []string `json:"@context"`
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Actor   string   `json:"actor"`
	To      []string `json:"to"`
	CC      []string `json:"cc,omitempty"`
	Object  Article  `json:"object"`
}

// ArticleDelete remove uma prática federada.
type ArticleDelete struct {
	Context []string  `json:"@context"`
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Actor   string    `json:"actor"`
	To      []string  `json:"to"`
	CC      []string  `json:"cc,omitempty"`
	Object  Tombstone `json:"object"`
}

// NewPracticeArticle traduz uma prática para um Article.
//
// Práticas só são federadas quando estão publicadas e públicas. A função
// continua montando um objeto seguro para reconstruir a publicação anterior
// durante a remoção.
func NewPracticeArticle(urls URLs, person identity.Person, practice practices.Practice) Article {
	published := practice.CreatedAt
	if practice.PublishedAt != nil && !practice.PublishedAt.IsZero() {
		published = *practice.PublishedAt
	}

	article := Article{
		Context:      []string{activityStreamsContext},
		ID:           urls.Article(practice.ID),
		Type:         "Article",
		AttributedTo: urls.Actor(person.ID),
		Name:         practiceTitle(practice),
		Content:      practiceContent(urls, practice),
		URL:          urls.Practice(practice.ID),
		To:           []string{PublicCollection},
		CC:           []string{urls.Followers(person.ID)},
	}
	if !published.IsZero() {
		article.Published = published.UTC().Format(time.RFC3339)
	}

	return article
}

// NewArticleCreate monta a atividade Create que distribui o Article.
func NewArticleCreate(urls URLs, person identity.Person, article Article, activityID string) ArticleCreate {
	return ArticleCreate{
		Context:   []string{activityStreamsContext},
		ID:        urls.Activity(activityID),
		Type:      "Create",
		Actor:     urls.Actor(person.ID),
		Published: article.Published,
		To:        article.To,
		CC:        article.CC,
		Object:    article,
	}
}

// NewArticleUpdate monta a atividade Update que redistribui o Article.
func NewArticleUpdate(urls URLs, person identity.Person, article Article, activityID string) ArticleUpdate {
	return ArticleUpdate{
		Context: []string{activityStreamsContext},
		ID:      urls.Activity(activityID),
		Type:    "Update",
		Actor:   urls.Actor(person.ID),
		To:      article.To,
		CC:      article.CC,
		Object:  article,
	}
}

// NewArticleDelete monta a atividade que remove a prática do Fediverso.
func NewArticleDelete(urls URLs, person identity.Person, article Article, activityID string) ArticleDelete {
	return ArticleDelete{
		Context: []string{activityStreamsContext},
		ID:      urls.Activity(activityID),
		Type:    "Delete",
		Actor:   urls.Actor(person.ID),
		To:      article.To,
		CC:      article.CC,
		Object: Tombstone{
			ID:   article.ID,
			Type: "Tombstone",
		},
	}
}

func practiceTitle(practice practices.Practice) string {
	title := strings.TrimSpace(practice.Title)
	if title == "" {
		return "Prática de cultivo"
	}
	return title
}

// practiceContent monta o corpo HTML do Article.
//
// O texto é escapado e as quebras de linha viram <br>, como no cartão dos
// registros. A dificuldade usa o rótulo em português e o link final leva à
// página completa da prática.
func practiceContent(urls URLs, practice practices.Practice) string {
	parts := make([]string, 0, 8)

	if summary := strings.TrimSpace(practice.Summary); summary != "" {
		parts = append(parts, "<p>"+practiceText(summary)+"</p>")
	}

	meta := make([]string, 0, 4)
	if category := strings.TrimSpace(practice.Category); category != "" {
		meta = append(meta, html.EscapeString(category))
	}
	if practice.Difficulty.Valid() {
		meta = append(meta, "Dificuldade: "+html.EscapeString(practice.Difficulty.Label()))
	}
	if estimate := strings.TrimSpace(practice.TimeEstimate); estimate != "" {
		meta = append(meta, "Tempo estimado: "+html.EscapeString(estimate))
	}
	if cost := strings.TrimSpace(practice.CostEstimate); cost != "" {
		meta = append(meta, "Custo aproximado: "+html.EscapeString(cost))
	}
	if len(meta) > 0 {
		parts = append(parts, "<p>"+strings.Join(meta, " · ")+"</p>")
	}

	if list := entryList("Materiais e ferramentas", practice.Materials); list != "" {
		parts = append(parts, list)
	}
	if list := practiceSteps(practice.Steps); list != "" {
		parts = append(parts, list)
	}
	if safety := strings.TrimSpace(practice.Safety); safety != "" {
		parts = append(parts,
			"<p><strong>Avisos de segurança e riscos</strong></p><p>"+practiceText(safety)+"</p>",
		)
	}
	if adaptations := strings.TrimSpace(practice.Adaptations); adaptations != "" {
		parts = append(parts,
			"<p><strong>Adaptações e variações</strong></p><p>"+practiceText(adaptations)+"</p>",
		)
	}
	if references := strings.TrimSpace(practice.References); references != "" {
		parts = append(parts,
			"<p><strong>Referências</strong></p><p>"+practiceText(references)+"</p>",
		)
	}
	if license := strings.TrimSpace(practice.License); license != "" {
		parts = append(parts, "<p>Licença: "+html.EscapeString(license)+"</p>")
	}

	if len(parts) == 0 {
		parts = append(parts, "<p>Prática de cultivo.</p>")
	}

	parts = append(parts,
		`<p><a href="`+html.EscapeString(urls.Practice(practice.ID))+`">Ver prática completa</a></p>`,
	)

	return strings.Join(parts, "")
}

// practiceText escapa o texto e preserva as quebras de linha.
func practiceText(value string) string {
	escaped := html.EscapeString(strings.TrimSpace(value))
	escaped = strings.ReplaceAll(escaped, "\r\n", "\n")
	return strings.ReplaceAll(escaped, "\n", "<br>")
}

// practiceSteps monta a lista ordenada do passo a passo.
func practiceSteps(items []string) string {
	clean := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			clean = append(clean, item)
		}
	}
	if len(clean) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("<p><strong>Passo a passo</strong></p><ol>")
	for _, item := range clean {
		builder.WriteString("<li>" + html.EscapeString(item) + "</li>")
	}
	builder.WriteString("</ol>")
	return builder.String()
}
