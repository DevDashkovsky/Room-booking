CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL DEFAULT '',
    role          TEXT NOT NULL CHECK (role IN ('admin', 'user')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- фиксированные пользователи для dummyLogin
INSERT INTO users (id, email, role) VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin@test.com', 'admin'),
    ('00000000-0000-0000-0000-000000000002', 'user@test.com', 'user')
ON CONFLICT (id) DO NOTHING;
