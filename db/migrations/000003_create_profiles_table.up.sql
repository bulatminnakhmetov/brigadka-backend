-- ИМПРОВИЗАЦИЯ

-- Справочник стилей импровизации
CREATE TABLE improv_style_catalog (
    style_code VARCHAR(50) PRIMARY KEY
);

-- Переводы стилей импровизации
CREATE TABLE improv_style_translation (
    style_code VARCHAR(50) REFERENCES improv_style_catalog(style_code) ON DELETE CASCADE,
    lang VARCHAR(10) NOT NULL,
    label TEXT NOT NULL,
    PRIMARY KEY (style_code, lang)
);

-- Начальные значения стилей импровизации
INSERT INTO improv_style_catalog (style_code) VALUES
    ('shortform'),
    ('longform'),
    ('battles'),
    ('musical'),
    ('rap'),
    ('playback'),
    ('absurd'),
    ('realistic');

INSERT INTO improv_style_translation (style_code, lang, label) VALUES
    ('shortform', 'ru', 'Короткая форма'),
    ('longform', 'ru', 'Длинная форма'),
    ('battles', 'ru', 'Баттлы'),
    ('musical', 'ru', 'Мюзикл'),
    ('rap', 'ru', 'Фристайл-рэп'),
    ('playback', 'ru', 'Плейбэк-театр'),
    ('absurd', 'ru', 'Абсурд'),
    ('realistic', 'ru', 'Реализм');

-- Каталог целей импровизации
CREATE TABLE improv_goals_catalog (
    goal_id VARCHAR(50) PRIMARY KEY -- например: Hobby, Career, Experiment
);

-- Переводы целей импровизации
CREATE TABLE improv_goals_translation (
    goal_id VARCHAR(50) REFERENCES improv_goals_catalog(goal_id) ON DELETE CASCADE,
    lang VARCHAR(10) NOT NULL,
    label TEXT NOT NULL,
    PRIMARY KEY (goal_id, lang)
);

-- Начальные значения целей импровизации
INSERT INTO improv_goals_catalog (goal_id) VALUES
    ('hobby'),
    ('career');

INSERT INTO improv_goals_translation (goal_id, lang, label) VALUES
    ('hobby', 'en', 'Hobby'),
    ('hobby', 'ru', 'Хобби'),
    ('career', 'en', 'Career'),
    ('career', 'ru', 'Карьера');

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

-- Начальные значения гендеров и переводов
INSERT INTO gender_catalog (gender_code) VALUES
    ('male'),
    ('female');

INSERT INTO gender_catalog_translation (gender_code, lang, label) VALUES
    ('male', 'ru', 'Мужчина'),
    ('female', 'ru', 'Женщина'),
    ('male', 'en', 'Male'),
    ('female', 'en', 'Female');


-- Каталог типов медиа (ролей)
CREATE TABLE media_role_catalog (
    role VARCHAR(50) PRIMARY KEY
);

-- Начальные значения ролей медиа
INSERT INTO media_role_catalog (role) VALUES
    ('avatar'),
    ('video');

-- Базовая таблица профилей
CREATE TABLE profiles (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),

    full_name VARCHAR(255) NOT NULL,
    birthday DATE NOT NULL,
    gender VARCHAR(50) REFERENCES gender_catalog(gender_code),
    city_id INT REFERENCES cities(city_id),
    bio TEXT,
    
    goal VARCHAR(50) REFERENCES improv_goals_catalog(goal_id),
    looking_for_team BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Таблица соответствий профилей и стилей импровизации
CREATE TABLE improv_profile_styles (
    user_id INT REFERENCES profiles(user_id) ON DELETE CASCADE,
    style VARCHAR(50) REFERENCES improv_style_catalog(style_code) ON DELETE CASCADE,
    PRIMARY KEY (user_id, style)
);


CREATE TABLE profile_media (
    media_id INT REFERENCES media(id) ON DELETE CASCADE,
    user_id INT REFERENCES profiles(user_id) ON DELETE CASCADE,
    role VARCHAR(50) REFERENCES media_role_catalog(role),
    PRIMARY KEY (user_id, media_id)
);


