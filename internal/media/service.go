package media

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ObjectStore reúne as operações necessárias de armazenamento de objetos.
type ObjectStore interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Remove(ctx context.Context, key string) error
}

// Service reúne as regras de upload e acesso a imagens.
type Service struct {
	store   *Store
	objects ObjectStore
}

// NewService cria o serviço de mídia.
func NewService(db *pgxpool.Pool, objects ObjectStore) *Service {
	return &Service{
		store:   NewStore(db),
		objects: objects,
	}
}

// Upload valida, normaliza e armazena uma imagem da pessoa informada.
func (s *Service) Upload(ctx context.Context, ownerID, filename string, r io.Reader) (*Asset, error) {
	prepared, err := Prepare(r)
	if err != nil {
		return nil, err
	}

	key, err := storageKey(ownerID, prepared.Extension)
	if err != nil {
		return nil, err
	}

	if err := s.objects.Put(ctx, key, bytes.NewReader(prepared.Data), int64(len(prepared.Data)), prepared.ContentType); err != nil {
		return nil, err
	}

	asset := &Asset{
		OwnerID:      ownerID,
		StorageKey:   key,
		OriginalName: safeName(filename),
		ContentType:  prepared.ContentType,
		ByteSize:     int64(len(prepared.Data)),
		Width:        prepared.Width,
		Height:       prepared.Height,
		Status:       StatusReady,
	}

	if err := s.store.Create(ctx, asset); err != nil {
		_ = s.objects.Remove(ctx, key)
		return nil, err
	}

	return asset, nil
}

// AttachToEntry vincula uma mídia a um registro.
func (s *Service) AttachToEntry(ctx context.Context, entryID, mediaID string, position int) error {
	return s.store.AttachToEntry(ctx, entryID, mediaID, position)
}

// ListByEntryIDs carrega as mídias de vários registros.
func (s *Service) ListByEntryIDs(ctx context.Context, entryIDs []string) (map[string][]Asset, error) {
	return s.store.ListByEntryIDs(ctx, entryIDs)
}

// Open devolve os metadados e o conteúdo de uma mídia.
func (s *Service) Open(ctx context.Context, id string) (*Asset, io.ReadCloser, error) {
	asset, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	reader, err := s.objects.Get(ctx, asset.StorageKey)
	if err != nil {
		return nil, nil, err
	}

	return asset, reader, nil
}

// CanView informa se a pessoa pode acessar a mídia.
func (s *Service) CanView(ctx context.Context, id, viewerID string) (bool, error) {
	return s.store.CanView(ctx, id, viewerID)
}

// CanViewFederated informa se a mídia pertence a conteúdo federado.
//
// É usado para validar URLs de capacidade de conteúdo restrito a seguidores.
func (s *Service) CanViewFederated(ctx context.Context, id string) (bool, error) {
	return s.store.CanViewFederated(ctx, id)
}

// Delete remove uma mídia da pessoa responsável.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	key, err := s.store.Delete(ctx, id, ownerID)
	if err != nil {
		return err
	}
	return s.objects.Remove(ctx, key)
}

func storageKey(ownerID, extension string) (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate media key: %w", err)
	}
	return "media/" + ownerID + "/" + hex.EncodeToString(random) + extension, nil
}

func safeName(filename string) string {
	name := filepath.Base(strings.TrimSpace(filename))
	if name == "." || name == "/" || name == "" {
		return "imagem"
	}

	runes := []rune(name)
	if len(runes) > 200 {
		name = string(runes[:200])
	}
	return name
}
