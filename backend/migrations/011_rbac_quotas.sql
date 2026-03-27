-- Resource quotas per user
CREATE TABLE user_quotas (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    max_websites    INT NOT NULL DEFAULT 0,
    max_databases   INT NOT NULL DEFAULT 0,
    max_ftp_users   INT NOT NULL DEFAULT 0,
    max_mail_domains INT NOT NULL DEFAULT 0,
    max_disk_mb     BIGINT NOT NULL DEFAULT 0,
    max_bandwidth_mb BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 0 = unlimited for all quota fields

-- Reseller packages (quota templates)
CREATE TABLE reseller_packages (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    max_websites    INT NOT NULL DEFAULT 10,
    max_databases   INT NOT NULL DEFAULT 5,
    max_ftp_users   INT NOT NULL DEFAULT 5,
    max_mail_domains INT NOT NULL DEFAULT 5,
    max_disk_mb     BIGINT NOT NULL DEFAULT 10240,
    max_bandwidth_mb BIGINT NOT NULL DEFAULT 102400,
    max_users       INT NOT NULL DEFAULT 10,
    price_monthly   DECIMAL(10,2) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Link users to packages
ALTER TABLE users ADD COLUMN IF NOT EXISTS package_id BIGINT REFERENCES reseller_packages(id);
ALTER TABLE users ADD COLUMN IF NOT EXISTS suspended BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS suspend_reason TEXT;
