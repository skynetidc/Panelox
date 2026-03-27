-- License system
CREATE TABLE IF NOT EXISTS licenses (
    id              BIGSERIAL PRIMARY KEY,
    license_key     TEXT NOT NULL UNIQUE,
    tier            VARCHAR(20) NOT NULL DEFAULT 'free',  -- free, pro, enterprise
    customer_name   TEXT NOT NULL DEFAULT '',
    customer_email  TEXT NOT NULL DEFAULT '',
    max_websites    INT NOT NULL DEFAULT 3,
    max_nodes       INT NOT NULL DEFAULT 1,
    max_databases   INT NOT NULL DEFAULT 5,
    max_apps        INT NOT NULL DEFAULT 3,
    features        JSONB NOT NULL DEFAULT '[]',
    status          VARCHAR(20) NOT NULL DEFAULT 'active', -- active, expired, revoked
    activated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ,
    last_verified   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
