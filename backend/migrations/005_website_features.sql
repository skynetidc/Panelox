-- PHP Runtimes
CREATE TABLE php_runtimes (
    id              BIGSERIAL PRIMARY KEY,
    version         VARCHAR(10) NOT NULL UNIQUE,
    container_id    VARCHAR(100),
    container_name  VARCHAR(100),
    status          VARCHAR(20) NOT NULL DEFAULT 'stopped',
    port            INT NOT NULL DEFAULT 9000,
    extensions      TEXT[] NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Website advanced features
ALTER TABLE websites ADD COLUMN IF NOT EXISTS hsts_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS hsts_max_age INT NOT NULL DEFAULT 31536000;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS hsts_subdomains BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS http3_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS basic_auth_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS basic_auth_user VARCHAR(255);
ALTER TABLE websites ADD COLUMN IF NOT EXISTS basic_auth_pass VARCHAR(255);
ALTER TABLE websites ADD COLUMN IF NOT EXISTS anti_hotlink BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS hotlink_extensions TEXT NOT NULL DEFAULT 'jpg,jpeg,png,gif,svg,webp';
ALTER TABLE websites ADD COLUMN IF NOT EXISTS hotlink_allowed_domains TEXT NOT NULL DEFAULT '';
ALTER TABLE websites ADD COLUMN IF NOT EXISTS proxy_cache BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS proxy_cache_time VARCHAR(20) NOT NULL DEFAULT '1h';
ALTER TABLE websites ADD COLUMN IF NOT EXISTS proxy_websocket BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS proxy_headers JSONB NOT NULL DEFAULT '{}';

-- Website redirect rules
CREATE TABLE website_redirects (
    id          BIGSERIAL PRIMARY KEY,
    website_id  BIGINT NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    type        VARCHAR(10) NOT NULL DEFAULT '301',
    source      VARCHAR(500) NOT NULL,
    target      VARCHAR(500) NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_website_redirects_site ON website_redirects(website_id);
