package federation

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/url"
	"strings"
)

// URLs reúne os endereços públicos canônicos usados na federação.
//
// Os identificadores de Actor são estáveis e opacos, no formato
// https://{domínio}/actors/{id}, conforme o ADR 0009.
//
// O segredo é usado para derivar tokens de capacidade para conteúdo visível
// apenas a seguidores. Sem ele, os endereços continuam funcionando, mas sem a
// proteção adicional usada em produção.
type URLs struct {
	base   string
	secret string
}

// NewURLs cria o conjunto de URLs a partir da URL pública da instância.
func NewURLs(baseURL string) URLs {
	return URLs{base: strings.TrimSuffix(strings.TrimSpace(baseURL), "/")}
}

// NewURLsWithSecret cria o conjunto de URLs com o segredo usado nos tokens de
// capacidade.
//
// O segredo deve ser o mesmo em todos os processos que geram e validam links de
// conteúdo restrito a seguidores.
func NewURLsWithSecret(baseURL, secret string) URLs {
	return URLs{
		base:   strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		secret: secret,
	}
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

// Key retorna o identificador da chave pública do Actor.
func (u URLs) Key(id string) string {
	return u.Actor(id) + "#main-key"
}

// Note retorna a URL canônica de uma publicação federada.
func (u URLs) Note(id string) string {
	return u.base + "/notas/" + url.PathEscape(id)
}

// NoteWithToken retorna a URL de uma publicação restrita a seguidores.
//
// O token funciona como uma URL de capacidade: quem recebeu a publicação pode
// acessá-la, mesmo sem uma sessão ou assinatura HTTP. O identificador do objeto
// muda em relação à versão pública, então transições de visibilidade trocam o
// objeto federado.
func (u URLs) NoteWithToken(id string) string {
	token := u.capabilityToken("note", id)
	if token == "" {
		return u.Note(id)
	}
	return u.Note(id) + "?token=" + url.QueryEscape(token)
}

// NoteToken devolve o token de capacidade de uma publicação.
func (u URLs) NoteToken(id string) string {
	return u.capabilityToken("note", id)
}

// ValidNoteToken informa se o token apresentado libera a publicação.
func (u URLs) ValidNoteToken(id, token string) bool {
	return validCapability(u.capabilityToken("note", id), token)
}

// ReplyNote retorna a URL canônica de uma resposta publicada pelo Cultivar.
//
// O namespace é separado de /notas para que um identificador de resposta nunca
// seja confundido com o identificador de um registro de cultivo.
func (u URLs) ReplyNote(id string) string {
	return u.base + "/respostas/" + url.PathEscape(id)
}

// Entry retorna a página pública de um registro de cultivo.
func (u URLs) Entry(cycleID, entryID string) string {
	return u.base + "/diario/" + url.PathEscape(cycleID) + "#registro-" + url.PathEscape(entryID)
}

// Activity retorna a URL canônica de uma atividade federada.
func (u URLs) Activity(id string) string {
	return u.base + "/activities/" + url.PathEscape(id)
}

// Media retorna a URL pública de uma imagem anexada a uma publicação.
func (u URLs) Media(id string) string {
	return u.base + "/media/" + url.PathEscape(id)
}

// MediaWithToken retorna a URL de uma imagem restrita a seguidores.
func (u URLs) MediaWithToken(id string) string {
	token := u.capabilityToken("media", id)
	if token == "" {
		return u.Media(id)
	}
	return u.Media(id) + "?token=" + url.QueryEscape(token)
}

// MediaToken devolve o token de capacidade de uma imagem.
func (u URLs) MediaToken(id string) string {
	return u.capabilityToken("media", id)
}

// ValidMediaToken informa se o token apresentado libera a imagem.
func (u URLs) ValidMediaToken(id, token string) bool {
	return validCapability(u.capabilityToken("media", id), token)
}

// capabilityToken deriva um token estável para um recurso.
//
// A derivação usa HMAC para que o token não possa ser calculado sem o segredo
// da instância. Como o identificador entra na mensagem, cada publicação e cada
// imagem têm um token próprio.
//
// Sem segredo configurado, nenhum token é gerado. O conteúdo restrito continua
// inacessível remotamente, em vez de usar um segredo vazio e previsível.
func (u URLs) capabilityToken(resource, id string) string {
	if u.secret == "" {
		return ""
	}

	mac := hmac.New(sha256.New, []byte(u.secret))
	mac.Write([]byte(resource))
	mac.Write([]byte{0})
	mac.Write([]byte(id))
	return hex.EncodeToString(mac.Sum(nil))
}

func validCapability(expected, provided string) bool {
	provided = strings.TrimSpace(provided)
	if expected == "" || provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}

// Practice retorna a página pública de uma prática.
func (u URLs) Practice(id string) string {
	return u.base + "/praticas/" + url.PathEscape(id)
}

// Profile retorna a página pública do perfil.
func (u URLs) Profile(handle string) string {
	return u.base + "/perfil/" + url.PathEscape(handle)
}
