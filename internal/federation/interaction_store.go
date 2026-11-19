package federation

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// FederatedEntryOwner devolve a pessoa responsável e a visibilidade efetiva de
// um registro que pode ser federado.
//
// A consulta considera registros públicos e registros visíveis apenas para
// seguidores. Interações para conteúdo privado são ignoradas para não criar
// vínculos com informação restrita.
func (s *Store) FederatedEntryOwner(ctx context.Context, entryID string) (string, string, error) {
	var personID, visibility string
	err := s.db.QueryRow(ctx, `
		SELECT
			c.person_id::text,
			CASE
				WHEN c.visibility = 'public'
				  AND e.visibility IN ('inherit', 'public') THEN 'public'
				ELSE 'followers'
			END
		FROM cultivation_entries e
		JOIN cultivation_cycles c ON c.id = e.cycle_id
		WHERE e.id::text = $1
		  AND c.visibility <> 'private'
		  AND e.visibility <> 'private'
	`, entryID).Scan(&personID, &visibility)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", err
	}
	return personID, visibility, nil
}

// FederatedPracticeOwner devolve a pessoa responsável por uma prática que pode
// ser federada.
//
// Práticas só são federadas quando estão publicadas e públicas; rascunhos e
// práticas arquivadas não recebem interações.
func (s *Store) FederatedPracticeOwner(ctx context.Context, practiceID string) (string, error) {
	var personID string
	err := s.db.QueryRow(ctx, `
		SELECT author_id::text
		FROM practices
		WHERE id::text = $1
		  AND status = 'published'
		  AND visibility = 'public'
	`, practiceID).Scan(&personID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return personID, nil
}

// IsRemoteFollower informa se um Actor remoto segue a pessoa e teve o Follow
// aceito.
//
// Interações em conteúdo restrito só são aceitas de seguidores confirmados.
func (s *Store) IsRemoteFollower(ctx context.Context, personID, actorID string) (bool, error) {
	var following bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM federation_follows
			WHERE person_id = $1::uuid
			  AND actor_id = $2
			  AND accepted_at IS NOT NULL
		)
	`, personID, actorID).Scan(&following)
	return following, err
}

// UpsertReply grava ou atualiza uma resposta recebida em um registro.
//
// O object_id é a Note remota e funciona como chave de idempotência: uma
// reentrega do Create atualiza a mesma linha em vez de duplicar a resposta.
func (s *Store) UpsertReply(ctx context.Context, personID, entryID string, reply Interaction) error {
	return s.upsertReply(ctx, personID, &entryID, nil, reply)
}

// UpsertPracticeReply grava ou atualiza uma resposta recebida em uma prática.
func (s *Store) UpsertPracticeReply(ctx context.Context, personID, practiceID string, reply Interaction) error {
	return s.upsertReply(ctx, personID, nil, &practiceID, reply)
}

func (s *Store) upsertReply(ctx context.Context, personID string, entryID, practiceID *string, reply Interaction) error {
	direction := reply.Direction
	if direction == "" {
		direction = "inbound"
	}
	parentID := reply.ParentID
	if parentID == "" {
		parentID = reply.InReplyTo
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO federation_replies (
			person_id, entry_id, practice_id, direction, activity_id, object_id, actor_id,
			actor_handle, actor_name, actor_url, actor_avatar, actor_inbox,
			in_reply_to, parent_object_id, content, published_at
		)
		VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16
		)
		ON CONFLICT (object_id) DO UPDATE
		SET activity_id = EXCLUDED.activity_id,
		    actor_handle = EXCLUDED.actor_handle,
		    actor_name = EXCLUDED.actor_name,
		    actor_url = EXCLUDED.actor_url,
		    actor_avatar = EXCLUDED.actor_avatar,
		    actor_inbox = EXCLUDED.actor_inbox,
		    in_reply_to = EXCLUDED.in_reply_to,
		    parent_object_id = EXCLUDED.parent_object_id,
		    content = EXCLUDED.content,
		    published_at = EXCLUDED.published_at,
		    updated_at = now()
	`, personID, entryID, practiceID, direction, reply.ActivityID, reply.ObjectID, reply.ActorID,
		reply.ActorHandle, reply.ActorName, reply.ActorURL, reply.ActorAvatar, reply.ActorInbox,
		reply.InReplyTo, parentID, reply.Content, reply.PublishedAt)
	return err
}

// InsertOutboundReply grava uma resposta publicada por uma pessoa local.
//
// O object_id é a URL /respostas/{id} e identifica a Note de saída. A inserção
// não sobrescreve uma linha existente: o identificador é opaco e gerado antes da
// publicação.
func (s *Store) InsertOutboundReply(ctx context.Context, personID, entryID string, reply Interaction) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO federation_replies (
			person_id, entry_id, direction, activity_id, object_id, actor_id,
			actor_handle, actor_name, actor_url, actor_avatar, actor_inbox,
			in_reply_to, parent_object_id, content, published_at
		)
		VALUES (
			$1::uuid, $2::uuid, 'outbound', $3, $4, $5,
			$6, $7, $8, '', '',
			$9, $10, $11, $12
		)
		ON CONFLICT (object_id) DO NOTHING
	`, personID, entryID, reply.ActivityID, reply.ObjectID, reply.ActorID,
		reply.ActorHandle, reply.ActorName, reply.ActorURL,
		reply.InReplyTo, reply.ParentID, reply.Content, reply.PublishedAt)
	return err
}

// ReplyRecord reúne os dados de uma resposta usados para publicar uma nova
// mensagem no mesmo fio.
type ReplyRecord struct {
	ID          string
	EntryID     string
	Direction   string
	ObjectID    string
	ActorID     string
	ActorHandle string
	ActorName   string
	ActorURL    string
	ActorInbox  string
	InReplyTo   string
}

// FindReply busca a resposta recebida que será respondida.
//
// A consulta exige que a linha pertença à pessoa e ao registro informados e que
// tenha vindo do Fediverso, evitando que uma resposta local seja usada como
// origem de outra resposta.
func (s *Store) FindReply(ctx context.Context, personID, entryID, replyID string) (*ReplyRecord, error) {
	var record ReplyRecord
	err := s.db.QueryRow(ctx, `
		SELECT
			id::text, entry_id::text, direction, object_id, actor_id,
			actor_handle, actor_name, actor_url, actor_inbox, in_reply_to
		FROM federation_replies
		WHERE id::text = $1
		  AND person_id = $2::uuid
		  AND entry_id = $3::uuid
		  AND direction = 'inbound'
	`, replyID, personID, entryID).Scan(
		&record.ID,
		&record.EntryID,
		&record.Direction,
		&record.ObjectID,
		&record.ActorID,
		&record.ActorHandle,
		&record.ActorName,
		&record.ActorURL,
		&record.ActorInbox,
		&record.InReplyTo,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateReply atualiza o texto de uma resposta já conhecida.
func (s *Store) UpdateReply(ctx context.Context, actorID, objectID, content string, publishedAt *time.Time) error {
	_, err := s.db.Exec(ctx, `
		UPDATE federation_replies
		SET content = $3,
		    published_at = COALESCE($4, published_at),
		    updated_at = now()
		WHERE actor_id = $1
		  AND object_id = $2
	`, actorID, objectID, content, publishedAt)
	return err
}

// DeleteReply remove uma resposta pelo identificador da Note remota.
func (s *Store) DeleteReply(ctx context.Context, actorID, objectID string) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM federation_replies
		WHERE actor_id = $1
		  AND object_id = $2
	`, actorID, objectID)
	return err
}

// UpsertReaction grava ou atualiza uma curtida ou impulso em um registro.
//
// A unicidade por registro, ator e tipo evita contagens duplicadas quando a
// instância remota reenvia a atividade.
func (s *Store) UpsertReaction(ctx context.Context, personID, entryID string, reaction Interaction) error {
	return s.upsertReaction(ctx, personID, &entryID, nil, reaction)
}

// UpsertPracticeReaction grava ou atualiza uma curtida ou impulso em uma
// prática.
func (s *Store) UpsertPracticeReaction(ctx context.Context, personID, practiceID string, reaction Interaction) error {
	return s.upsertReaction(ctx, personID, nil, &practiceID, reaction)
}

func (s *Store) upsertReaction(ctx context.Context, personID string, entryID, practiceID *string, reaction Interaction) error {
	// O alvo muda o índice de unicidade: registros usam a restrição original e
	// práticas usam o índice parcial criado na migração 00009.
	conflict := `
		ON CONFLICT (entry_id, actor_id, kind) DO UPDATE
		SET activity_id = EXCLUDED.activity_id,
		    object_id = EXCLUDED.object_id,
		    actor_handle = EXCLUDED.actor_handle,
		    actor_name = EXCLUDED.actor_name,
		    actor_url = EXCLUDED.actor_url,
		    actor_avatar = EXCLUDED.actor_avatar,
		    updated_at = now()
	`
	if practiceID != nil {
		conflict = `
			ON CONFLICT (practice_id, actor_id, kind) WHERE practice_id IS NOT NULL DO UPDATE
			SET activity_id = EXCLUDED.activity_id,
			    object_id = EXCLUDED.object_id,
			    actor_handle = EXCLUDED.actor_handle,
			    actor_name = EXCLUDED.actor_name,
			    actor_url = EXCLUDED.actor_url,
			    actor_avatar = EXCLUDED.actor_avatar,
			    updated_at = now()
		`
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO federation_reactions (
			person_id, entry_id, practice_id, kind, activity_id, object_id, actor_id,
			actor_handle, actor_name, actor_url, actor_avatar
		)
		VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7,
			$8, $9, $10, $11
		)
	`+conflict, personID, entryID, practiceID, string(reaction.Kind), reaction.ActivityID, reaction.ObjectID, reaction.ActorID,
		reaction.ActorHandle, reaction.ActorName, reaction.ActorURL, reaction.ActorAvatar)
	return err
}

// DeleteReaction remove a reação de um ator em um registro.
func (s *Store) DeleteReaction(ctx context.Context, actorID, entryID string, kind InteractionKind) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM federation_reactions
		WHERE actor_id = $1
		  AND entry_id::text = $2
		  AND kind = $3
	`, actorID, entryID, string(kind))
	return err
}

// DeletePracticeReaction remove a reação de um ator em uma prática.
func (s *Store) DeletePracticeReaction(ctx context.Context, actorID, practiceID string, kind InteractionKind) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM federation_reactions
		WHERE actor_id = $1
		  AND practice_id::text = $2
		  AND kind = $3
	`, actorID, practiceID, string(kind))
	return err
}

// DeleteReactionByActivity remove uma reação pelo identificador da atividade.
//
// É usado quando o Undo referencia a atividade Like/Announce apenas por URL, sem
// o objeto embutido.
func (s *Store) DeleteReactionByActivity(ctx context.Context, actorID, activityID string) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM federation_reactions
		WHERE actor_id = $1
		  AND activity_id = $2
	`, actorID, activityID)
	return err
}

// RemoveActorInteractions apaga respostas e reações de uma conta excluída.
func (s *Store) RemoveActorInteractions(ctx context.Context, actorID string) error {
	if _, err := s.db.Exec(ctx, `
		DELETE FROM federation_replies WHERE actor_id = $1
	`, actorID); err != nil {
		return err
	}

	_, err := s.db.Exec(ctx, `
		DELETE FROM federation_reactions WHERE actor_id = $1
	`, actorID)
	return err
}

// ListEntryInteractions carrega respostas e reações de vários registros.
//
// O resultado é indexado pelo identificador do registro. Registros sem
// interações simplesmente não aparecem no mapa.
func (s *Store) ListEntryInteractions(ctx context.Context, entryIDs []string) (map[string]EntryInteractions, error) {
	return s.listInteractions(ctx, "entry_id", entryIDs)
}

// ListPracticeInteractions carrega respostas e reações de várias práticas.
//
// O resultado é indexado pelo identificador da prática.
func (s *Store) ListPracticeInteractions(ctx context.Context, practiceIDs []string) (map[string]EntryInteractions, error) {
	return s.listInteractions(ctx, "practice_id", practiceIDs)
}

func (s *Store) listInteractions(ctx context.Context, targetColumn string, targetIDs []string) (map[string]EntryInteractions, error) {
	result := make(map[string]EntryInteractions)
	if len(targetIDs) == 0 {
		return result, nil
	}

	replies, err := s.listReplies(ctx, targetColumn, targetIDs)
	if err != nil {
		return nil, err
	}
	for _, reply := range replies {
		value := result[reply.EntryID]
		value.Replies = append(value.Replies, reply)
		result[reply.EntryID] = value
	}

	reactions, err := s.listReactions(ctx, targetColumn, targetIDs)
	if err != nil {
		return nil, err
	}
	for _, reaction := range reactions {
		value := result[reaction.EntryID]
		switch reaction.Kind {
		case InteractionLike:
			value.Likes = append(value.Likes, reaction)
		case InteractionAnnounce:
			value.Announces = append(value.Announces, reaction)
		}
		result[reaction.EntryID] = value
	}

	return result, nil
}

func (s *Store) listReplies(ctx context.Context, targetColumn string, targetIDs []string) ([]Interaction, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			id::text, COALESCE(entry_id, practice_id)::text, direction, activity_id, object_id, actor_id,
			actor_handle, actor_name, actor_url, actor_avatar, actor_inbox,
			in_reply_to, parent_object_id, content, published_at, created_at
		FROM federation_replies
		WHERE `+targetColumn+` = ANY($1::uuid[])
		ORDER BY COALESCE(published_at, created_at), created_at
	`, targetIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	replies := make([]Interaction, 0)
	for rows.Next() {
		var reply Interaction
		var publishedAt *time.Time
		if err := rows.Scan(
			&reply.ID,
			&reply.EntryID,
			&reply.Direction,
			&reply.ActivityID,
			&reply.ObjectID,
			&reply.ActorID,
			&reply.ActorHandle,
			&reply.ActorName,
			&reply.ActorURL,
			&reply.ActorAvatar,
			&reply.ActorInbox,
			&reply.InReplyTo,
			&reply.ParentID,
			&reply.Content,
			&publishedAt,
			&reply.CreatedAt,
		); err != nil {
			return nil, err
		}
		reply.Kind = InteractionReply
		reply.PublishedAt = publishedAt
		replies = append(replies, reply)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return replies, nil
}

func (s *Store) listReactions(ctx context.Context, targetColumn string, targetIDs []string) ([]Interaction, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			id::text, COALESCE(entry_id, practice_id)::text, kind, activity_id, object_id, actor_id,
			actor_handle, actor_name, actor_url, actor_avatar, created_at
		FROM federation_reactions
		WHERE `+targetColumn+` = ANY($1::uuid[])
		ORDER BY created_at
	`, targetIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reactions := make([]Interaction, 0)
	for rows.Next() {
		var reaction Interaction
		var kind string
		if err := rows.Scan(
			&reaction.ID,
			&reaction.EntryID,
			&kind,
			&reaction.ActivityID,
			&reaction.ObjectID,
			&reaction.ActorID,
			&reaction.ActorHandle,
			&reaction.ActorName,
			&reaction.ActorURL,
			&reaction.ActorAvatar,
			&reaction.CreatedAt,
		); err != nil {
			return nil, err
		}
		reaction.Kind = InteractionKind(kind)
		reactions = append(reactions, reaction)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reactions, nil
}
