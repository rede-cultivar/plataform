-- +goose Up
-- Respostas publicadas a partir do Cultivar e dados necessários para entregá-las.
--
-- Até aqui a tabela guardava apenas respostas recebidas. A coluna direction
-- separa o que chegou do Fediverso do que foi publicado por uma pessoa local, e
-- actor_inbox guarda a caixa de entrada do autor respondido para que a entrega
-- não dependa de uma nova consulta ao Actor remoto.
ALTER TABLE federation_replies
    ADD COLUMN direction TEXT NOT NULL DEFAULT 'inbound'
        CHECK (direction IN ('inbound', 'outbound')),
    ADD COLUMN parent_object_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN actor_inbox TEXT NOT NULL DEFAULT '';

CREATE INDEX federation_replies_direction_idx
    ON federation_replies (entry_id, direction, published_at, created_at);

-- +goose Down
DROP INDEX federation_replies_direction_idx;
ALTER TABLE federation_replies
    DROP COLUMN actor_inbox,
    DROP COLUMN parent_object_id,
    DROP COLUMN direction;
