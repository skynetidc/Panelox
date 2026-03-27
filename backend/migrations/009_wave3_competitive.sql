-- Git Deploy
CREATE TABLE git_deploys (
    id              BIGSERIAL PRIMARY KEY,
    website_id      BIGINT NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    repo_url        VARCHAR(500) NOT NULL,
    branch          VARCHAR(100) NOT NULL DEFAULT 'main',
    auto_deploy     BOOLEAN NOT NULL DEFAULT true,
    webhook_secret  VARCHAR(100) NOT NULL,
    build_command   TEXT NOT NULL DEFAULT '',
    deploy_path     VARCHAR(500) NOT NULL DEFAULT '',
    last_commit     VARCHAR(100),
    last_deploy     TIMESTAMPTZ,
    last_status     VARCHAR(20),
    last_log        TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Git deploy history
CREATE TABLE deploy_history (
    id          BIGSERIAL PRIMARY KEY,
    deploy_id   BIGINT NOT NULL REFERENCES git_deploys(id) ON DELETE CASCADE,
    commit_hash VARCHAR(100) NOT NULL,
    commit_msg  TEXT,
    status      VARCHAR(20) NOT NULL,
    log         TEXT,
    duration_ms INT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- API Webhooks
CREATE TABLE webhooks (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    url         VARCHAR(500) NOT NULL,
    secret      VARCHAR(100),
    events      TEXT[] NOT NULL DEFAULT '{}',
    enabled     BOOLEAN NOT NULL DEFAULT true,
    last_sent   TIMESTAMPTZ,
    last_status INT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- White-label branding
INSERT INTO settings (key, value) VALUES ('brand_name', 'Panelox') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('brand_logo', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('brand_favicon', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('brand_primary_color', '#3b82f6') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('brand_sidebar_color', '#0f172a') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('brand_login_bg', '') ON CONFLICT (key) DO NOTHING;

-- Resource usage tracking per user
CREATE TABLE resource_usage (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    period      VARCHAR(10) NOT NULL,
    disk_bytes  BIGINT NOT NULL DEFAULT 0,
    bandwidth   BIGINT NOT NULL DEFAULT 0,
    cpu_minutes DOUBLE PRECISION NOT NULL DEFAULT 0,
    websites    INT NOT NULL DEFAULT 0,
    databases   INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_usage_user_period ON resource_usage(user_id, period);

-- Staging environments
CREATE TABLE staging_sites (
    id              BIGSERIAL PRIMARY KEY,
    website_id      BIGINT NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    subdomain       VARCHAR(255) NOT NULL,
    container_id    VARCHAR(100),
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
