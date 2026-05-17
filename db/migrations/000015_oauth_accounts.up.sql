CREATE TABLE IF NOT EXISTS oauth_account (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    provider TEXT NOT NULL CHECK(LENGTH(provider) >= 1 AND LENGTH(provider) <= 32),
    provider_user_id TEXT NOT NULL CHECK(LENGTH(provider_user_id) >= 1 AND LENGTH(provider_user_id) <= 128),
    user_account_id BIGINT NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
    email TEXT CHECK (
        email IS NULL
        OR (email ~ '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$' AND LENGTH(email) >= 5 AND LENGTH(email) <= 255)
    ),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_user_id),
    UNIQUE(provider, user_account_id)
);

CREATE INDEX IF NOT EXISTS oauth_account_user_account_id_idx
    ON oauth_account(user_account_id);
