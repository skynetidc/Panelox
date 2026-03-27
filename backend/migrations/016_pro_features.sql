-- Nginx snippet presets (PRO)
CREATE TABLE IF NOT EXISTS nginx_snippets (
    id          BIGSERIAL PRIMARY KEY,
    website_id  BIGINT NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    preset_key  VARCHAR(100) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    category    VARCHAR(50) NOT NULL DEFAULT 'custom',
    content     TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-website rate limiting (PRO)
ALTER TABLE websites ADD COLUMN IF NOT EXISTS rate_limit_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS rate_limit_rps INT NOT NULL DEFAULT 0;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS rate_limit_burst INT NOT NULL DEFAULT 0;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS rate_limit_nodelay BOOLEAN NOT NULL DEFAULT true;
