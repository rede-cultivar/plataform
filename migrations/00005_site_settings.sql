-- +goose Up
-- Papéis de instância conforme o ADR 0008. A promoção do primeiro
-- administrador é feita pelo comando `cultivar admin promote <identificador>`,
-- nunca pelo cadastro público.
ALTER TABLE accounts
    ADD COLUMN role TEXT NOT NULL DEFAULT 'member'
    CHECK (role IN ('admin', 'moderator', 'member'));

-- Contas com cadastro aprovado ficam pendentes até a aprovação administrativa.
ALTER TABLE accounts DROP CONSTRAINT accounts_status_check;
ALTER TABLE accounts ADD CONSTRAINT accounts_status_check
    CHECK (status IN ('active', 'suspended', 'closed', 'pending'));

-- Configurações administrativas da instância. O modelo segue o Mastodon:
-- cadastro aberto, sujeito a aprovação ou fechado, com mensagem configurável.
CREATE TABLE site_settings (
    id                           BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    registrations_mode           TEXT NOT NULL DEFAULT 'open'
                                 CHECK (registrations_mode IN ('open', 'approved', 'none')),
    closed_registrations_message TEXT NOT NULL DEFAULT '',
    updated_at                   TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO site_settings (id) VALUES (TRUE);

-- +goose Down
DROP TABLE site_settings;

UPDATE accounts SET status = 'closed' WHERE status = 'pending';

ALTER TABLE accounts DROP CONSTRAINT accounts_status_check;
ALTER TABLE accounts ADD CONSTRAINT accounts_status_check
    CHECK (status IN ('active', 'suspended', 'closed'));

ALTER TABLE accounts DROP COLUMN role;
