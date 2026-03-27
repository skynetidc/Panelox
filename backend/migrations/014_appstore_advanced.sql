-- App health check status
ALTER TABLE app_installs ADD COLUMN IF NOT EXISTS health_status VARCHAR(20) NOT NULL DEFAULT 'unknown';
ALTER TABLE app_installs ADD COLUMN IF NOT EXISTS health_checked TIMESTAMPTZ;

-- Custom app repositories
CREATE TABLE IF NOT EXISTS app_repos (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    url         VARCHAR(500) NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    last_sync   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
