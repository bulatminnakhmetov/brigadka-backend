-- Таблица городов
CREATE TABLE cities (
    city_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

-- Начальные города
INSERT INTO cities (name) VALUES
    ('Москва'),
    ('Санкт-Петербург');

-- Каталог гендерных идентичностей
CREATE TABLE gender_catalog (
    gender_code VARCHAR(50) PRIMARY KEY -- например: male, female, non-binary, other
);

-- Переводы для гендерных идентичностей
CREATE TABLE gender_catalog_translation (
    gender_code VARCHAR(50) REFERENCES gender_catalog(gender_code) ON DELETE CASCADE,
    lang VARCHAR(10) NOT NULL, -- например: 'en', 'ru'
    label TEXT NOT NULL,
    PRIMARY KEY (gender_code, lang)
);

-- Таблица пользователей
CREATE TABLE users (
    full_name VARCHAR(255) NOT NULL, -- реальное имя пользователя
    city_id INT REFERENCES cities(city_id), -- ссылка на город
    user_id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    gender VARCHAR(50) REFERENCES gender_catalog(gender_code),
    age INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Начальные значения гендеров и переводов
INSERT INTO gender_catalog (gender_code) VALUES
    ('male'),
    ('female');

INSERT INTO gender_catalog_translation (gender_code, lang, label) VALUES
    ('male', 'ru', 'Мужской'),
    ('female', 'ru', 'Женский');
