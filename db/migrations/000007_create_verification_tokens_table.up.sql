-- Create verification tokens table
CREATE TABLE verification_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_user_token_type UNIQUE (user_id, type)
);

-- Create index on token for faster lookups
CREATE INDEX idx_verification_tokens_token ON verification_tokens(token);