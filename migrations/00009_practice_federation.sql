-- +goose Up
-- Interações federadas em práticas.
--
-- Até aqui respostas e reações estavam sempre vinculadas a um registro do
-- diário. A federação de práticas como Article reaproveita as mesmas tabelas,
-- então o alvo passa a ser exclusivo: ou um registro, ou uma prática. O CHECK
-- garante que nenhuma linha fique sem alvo ou aponte para os dois ao mesmo
-- tempo, e os índices parciais mantêm as consultas por prática eficientes.
ALTER TABLE federation_replies
    ALTER COLUMN entry_id DROP NOT NULL,
    ADD COLUMN practice_id UUID REFERENCES practices (id) ON DELETE CASCADE;

ALTER TABLE federation_replies
    ADD CONSTRAINT federation_replies_target_check
        CHECK ((entry_id IS NOT NULL) <> (practice_id IS NOT NULL));

CREATE INDEX federation_replies_practice_idx
    ON federation_replies (practice_id, direction, published_at, created_at);

ALTER TABLE federation_reactions
    ALTER COLUMN entry_id DROP NOT NULL,
    ADD COLUMN practice_id UUID REFERENCES practices (id) ON DELETE CASCADE;

ALTER TABLE federation_reactions
    ADD CONSTRAINT federation_reactions_target_check
        CHECK ((entry_id IS NOT NULL) <> (practice_id IS NOT NULL));

-- A restrição original (entry_id, actor_id, kind) não vale para linhas com
-- entry_id nulo, porque NULLs são distintos. O índice parcial equivalente
-- garante no máximo uma curtida e um impulso por ator em cada prática.
CREATE UNIQUE INDEX federation_reactions_practice_unique
    ON federation_reactions (practice_id, actor_id, kind)
    WHERE practice_id IS NOT NULL;

CREATE INDEX federation_reactions_practice_idx
    ON federation_reactions (practice_id, kind, created_at);

-- +goose Down
DROP INDEX federation_reactions_practice_idx;
DROP INDEX federation_reactions_practice_unique;
DELETE FROM federation_reactions WHERE practice_id IS NOT NULL;
ALTER TABLE federation_reactions
    DROP CONSTRAINT federation_reactions_target_check,
    DROP COLUMN practice_id,
    ALTER COLUMN entry_id SET NOT NULL;

DROP INDEX federation_replies_practice_idx;
DELETE FROM federation_replies WHERE practice_id IS NOT NULL;
ALTER TABLE federation_replies
    DROP CONSTRAINT federation_replies_target_check,
    DROP COLUMN practice_id,
    ALTER COLUMN entry_id SET NOT NULL;
