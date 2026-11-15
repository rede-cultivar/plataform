-- +goose Up
CREATE TABLE people (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    handle       TEXT NOT NULL,
    display_name TEXT NOT NULL,
    bio          TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX people_handle_key ON people (lower(handle));

CREATE TABLE accounts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id         UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    email             TEXT NOT NULL,
    password_hash     TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'active'
                      CHECK (status IN ('active', 'suspended', 'closed')),
    email_verified_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX accounts_email_key ON accounts (lower(email));
CREATE UNIQUE INDEX accounts_person_id_key ON accounts (person_id);

CREATE TABLE sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id          UUID NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    token_hash          BYTEA NOT NULL UNIQUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    idle_expires_at     TIMESTAMPTZ NOT NULL,
    absolute_expires_at TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ,
    ip_address          TEXT NOT NULL DEFAULT '',
    user_agent          TEXT NOT NULL DEFAULT ''
);

CREATE INDEX sessions_account_id_idx ON sessions (account_id);
CREATE INDEX sessions_idle_expires_at_idx ON sessions (idle_expires_at);

-- +goose Down
DROP TABLE sessions;
DROP TABLE accounts;
DROP TABLE people;
