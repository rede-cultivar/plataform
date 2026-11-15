-- +goose Up
CREATE TABLE practices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id       UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    summary         TEXT NOT NULL DEFAULT '',
    category        TEXT NOT NULL DEFAULT '',
    difficulty      TEXT NOT NULL DEFAULT 'beginner'
                    CHECK (difficulty IN ('beginner', 'intermediate', 'advanced')),
    time_estimate   TEXT NOT NULL DEFAULT '',
    cost_estimate   TEXT NOT NULL DEFAULT '',
    materials       JSONB NOT NULL DEFAULT '[]'::jsonb
                    CHECK (jsonb_typeof(materials) = 'array'),
    steps           JSONB NOT NULL DEFAULT '[]'::jsonb
                    CHECK (jsonb_typeof(steps) = 'array'),
    safety            TEXT NOT NULL DEFAULT '',
    adaptations       TEXT NOT NULL DEFAULT '',
    source_references TEXT NOT NULL DEFAULT '',
    license           TEXT NOT NULL DEFAULT '',
    version           INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    status            TEXT NOT NULL DEFAULT 'draft'
                      CHECK (status IN ('draft', 'published', 'archived')),
    visibility        TEXT NOT NULL DEFAULT 'private'
                      CHECK (visibility IN ('private', 'public')),
    published_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX practices_author_id_idx ON practices (author_id);
CREATE INDEX practices_published_idx ON practices (status, visibility, published_at DESC);
CREATE INDEX practices_category_idx ON practices (category);
CREATE INDEX practices_difficulty_idx ON practices (difficulty);

CREATE TABLE cultivation_cycles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id   UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    species     TEXT NOT NULL,
    variety     TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL DEFAULT '',
    environment TEXT NOT NULL DEFAULT 'outdoor'
                CHECK (environment IN ('indoor', 'outdoor', 'greenhouse', 'nursery', 'other')),
    location    TEXT NOT NULL DEFAULT '',
    substrate   TEXT NOT NULL DEFAULT '',
    container   TEXT NOT NULL DEFAULT '',
    technique   TEXT NOT NULL DEFAULT '',
    stage       TEXT NOT NULL DEFAULT 'germination'
                CHECK (stage IN ('germination', 'seedling', 'vegetative', 'flowering', 'fruiting', 'harvest', 'finished')),
    status      TEXT NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'finished', 'archived')),
    visibility  TEXT NOT NULL DEFAULT 'private'
                CHECK (visibility IN ('private', 'followers', 'public')),
    started_on  DATE NOT NULL,
    finished_on DATE,
    outcome     TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (finished_on IS NULL OR finished_on >= started_on)
);

CREATE INDEX cultivation_cycles_person_idx ON cultivation_cycles (person_id, started_on DESC);
CREATE INDEX cultivation_cycles_public_idx ON cultivation_cycles (visibility, status, started_on DESC);
CREATE INDEX cultivation_cycles_species_idx ON cultivation_cycles (lower(species));
CREATE INDEX cultivation_cycles_environment_idx ON cultivation_cycles (environment);
CREATE INDEX cultivation_cycles_stage_idx ON cultivation_cycles (stage);

CREATE TABLE cultivation_entries (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cycle_id         UUID NOT NULL REFERENCES cultivation_cycles (id) ON DELETE CASCADE,
    author_id        UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    occurred_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    body             TEXT NOT NULL DEFAULT '',
    stage            TEXT
                     CHECK (stage IS NULL OR stage IN ('germination', 'seedling', 'vegetative', 'flowering', 'fruiting', 'harvest', 'finished')),
    measurements     JSONB NOT NULL DEFAULT '{}'::jsonb
                     CHECK (jsonb_typeof(measurements) = 'object'),
    interventions    TEXT[] NOT NULL DEFAULT '{}',
    occurrences      TEXT[] NOT NULL DEFAULT '{}',
    harvest_quantity NUMERIC(12, 3),
    harvest_unit     TEXT NOT NULL DEFAULT '',
    practice_id      UUID REFERENCES practices (id) ON DELETE SET NULL,
    visibility       TEXT NOT NULL DEFAULT 'inherit'
                     CHECK (visibility IN ('inherit', 'private', 'followers', 'public')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (harvest_quantity IS NULL OR harvest_quantity >= 0)
);

CREATE INDEX cultivation_entries_cycle_idx ON cultivation_entries (cycle_id, occurred_at DESC);
CREATE INDEX cultivation_entries_author_idx ON cultivation_entries (author_id);
CREATE INDEX cultivation_entries_practice_idx ON cultivation_entries (practice_id);
CREATE INDEX cultivation_entries_interventions_idx ON cultivation_entries USING GIN (interventions);
CREATE INDEX cultivation_entries_occurrences_idx ON cultivation_entries USING GIN (occurrences);

CREATE TABLE media_assets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id      UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    storage_key   TEXT NOT NULL UNIQUE,
    original_name TEXT NOT NULL DEFAULT '',
    content_type  TEXT NOT NULL,
    byte_size     BIGINT NOT NULL DEFAULT 0 CHECK (byte_size >= 0),
    width         INTEGER CHECK (width IS NULL OR width > 0),
    height        INTEGER CHECK (height IS NULL OR height > 0),
    status        TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'ready', 'failed')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX media_assets_owner_idx ON media_assets (owner_id);

CREATE TABLE cultivation_entry_media (
    entry_id UUID NOT NULL REFERENCES cultivation_entries (id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES media_assets (id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    PRIMARY KEY (entry_id, media_id)
);

CREATE INDEX cultivation_entry_media_media_idx ON cultivation_entry_media (media_id);

CREATE TABLE practice_media (
    practice_id UUID NOT NULL REFERENCES practices (id) ON DELETE CASCADE,
    media_id    UUID NOT NULL REFERENCES media_assets (id) ON DELETE CASCADE,
    position    INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    PRIMARY KEY (practice_id, media_id)
);

CREATE INDEX practice_media_media_idx ON practice_media (media_id);

CREATE TABLE practice_versions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    practice_id UUID NOT NULL REFERENCES practices (id) ON DELETE CASCADE,
    version     INTEGER NOT NULL CHECK (version > 0),
    author_id   UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    snapshot    JSONB NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (practice_id, version)
);

CREATE INDEX practice_versions_practice_idx ON practice_versions (practice_id, version DESC);

-- +goose Down
DROP TABLE practice_versions;
DROP TABLE practice_media;
DROP TABLE cultivation_entry_media;
DROP TABLE media_assets;
DROP TABLE cultivation_entries;
DROP TABLE cultivation_cycles;
DROP TABLE practices;
