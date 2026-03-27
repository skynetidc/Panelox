-- Login Log
CREATE TABLE login_log (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id),
    email       VARCHAR(255) NOT NULL,
    ip_address  VARCHAR(45) NOT NULL,
    user_agent  TEXT,
    success     BOOLEAN NOT NULL,
    message     TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_login_log_created ON login_log(created_at);
CREATE INDEX idx_login_log_ip ON login_log(ip_address);

-- 2FA TOTP
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret VARCHAR(64);
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN NOT NULL DEFAULT false;

-- Fail2Ban rules
CREATE TABLE fail2ban_rules (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    port        VARCHAR(50) NOT NULL,
    max_retry   INT NOT NULL DEFAULT 5,
    ban_time    INT NOT NULL DEFAULT 3600,
    find_time   INT NOT NULL DEFAULT 600,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Banned IPs
CREATE TABLE fail2ban_banned (
    id          BIGSERIAL PRIMARY KEY,
    rule_id     BIGINT REFERENCES fail2ban_rules(id) ON DELETE CASCADE,
    ip_address  VARCHAR(45) NOT NULL,
    reason      TEXT,
    banned_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_banned_ip ON fail2ban_banned(ip_address);

-- Notification channels
CREATE TABLE notification_channels (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    type        VARCHAR(50) NOT NULL,
    config      JSONB NOT NULL DEFAULT '{}',
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Alert rules
CREATE TABLE alert_rules (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    metric          VARCHAR(50) NOT NULL,
    operator        VARCHAR(10) NOT NULL DEFAULT '>',
    threshold       DOUBLE PRECISION NOT NULL,
    duration_sec    INT NOT NULL DEFAULT 300,
    channel_id      BIGINT REFERENCES notification_channels(id),
    enabled         BOOLEAN NOT NULL DEFAULT true,
    last_triggered  TIMESTAMPTZ,
    status          VARCHAR(20) NOT NULL DEFAULT 'ok',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Snapshots
CREATE TABLE snapshots (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    file_path   TEXT NOT NULL,
    file_size   BIGINT NOT NULL DEFAULT 0,
    includes    TEXT[] NOT NULL DEFAULT '{}',
    status      VARCHAR(20) NOT NULL DEFAULT 'creating',
    message     TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
