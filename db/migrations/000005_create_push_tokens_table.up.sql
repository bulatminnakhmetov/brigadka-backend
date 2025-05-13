CREATE TABLE push_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    token TEXT NOT NULL UNIQUE,
    platform VARCHAR(10) NOT NULL,
    device_id TEXT,
    last_seen_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create an index on user_id for faster lookups
CREATE INDEX push_tokens_user_id_idx ON push_tokens(user_id);