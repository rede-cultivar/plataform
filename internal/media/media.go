package media

import "time"

// Situações possíveis de um ativo de mídia.
const (
	StatusPending = "pending"
	StatusReady   = "ready"
	StatusFailed  = "failed"
)

// Asset representa uma imagem armazenada e seus metadados.
type Asset struct {
	ID           string
	OwnerID      string
	StorageKey   string
	OriginalName string
	ContentType  string
	ByteSize     int64
	Width        int
	Height       int
	Status       string
	CreatedAt    time.Time
}
