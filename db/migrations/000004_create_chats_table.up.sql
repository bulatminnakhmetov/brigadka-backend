CREATE TABLE chats (
    id UUID PRIMARY KEY,
    chat_name VARCHAR(255) CHECK (chat_name IS NULL OR LENGTH(TRIM(chat_name)) > 0),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE chat_participants (
    chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (chat_id, user_id)
);

CREATE INDEX idx_chat_participants_joined_at ON chat_participants(joined_at);

CREATE TABLE messages (
    id UUID PRIMARY KEY,
    chat_id INT REFERENCES chats(id) ON DELETE CASCADE,
    sender_id INT REFERENCES users(user_id),
    content TEXT NOT NULL CHECK (LENGTH(TRIM(content)) > 0),
    sent_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Добавляем индекс и ограничение уникальности для client_message_id
CREATE UNIQUE INDEX idx_messages_client_message_id ON messages(client_message_id) 
    WHERE client_message_id IS NOT NULL;
CREATE INDEX idx_messages_chat_id_sent_at ON messages(chat_id, sent_at);

CREATE TABLE reaction_catalog (
    reaction_code VARCHAR(50) PRIMARY KEY,
    emoji TEXT NOT NULL CHECK (LENGTH(TRIM(emoji)) > 0)
);

INSERT INTO reaction_catalog (reaction_code, emoji) VALUES
    ('like', '👍'),
    ('laugh', '😂'),
    ('clap', '👏'),
    ('heart', '❤️'),
    ('wow', '😮');

CREATE TABLE message_reactions (
    id UUID PRIMARY KEY,
    message_id INT REFERENCES messages(message_id) ON DELETE CASCADE,
    user_id INT REFERENCES users(user_id) ON DELETE CASCADE,
    reaction_code VARCHAR(50) REFERENCES reaction_catalog(reaction_code),
    reacted_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (message_id, user_id, reaction_code)
);

-- Добавляем индекс для client_reaction_id
CREATE UNIQUE INDEX idx_message_reactions_client_id ON message_reactions(client_reaction_id) 
    WHERE client_reaction_id IS NOT NULL;

CREATE TABLE message_read_receipts (
    user_id INT REFERENCES users(user_id) ON DELETE CASCADE,
    chat_id INT REFERENCES chats(id) ON DELETE CASCADE,
    last_read_message_id INT REFERENCES messages(message_id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, chat_id)
);

-- Добавляем индекс для client_read_id
CREATE INDEX idx_message_read_receipts_chat ON message_read_receipts(chat_id);
CREATE INDEX idx_message_reactions_message ON message_reactions(message_id);