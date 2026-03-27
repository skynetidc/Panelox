-- Users & Auth
CREATE TABLE users (
    id          BIGSERIAL PRIMARY KEY,
    email       VARCHAR(255) NOT NULL UNIQUE,
    password    VARCHAR(255) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    role        VARCHAR(50) NOT NULL DEFAULT 'user',
    parent_id   BIGINT REFERENCES users(id),
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    language    VARCHAR(10) NOT NULL DEFAULT 'en',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sessions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token VARCHAR(512) NOT NULL UNIQUE,
    user_agent    TEXT,
    ip_address    VARCHAR(45),
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Websites
CREATE TABLE websites (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    domain          VARCHAR(255) NOT NULL,
    aliases         TEXT[],
    type            VARCHAR(50) NOT NULL DEFAULT 'static',
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    document_root   VARCHAR(500),
    proxy_target    VARCHAR(500),
    container_id    VARCHAR(100),
    php_version     VARCHAR(10),
    ssl_enabled     BOOLEAN NOT NULL DEFAULT false,
    ssl_cert_id     BIGINT,
    nginx_custom    TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_websites_domain ON websites(domain);

-- SSL Certificates
CREATE TABLE ssl_certificates (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    domains         TEXT[] NOT NULL,
    provider        VARCHAR(50) NOT NULL DEFAULT 'letsencrypt',
    cert_pem        TEXT,
    key_pem         TEXT,
    chain_pem       TEXT,
    auto_renew      BOOLEAN NOT NULL DEFAULT true,
    expires_at      TIMESTAMPTZ,
    issued_at       TIMESTAMPTZ,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE websites ADD CONSTRAINT fk_websites_ssl FOREIGN KEY (ssl_cert_id) REFERENCES ssl_certificates(id);

-- DNS
CREATE TABLE dns_zones (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    name        VARCHAR(255) NOT NULL UNIQUE,
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    serial      BIGINT NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE dns_records (
    id          BIGSERIAL PRIMARY KEY,
    zone_id     BIGINT NOT NULL REFERENCES dns_zones(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(10) NOT NULL,
    content     TEXT NOT NULL,
    ttl         INT NOT NULL DEFAULT 3600,
    priority    INT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_dns_records_zone ON dns_records(zone_id);

-- Mail
CREATE TABLE mail_domains (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    domain          VARCHAR(255) NOT NULL UNIQUE,
    container_id    VARCHAR(100),
    dkim_public_key TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mailboxes (
    id              BIGSERIAL PRIMARY KEY,
    mail_domain_id  BIGINT NOT NULL REFERENCES mail_domains(id) ON DELETE CASCADE,
    address         VARCHAR(255) NOT NULL,
    password        VARCHAR(255) NOT NULL,
    quota_mb        INT NOT NULL DEFAULT 1024,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_mailboxes_address_domain ON mailboxes(address, mail_domain_id);

-- Databases
CREATE TABLE db_servers (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(20) NOT NULL,
    version         VARCHAR(20) NOT NULL,
    host            VARCHAR(255) NOT NULL,
    port            INT NOT NULL,
    root_password   VARCHAR(255) NOT NULL,
    container_id    VARCHAR(100),
    status          VARCHAR(20) NOT NULL DEFAULT 'running',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE databases (
    id          BIGSERIAL PRIMARY KEY,
    server_id   BIGINT NOT NULL REFERENCES db_servers(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    charset     VARCHAR(50) NOT NULL DEFAULT 'utf8mb4',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE db_users (
    id          BIGSERIAL PRIMARY KEY,
    server_id   BIGINT NOT NULL REFERENCES db_servers(id) ON DELETE CASCADE,
    username    VARCHAR(255) NOT NULL,
    password    VARCHAR(255) NOT NULL,
    privileges  JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Firewall
CREATE TABLE firewall_rules (
    id          BIGSERIAL PRIMARY KEY,
    chain       VARCHAR(20) NOT NULL DEFAULT 'INPUT',
    protocol    VARCHAR(10) NOT NULL,
    src_ip      VARCHAR(50),
    dst_port    VARCHAR(50) NOT NULL,
    action      VARCHAR(10) NOT NULL,
    description TEXT,
    position    INT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit log
CREATE TABLE audit_log (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id),
    action      VARCHAR(100) NOT NULL,
    resource    VARCHAR(100) NOT NULL,
    resource_id BIGINT,
    details     JSONB,
    ip_address  VARCHAR(45),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_log_user ON audit_log(user_id);
CREATE INDEX idx_audit_log_created ON audit_log(created_at);

-- Settings
CREATE TABLE settings (
    key         VARCHAR(255) PRIMARY KEY,
    value       TEXT NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
