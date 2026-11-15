package federation

import (
	"net/url"
	"strings"
)

// URLs reúne os endereços públicos canônicos usados na federação.
//
// Os identificadores de Actor são estáveis e opacos, no formato
// https://{domínio}/actors/{id}, conforme o ADR 0009.
type URLs struct {
	base string
}

// NewURLs cria o conjunto de URLs a partir da URL pública da instância.
func NewURLs(baseURL string) URLs {
	return URLs{base: strings.TrimSuffix(strings.TrimSpace(baseURL), "/")}
}

// Base retorna a URL pública da instância sem barra final.
func (u URLs) Base() string {
	return u.base
}

// Host retorna o host público, incluindo a porta quando houver.
func (u URLs) Host() string {
	parsed, err := url.Parse(u.base)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Host)
}

// Actor retorna a URL canônica do Actor.
func (u URLs) Actor(id string) string {
	return u.base + "/actors/" + url.PathEscape(id)
}

// Inbox retorna a caixa de entrada do Actor.
func (u URLs) Inbox(id string) string {
	return u.Actor(id) + "/inbox"
}

// Outbox retorna a caixa de saída do Actor.
func (u URLs) Outbox(id string) string {
	return u.Actor(id) + "/outbox"
}

// Followers retorna a coleção de seguidores do Actor.
func (u URLs) Followers(id string) string {
	return u.Actor(id) + "/followers"
}

// Following retorna a coleção de pessoas seguidas pelo Actor.
func (u URLs) Following(id string) string {
	return u.Actor(id) + "/following"
}

// Profile retorna a página pública do perfil.
func (u URLs) Profile(handle string) string {
	return u.base + "/perfil/" + url.PathEscape(handle)
}
