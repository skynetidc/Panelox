-- Cron Jobs
CREATE TABLE cronjobs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(50) NOT NULL DEFAULT 'shell',
    schedule    VARCHAR(100) NOT NULL,
    command     TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    last_run    TIMESTAMPTZ,
    last_status VARCHAR(20),
    last_output TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Backups
CREATE TABLE backups (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(50) NOT NULL,
    source_id   BIGINT,
    file_path   TEXT NOT NULL,
    file_size   BIGINT NOT NULL DEFAULT 0,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    message     TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- App Store
CREATE TABLE app_catalog (
    id          BIGSERIAL PRIMARY KEY,
    key         VARCHAR(100) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon        VARCHAR(500) NOT NULL DEFAULT '',
    category    VARCHAR(100) NOT NULL DEFAULT 'other',
    versions    JSONB NOT NULL DEFAULT '[]',
    compose     TEXT NOT NULL,
    env_schema  JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE app_installs (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id),
    app_key       VARCHAR(100) NOT NULL,
    name          VARCHAR(255) NOT NULL,
    version       VARCHAR(50) NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'running',
    container_ids TEXT[] NOT NULL DEFAULT '{}',
    env_values    JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- System metrics snapshots
CREATE TABLE metrics (
    id          BIGSERIAL PRIMARY KEY,
    cpu_percent DOUBLE PRECISION NOT NULL,
    mem_used    BIGINT NOT NULL,
    mem_total   BIGINT NOT NULL,
    disk_used   BIGINT NOT NULL,
    disk_total  BIGINT NOT NULL,
    net_in      BIGINT NOT NULL DEFAULT 0,
    net_out     BIGINT NOT NULL DEFAULT 0,
    load_1      DOUBLE PRECISION NOT NULL DEFAULT 0,
    load_5      DOUBLE PRECISION NOT NULL DEFAULT 0,
    load_15     DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_metrics_created ON metrics(created_at);

-- DNS provider accounts for ACME
CREATE TABLE dns_provider_accounts (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    provider_type VARCHAR(50) NOT NULL,
    credentials   JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE ssl_certificates ADD COLUMN IF NOT EXISTS dns_provider_id BIGINT REFERENCES dns_provider_accounts(id);
ALTER TABLE ssl_certificates ADD COLUMN IF NOT EXISTS challenge_type VARCHAR(20) NOT NULL DEFAULT 'http';
