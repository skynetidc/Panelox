-- Backup storage accounts (S3, SFTP, local)
CREATE TABLE backup_accounts (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(50) NOT NULL,
    config      JSONB NOT NULL DEFAULT '{}',
    is_default  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Extend backups table
ALTER TABLE backups ADD COLUMN IF NOT EXISTS account_id BIGINT REFERENCES backup_accounts(id);
ALTER TABLE backups ADD COLUMN IF NOT EXISTS remote_path TEXT;

-- Scheduled backups
CREATE TABLE backup_schedules (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(50) NOT NULL,
    account_id      BIGINT REFERENCES backup_accounts(id),
    schedule        VARCHAR(100) NOT NULL DEFAULT '@daily',
    retention_days  INT NOT NULL DEFAULT 30,
    retention_count INT NOT NULL DEFAULT 10,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    last_run        TIMESTAMPTZ,
    last_status     VARCHAR(20),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default local account
INSERT INTO backup_accounts (name, type, config, is_default)
VALUES ('Local', 'local', '{"path":"/opt/panelox/backups"}', true)
ON CONFLICT DO NOTHING;
