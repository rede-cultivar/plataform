package federation

import (
	"strings"

	"github.com/rede-cultivar/plataform/internal/identity"
)

// WebFingerLink representa um link de um documento WebFinger.
type WebFingerLink struct {
	Rel  string `json:"rel"`
	Type string `json:"type,omitempty"`
	Href string `json:"href,omitempty"`
}

// WebFinger representa a resposta JRD do WebFinger.
type WebFinger struct {
	Subject string          `json:"subject"`
	Aliases []string        `json:"aliases,omitempty"`
	Links   []WebFingerLink `json:"links"`
}

// NewWebFinger monta a resposta de descoberta de uma pessoa.
func NewWebFinger(urls URLs, host string, person identity.Person) WebFinger {
	actorURL := urls.Actor(person.ID)
	profileURL := urls.Profile(person.Handle)

	return WebFinger{
		Subject: "acct:" + person.Handle + "@" + host,
		Aliases: []string{profileURL, actorURL},
		Links: []WebFingerLink{
			{
				Rel:  "http://webfinger.net/rel/profile-page",
				Type: "text/html",
				Href: profileURL,
			},
			{
				Rel:  "self",
				Type: "application/activity+json",
				Href: actorURL,
			},
		},
	}
}

// ParseAcctResource extrai o identificador e o domínio de um recurso acct:.
//
// O formato aceito é acct:identificador@domínio, com ou sem o caractere @
// inicial no identificador.
func ParseAcctResource(resource string) (handle, host string, ok bool) {
	resource = strings.TrimSpace(resource)
	if !strings.HasPrefix(strings.ToLower(resource), "acct:") {
		return "", "", false
	}

	value := resource[len("acct:"):]
	value = strings.TrimPrefix(value, "@")

	at := strings.LastIndex(value, "@")
	if at <= 0 || at == len(value)-1 {
		return "", "", false
	}

	return identity.NormalizeHandle(value[:at]), strings.ToLower(value[at+1:]), true
}
