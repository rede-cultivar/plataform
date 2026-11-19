package federation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// RemoteActor representa o documento Actor de outra instância.
type RemoteActor struct {
	ID                string      `json:"id"`
	Type              string      `json:"type"`
	PreferredUsername string      `json:"preferredUsername"`
	Name              string      `json:"name"`
	URL               string      `json:"url"`
	Icon              remoteImage `json:"icon"`
	Image             remoteImage `json:"image"`
	Inbox             string      `json:"inbox"`
	Endpoints         struct {
		SharedInbox string `json:"sharedInbox"`
	} `json:"endpoints"`
	PublicKey struct {
		ID           string `json:"id"`
		Owner        string `json:"owner"`
		PublicKeyPEM string `json:"publicKeyPem"`
	} `json:"publicKey"`
}

// remoteImage aceita tanto uma URL simples quanto um objeto com o campo url,
// formatos usados por diferentes implementações do ActivityPub.
type remoteImage struct {
	URL string `json:"url"`
}

func (i *remoteImage) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err == nil {
		i.URL = raw
		return nil
	}

	var object struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}

	i.URL = object.URL
	return nil
}

// SharedInbox devolve a caixa de entrada compartilhada, quando anunciada.
func (a *RemoteActor) SharedInbox() string {
	if a == nil {
		return ""
	}
	return a.Endpoints.SharedInbox
}

// InboxURL escolhe a caixa compartilhada e recorre à caixa individual.
func (a *RemoteActor) InboxURL() string {
	if a == nil {
		return ""
	}
	if a.Endpoints.SharedInbox != "" {
		return a.Endpoints.SharedInbox
	}
	return a.Inbox
}

// fetchActorDocument busca e interpreta o documento Actor de outra instância.
//
// Diferente de FetchActor, não exige inbox nem chave pública: esses campos só
// são necessários para verificar assinaturas e entregar atividades.
func fetchActorDocument(ctx context.Context, client *http.Client, actorURL string) (*RemoteActor, error) {
	parsed, err := url.Parse(actorURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("%w: endereço de Actor inválido", ErrInvalidActivity)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, actorURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", ActivityContentType+`, application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("buscar Actor remoto: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, fmt.Errorf("%w: status %d", ErrActorGone, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("buscar Actor remoto: status %d", resp.StatusCode)
	}

	var actor RemoteActor
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxRemoteDocumentBytes)).Decode(&actor); err != nil {
		return nil, fmt.Errorf("interpretar Actor remoto: %w", err)
	}

	if strings.TrimSpace(actor.ID) == "" {
		return nil, fmt.Errorf("%w: Actor remoto sem identificador", ErrInvalidActivity)
	}
	if !sameURL(actor.ID, actorURL) {
		return nil, fmt.Errorf("%w: identificador do Actor não confere", ErrInvalidActivity)
	}

	return &actor, nil
}
