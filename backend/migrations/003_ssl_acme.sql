-- ACME accounts (Let's Encrypt, ZeroSSL, etc.)
CREATE TABLE acme_accounts (
    id          BIGSERIAL PRIMARY KEY,
    email       VARCHAR(255) NOT NULL,
    provider    VARCHAR(50) NOT NULL DEFAULT 'letsencrypt',
    private_key TEXT NOT NULL,
    url         TEXT,
    key_type    VARCHAR(20) NOT NULL DEFAULT 'EC256',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Extend ssl_certificates table with ACME fields
ALTER TABLE ssl_certificates ADD COLUMN IF NOT EXISTS acme_account_id BIGINT REFERENCES acme_accounts(id);
ALTER TABLE ssl_certificates ADD COLUMN IF NOT EXISTS key_type VARCHAR(20) NOT NULL DEFAULT 'EC256';
ALTER TABLE ssl_certificates ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE ssl_certificates ADD COLUMN IF NOT EXISTS cert_url TEXT;
ALTER TABLE ssl_certificates ADD COLUMN IF NOT EXISTS message TEXT NOT NULL DEFAULT '';
ALTER TABLE ssl_certificates ADD COLUMN IF NOT EXISTS http_config VARCHAR(20) NOT NULL DEFAULT 'HTTPToHTTPS';
