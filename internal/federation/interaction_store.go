package federation

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// PublicEntryOwner devolve a pessoa responsável por um registro público.
//
// A consulta só considera registros que podem ser federados: ciclo público e
// visibilidade do registro herdada ou pública. Interações para outros registros
// são ignoradas para não criar vínculos com conteúdo privado.
func (s *Store) PublicEntryOwner(ctx context.Context, entryID string) (string, error) {
	var personID string
	err := s.db.QueryRow(ctx, `
		SELECT c.person_id::text
		FROM cultivation_entries e
		JOIN cultivation_cycles c ON c.id = e.cycle_id
		WHERE e.id::text = $1
		  AND c.visibility = 'public'
		  AND e.visibility IN ('inherit', 'public')
	`, entryID).Scan(&personID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return personID, nil
}

// UpsertReply grava ou atualiza uma resposta recebida.
//
// O object_id é a Note remota e funciona como chave de idempotência: uma
// reentrega do Create atualiza a mesma linha em vez de duplicar a resposta.
func (s *Store) UpsertReply(ctx context.Context, personID, entryID string, reply Interaction) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO federation_replies (
			person_id, entry_id, activity_id, object_id, actor_id,
			actor_handle, actor_name, actor_url, actor_avatar,
			in_reply_to, content, published_at
		)
		VALUES (
			$1::uuid, $2::uuid, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12
		)
		ON CONFLICT (object_id) DO UPDATE
		SET activity_id = EXCLUDED.activity_id,
		    actor_handle = EXCLUDED.actor_handle,
		    actor_name = EXCLUDED.actor_name,
		    actor_url = EXCLUDED.actor_url,
		    actor_avatar = EXCLUDED.actor_avatar,
		    in_reply_to = EXCLUDED.in_reply_to,
		    content = EXCLUDED.content,
		    published_at = EXCLUDED.published_at,
		    updated_at = now()
	`, personID, entryID, reply.ActivityID, reply.ObjectID, reply.ActorID,
		reply.ActorHandle, reply.ActorName, reply.ActorURL, reply.ActorAvatar,
		reply.InReplyTo, reply.Content, reply.PublishedAt)
	return err
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

// UpsertReaction grava ou atualiza uma curtida ou impulso.
//
// A unicidade por registro, ator e tipo evita contagens duplicadas quando a
// instância remota reenvia a atividade.
func (s *Store) UpsertReaction(ctx context.Context, personID, entryID string, reaction Interaction) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO federation_reactions (
			person_id, entry_id, kind, activity_id, object_id, actor_id,
			actor_handle, actor_name, actor_url, actor_avatar
		)
		VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6,
			$7, $8, $9, $10
		)
		ON CONFLICT (entry_id, actor_id, kind) DO UPDATE
		SET activity_id = EXCLUDED.activity_id,
		    object_id = EXCLUDED.object_id,
		    actor_handle = EXCLUDED.actor_handle,
		    actor_name = EXCLUDED.actor_name,
		    actor_url = EXCLUDED.actor_url,
		    actor_avatar = EXCLUDED.actor_avatar,
		    updated_at = now()
	`, personID, entryID, string(reaction.Kind), reaction.ActivityID, reaction.ObjectID, reaction.ActorID,
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
	result := make(map[string]EntryInteractions)
	if len(entryIDs) == 0 {
		return result, nil
	}

	replies, err := s.listReplies(ctx, entryIDs)
	if err != nil {
		return nil, err
	}
	for _, reply := range replies {
		value := result[reply.EntryID]
		value.Replies = append(value.Replies, reply)
		result[reply.EntryID] = value
	}

	reactions, err := s.listReactions(ctx, entryIDs)
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

func (s *Store) listReplies(ctx context.Context, entryIDs []string) ([]Interaction, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			id::text, entry_id::text, activity_id, object_id, actor_id,
			actor_handle, actor_name, actor_url, actor_avatar,
			in_reply_to, content, published_at, created_at
		FROM federation_replies
		WHERE entry_id = ANY($1::uuid[])
		ORDER BY COALESCE(published_at, created_at), created_at
	`, entryIDs)
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
			&reply.ActivityID,
			&reply.ObjectID,
			&reply.ActorID,
			&reply.ActorHandle,
			&reply.ActorName,
			&reply.ActorURL,
			&reply.ActorAvatar,
			&reply.InReplyTo,
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

func (s *Store) listReactions(ctx context.Context, entryIDs []string) ([]Interaction, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			id::text, entry_id::text, kind, activity_id, object_id, actor_id,
			actor_handle, actor_name, actor_url, actor_avatar, created_at
		FROM federation_reactions
		WHERE entry_id = ANY($1::uuid[])
		ORDER BY created_at
	`, entryIDs)
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
