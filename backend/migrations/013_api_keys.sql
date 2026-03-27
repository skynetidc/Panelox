-- API Keys
CREATE TABLE api_keys (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    key_hash    VARCHAR(255) NOT NULL UNIQUE,
    key_prefix  VARCHAR(20) NOT NULL,
    permissions VARCHAR(20) NOT NULL DEFAULT 'full',
    expires_at  TIMESTAMPTZ,
    last_used   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
