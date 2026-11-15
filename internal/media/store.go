package media

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound indica que a mídia não existe ou não está visível.
var ErrNotFound = errors.New("mídia não encontrada")

// Store implementa o acesso ao banco para metadados de mídia.
type Store struct {
	db *pgxpool.Pool
}

// NewStore cria um repositório de mídia.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

const assetColumns = `
	m.id::text,
	m.owner_id::text,
	m.storage_key,
	m.original_name,
	m.content_type,
	m.byte_size,
	COALESCE(m.width, 0),
	COALESCE(m.height, 0),
	m.status,
	m.created_at
`

// Create insere os metadados de um ativo e preenche identificador e data.
func (s *Store) Create(ctx context.Context, asset *Asset) error {
	return s.db.QueryRow(ctx, `
		INSERT INTO media_assets (
			owner_id, storage_key, original_name, content_type, byte_size,
			width, height, status
		)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, created_at
	`,
		asset.OwnerID,
		asset.StorageKey,
		asset.OriginalName,
		asset.ContentType,
		asset.ByteSize,
		asset.Width,
		asset.Height,
		asset.Status,
	).Scan(&asset.ID, &asset.CreatedAt)
}

// Find busca um ativo pelo identificador.
func (s *Store) Find(ctx context.Context, id string) (*Asset, error) {
	row := s.db.QueryRow(ctx, `
		SELECT `+assetColumns+`
		FROM media_assets m
		WHERE m.id::text = $1
	`, id)

	asset, err := scanAsset(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return asset, nil
}

// AttachToEntry vincula uma mídia a um registro de cultivo.
func (s *Store) AttachToEntry(ctx context.Context, entryID, mediaID string, position int) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO cultivation_entry_media (entry_id, media_id, position)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (entry_id, media_id) DO UPDATE SET position = EXCLUDED.position
	`, entryID, mediaID, position)
	return err
}

// Delete remove um ativo da pessoa responsável e devolve a chave de armazenamento.
func (s *Store) Delete(ctx context.Context, id, ownerID string) (string, error) {
	var key string
	err := s.db.QueryRow(ctx, `
		DELETE FROM media_assets
		WHERE id::text = $1
		  AND owner_id::text = $2
		RETURNING storage_key
	`, id, ownerID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return key, nil
}

// ListByEntryIDs carrega as mídias de vários registros de uma vez.
func (s *Store) ListByEntryIDs(ctx context.Context, entryIDs []string) (map[string][]Asset, error) {
	result := make(map[string][]Asset)
	if len(entryIDs) == 0 {
		return result, nil
	}

	rows, err := s.db.Query(ctx, `
		SELECT em.entry_id::text, `+assetColumns+`
		FROM cultivation_entry_media em
		JOIN media_assets m ON m.id = em.media_id
		WHERE em.entry_id = ANY($1::uuid[])
		ORDER BY em.entry_id, em.position, m.created_at
	`, entryIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entryID string
		asset, err := scanAssetWithPrefix(rows, &entryID)
		if err != nil {
			return nil, err
		}
		result[entryID] = append(result[entryID], *asset)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// CanView informa se a pessoa pode acessar a mídia.
//
// A pessoa responsável sempre pode acessar. As demais só podem ver mídias
// vinculadas a registros ou práticas efetivamente públicas.
func (s *Store) CanView(ctx context.Context, id, viewerID string) (bool, error) {
	var allowed bool
	err := s.db.QueryRow(ctx, `
		SELECT
			EXISTS (
				SELECT 1
				FROM media_assets m
				WHERE m.id::text = $1
				  AND m.owner_id::text = $2
			)
			OR EXISTS (
				SELECT 1
				FROM cultivation_entry_media em
				JOIN cultivation_entries e ON e.id = em.entry_id
				JOIN cultivation_cycles c ON c.id = e.cycle_id
				WHERE em.media_id::text = $1
				  AND c.visibility = 'public'
				  AND c.status <> 'archived'
				  AND e.visibility IN ('inherit', 'public')
			)
			OR EXISTS (
				SELECT 1
				FROM practice_media pm
				JOIN practices p ON p.id = pm.practice_id
				WHERE pm.media_id::text = $1
				  AND p.status = 'published'
				  AND p.visibility = 'public'
			)
	`, id, viewerID).Scan(&allowed)
	return allowed, err
}

func scanAsset(row rowScanner) (*Asset, error) {
	var asset Asset
	err := row.Scan(
		&asset.ID,
		&asset.OwnerID,
		&asset.StorageKey,
		&asset.OriginalName,
		&asset.ContentType,
		&asset.ByteSize,
		&asset.Width,
		&asset.Height,
		&asset.Status,
		&asset.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func scanAssetWithPrefix(row rowScanner, prefix *string) (*Asset, error) {
	var asset Asset
	err := row.Scan(
		prefix,
		&asset.ID,
		&asset.OwnerID,
		&asset.StorageKey,
		&asset.OriginalName,
		&asset.ContentType,
		&asset.ByteSize,
		&asset.Width,
		&asset.Height,
		&asset.Status,
		&asset.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}
