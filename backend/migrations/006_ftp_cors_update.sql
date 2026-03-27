-- FTP users
CREATE TABLE ftp_users (
    id          BIGSERIAL PRIMARY KEY,
    website_id  BIGINT REFERENCES websites(id) ON DELETE CASCADE,
    username    VARCHAR(255) NOT NULL UNIQUE,
    password    VARCHAR(255) NOT NULL,
    home_dir    VARCHAR(500) NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- CORS per-website
ALTER TABLE websites ADD COLUMN IF NOT EXISTS cors_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE websites ADD COLUMN IF NOT EXISTS cors_origins TEXT NOT NULL DEFAULT '*';
ALTER TABLE websites ADD COLUMN IF NOT EXISTS cors_methods TEXT NOT NULL DEFAULT 'GET,POST,PUT,DELETE,OPTIONS';
ALTER TABLE websites ADD COLUMN IF NOT EXISTS cors_headers TEXT NOT NULL DEFAULT 'Content-Type,Authorization';
ALTER TABLE websites ADD COLUMN IF NOT EXISTS cors_max_age INT NOT NULL DEFAULT 86400;

-- Panel version tracking
INSERT INTO settings (key, value) VALUES ('panel_version', '1.0.0') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('auto_update', 'false') ON CONFLICT (key) DO NOTHING;
