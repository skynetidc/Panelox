-- Per-website IP access control (PRO)
CREATE TABLE IF NOT EXISTS website_ip_rules (
    id          BIGSERIAL PRIMARY KEY,
    website_id  BIGINT NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    mode        VARCHAR(10) NOT NULL DEFAULT 'deny',  -- allow or deny
    ip_address  VARCHAR(45) NOT NULL,                  -- IPv4 or IPv6 or CIDR
    description VARCHAR(255) DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Website-level access mode: off, whitelist, blacklist
ALTER TABLE websites ADD COLUMN IF NOT EXISTS ip_access_mode VARCHAR(20) NOT NULL DEFAULT 'off';
