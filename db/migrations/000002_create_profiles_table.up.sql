-- Каталог типов активности профиля
CREATE TABLE activity_type_catalog (
    activity_type VARCHAR(50) PRIMARY KEY, -- например: improv, music, dance
    description TEXT
);

-- Начальные значения типов активности
INSERT INTO activity_type_catalog (activity_type, description) VALUES
    ('improv', 'Комедийная импровизация'),
    ('music', 'Музыкальное исполнение'),
    ('dance', 'Танцевальные выступления');

-- Базовая таблица профилей
CREATE TABLE profiles (
    profile_id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(user_id) ON DELETE CASCADE,
    description TEXT,
    activity_type VARCHAR(50) REFERENCES activity_type_catalog(activity_type),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);