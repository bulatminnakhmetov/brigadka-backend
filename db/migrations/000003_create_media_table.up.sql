-- Каталог типов медиа (ролей)
CREATE TABLE role_catalog (
    role VARCHAR(50) PRIMARY KEY, -- например: avatar, gallery, cover
    description TEXT
);

-- Начальные значения ролей медиа
INSERT INTO role_catalog (role, description) VALUES
    ('avatar', 'Основная фотография профиля или команды'),
    ('gallery', 'Дополнительные изображения'),
    ('cover', 'Обложка для страницы');

-- Таблица медиа
CREATE TABLE media (
    profile_id INT REFERENCES profiles(profile_id) ON DELETE CASCADE,
    id SERIAL PRIMARY KEY,
    type VARCHAR(50),
    role VARCHAR(50) REFERENCES role_catalog(role),
    url TEXT NOT NULL,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);