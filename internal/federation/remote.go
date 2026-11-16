package federation

// RemoteActor representa o documento Actor de outra instância.
type RemoteActor struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	PreferredUsername string `json:"preferredUsername"`
	Inbox             string `json:"inbox"`
	Endpoints         struct {
		SharedInbox string `json:"sharedInbox"`
	} `json:"endpoints"`
	PublicKey struct {
		ID           string `json:"id"`
		Owner        string `json:"owner"`
		PublicKeyPEM string `json:"publicKeyPem"`
	} `json:"publicKey"`
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
