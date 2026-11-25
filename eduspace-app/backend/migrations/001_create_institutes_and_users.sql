-- Schools (each school is one "institute")
CREATE TABLE IF NOT EXISTS institutes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(150) NOT NULL,
    status     VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Login accounts
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) UNIQUE,
    phone         VARCHAR(30),
    password_hash TEXT         NOT NULL,
    full_name     VARCHAR(150) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Which school a user belongs to, and in what role
CREATE TABLE IF NOT EXISTS institute_members (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID        NOT NULL REFERENCES institutes(id),
    user_id      UUID        NOT NULL REFERENCES users(id),
    role         VARCHAR(30) NOT NULL,
    UNIQUE (institute_id, user_id)
);
