-- +goose Up
CREATE TABLE actor_keys (
    person_id   UUID PRIMARY KEY REFERENCES people (id) ON DELETE CASCADE,
    public_key  TEXT NOT NULL,
    private_key TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE federation_follows (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id        UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    actor_id         TEXT NOT NULL,
    inbox_url        TEXT NOT NULL,
    shared_inbox_url TEXT NOT NULL DEFAULT '',
    accepted_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (person_id, actor_id)
);

CREATE INDEX federation_follows_person_idx ON federation_follows (person_id);

CREATE TABLE federation_activities (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id     UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    activity_id   TEXT NOT NULL UNIQUE,
    activity_type TEXT NOT NULL,
    object_id     TEXT NOT NULL DEFAULT '',
    payload       JSONB NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    visibility    TEXT NOT NULL DEFAULT 'public'
                  CHECK (visibility IN ('private', 'followers', 'public')),
    direction     TEXT NOT NULL DEFAULT 'outbound'
                  CHECK (direction IN ('inbound', 'outbound')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX federation_activities_person_idx
    ON federation_activities (person_id, direction, created_at DESC);

CREATE TABLE federation_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id     UUID NOT NULL REFERENCES federation_activities (id) ON DELETE CASCADE,
    inbox_url       TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'delivered', 'failed')),
    attempts        INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error      TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (activity_id, inbox_url)
);

CREATE INDEX federation_deliveries_pending_idx
    ON federation_deliveries (status, next_attempt_at);

-- +goose Down
DROP TABLE federation_deliveries;
DROP TABLE federation_activities;
DROP TABLE federation_follows;
DROP TABLE actor_keys;
