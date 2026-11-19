-- +goose Up
-- Relacionamento local de acompanhamento entre pessoas da instância.
--
-- O vínculo é imediato, sem pedido de aprovação: ao seguir alguém, a pessoa
-- passa a acessar os ciclos e registros com visibilidade "followers" dessa
-- pessoa. Seguidores remotos continuam registrados em federation_follows.
CREATE TABLE person_follows (
    follower_id UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    followed_id UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (follower_id, followed_id),
    CONSTRAINT person_follows_distinct_people CHECK (follower_id <> followed_id)
);

CREATE INDEX person_follows_followed_idx ON person_follows (followed_id);

-- +goose Down
DROP TABLE person_follows;
