-- Server nodes (managed by core)
CREATE TABLE IF NOT EXISTS nodes (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL UNIQUE,
    host        VARCHAR(255) NOT NULL,
    port        INT NOT NULL DEFAULT 22,
    ssh_user    VARCHAR(100) NOT NULL DEFAULT 'root',
    auth_mode   VARCHAR(20) NOT NULL DEFAULT 'password',
    password    TEXT,
    private_key TEXT,
    passphrase  TEXT,
    agent_port  INT NOT NULL DEFAULT 9090,
    status      VARCHAR(20) NOT NULL DEFAULT 'offline',
    is_local    BOOLEAN NOT NULL DEFAULT false,
    last_ping   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed local node
INSERT INTO nodes (name, host, port, ssh_user, agent_port, status, is_local)
VALUES ('local', '127.0.0.1', 22, 'root', 9090, 'online', true)
ON CONFLICT (name) DO NOTHING;
