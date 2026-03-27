-- Cron job run history
CREATE TABLE cronjob_history (
    id          BIGSERIAL PRIMARY KEY,
    job_id      BIGINT NOT NULL REFERENCES cronjobs(id) ON DELETE CASCADE,
    status      VARCHAR(20) NOT NULL,
    output      TEXT,
    duration_ms INT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_cronjob_history_job ON cronjob_history(job_id);

-- SSH keys
CREATE TABLE ssh_keys (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    public_key  TEXT NOT NULL,
    private_key TEXT NOT NULL,
    fingerprint VARCHAR(100),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Node groups / regions
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS node_group VARCHAR(100) NOT NULL DEFAULT 'default';
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS region VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS cpu_cores INT NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS mem_total_mb BIGINT NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS disk_total_mb BIGINT NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS cpu_usage DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS mem_usage_mb BIGINT NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS disk_usage_mb BIGINT NOT NULL DEFAULT 0;
