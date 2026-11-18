-- +goose Up
-- Interações recebidas do Fediverso, vinculadas a um registro de cultivo.
--
-- Respostas e reações ficam em tabelas separadas porque têm ciclos de vida
-- diferentes: uma pessoa pode responder várias vezes ao mesmo registro, mas
-- cada ator só pode ter uma curtida e um impulso por registro.
CREATE TABLE federation_replies (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id    UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    entry_id     UUID NOT NULL REFERENCES cultivation_entries (id) ON DELETE CASCADE,
    activity_id  TEXT NOT NULL UNIQUE,
    object_id    TEXT NOT NULL UNIQUE,
    actor_id     TEXT NOT NULL,
    actor_handle TEXT NOT NULL DEFAULT '',
    actor_name   TEXT NOT NULL DEFAULT '',
    actor_url    TEXT NOT NULL DEFAULT '',
    actor_avatar TEXT NOT NULL DEFAULT '',
    in_reply_to  TEXT NOT NULL DEFAULT '',
    content      TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX federation_replies_entry_idx
    ON federation_replies (entry_id, published_at, created_at);

CREATE TABLE federation_reactions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id    UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    entry_id     UUID NOT NULL REFERENCES cultivation_entries (id) ON DELETE CASCADE,
    kind         TEXT NOT NULL CHECK (kind IN ('like', 'announce')),
    activity_id  TEXT NOT NULL UNIQUE,
    object_id    TEXT NOT NULL DEFAULT '',
    actor_id     TEXT NOT NULL,
    actor_handle TEXT NOT NULL DEFAULT '',
    actor_name   TEXT NOT NULL DEFAULT '',
    actor_url    TEXT NOT NULL DEFAULT '',
    actor_avatar TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entry_id, actor_id, kind)
);

CREATE INDEX federation_reactions_entry_idx
    ON federation_reactions (entry_id, kind, created_at);

-- +goose Down
DROP TABLE federation_reactions;
DROP TABLE federation_replies;
