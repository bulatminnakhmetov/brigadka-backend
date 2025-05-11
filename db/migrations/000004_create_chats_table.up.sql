CREATE TABLE chats (
    id UUID PRIMARY KEY,
    chat_name VARCHAR(255) CHECK (chat_name IS NULL OR LENGTH(TRIM(chat_name)) > 0),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    is_group BOOLEAN NOT NULL DEFAULT FALSE
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
	chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
	sender_id INT REFERENCES users(id) ON DELETE CASCADE,
	content TEXT NOT NULL CHECK (LENGTH(TRIM(content)) > 0),
	sent_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
	seq BIGSERIAL
);

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
	message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
	user_id INT REFERENCES users(id) ON DELETE CASCADE,
	reaction_code VARCHAR(50) REFERENCES reaction_catalog(reaction_code) ON DELETE CASCADE,
	reacted_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (message_id, user_id, reaction_code)
);

CREATE TABLE message_read_receipts (
	user_id INT REFERENCES users(id) ON DELETE CASCADE,
	chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
	last_read_seq BIGINT,
	read_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (user_id, chat_id)
);
