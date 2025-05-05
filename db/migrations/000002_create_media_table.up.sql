-- Каталог типов медиа (ролей)
CREATE TABLE media_type_catalog (
    id VARCHAR(50) PRIMARY KEY
);

-- Начальные значения ролей медиа
INSERT INTO media_type_catalog (id) VALUES
    ('image'),
    ('video');

-- Таблица медиа
CREATE TABLE media (
    id SERIAL PRIMARY KEY,
    owner_id INT REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) REFERENCES media_type_catalog(id),
    url TEXT NOT NULL,
    thumbnail_url TEXT NOT NULL,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);