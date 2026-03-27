-- Website domains (multi-domain support per website, inspired by 1Panel)
CREATE TABLE website_domains (
    id          BIGSERIAL PRIMARY KEY,
    website_id  BIGINT NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    domain      VARCHAR(255) NOT NULL,
    port        INT NOT NULL DEFAULT 80,
    ssl         BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_website_domains_website ON website_domains(website_id);
CREATE UNIQUE INDEX idx_website_domains_unique ON website_domains(domain, port);

-- Add alias and log columns to websites
ALTER TABLE websites ADD COLUMN IF NOT EXISTS alias VARCHAR(255);
ALTER TABLE websites ADD COLUMN IF NOT EXISTS access_log BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS error_log BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS default_server BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS ipv6 BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS remark TEXT NOT NULL DEFAULT '';

-- Create unique index on alias
CREATE UNIQUE INDEX IF NOT EXISTS idx_websites_alias ON websites(alias);
