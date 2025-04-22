-- Каталог типов активности профиля
CREATE TABLE activity_type_catalog (
    activity_type VARCHAR(50) PRIMARY KEY, -- например: improv, music, dance
    description TEXT
);

-- Начальные значения типов активности
INSERT INTO activity_type_catalog (activity_type, description) VALUES
    ('improv', 'Комедийная импровизация'),
    ('music', 'Музыкальное исполнение');

-- Базовая таблица профилей
CREATE TABLE profiles (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    description TEXT,
    activity_type VARCHAR(50) REFERENCES activity_type_catalog(activity_type),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, activity_type)
);

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
    description TEXT,
    PRIMARY KEY (style_code, lang)
);

-- Начальные значения стилей импровизации
INSERT INTO improv_style_catalog (style_code) VALUES
    ('Short Form'),
    ('Long Form');
    
INSERT INTO improv_style_translation (style_code, lang, label, description) VALUES
    ('Short Form', 'en', 'Short Form', 'Fast-paced, game-based improv'),
    ('Short Form', 'ru', 'Короткая форма', 'Импровизация в формате коротких игр'),
    ('Long Form', 'en', 'Long Form', 'Extended scenes and narratives'),
    ('Long Form', 'ru', 'Длинная форма', 'Импровизация с длинными сценами и историей');

-- Каталог целей импровизации
CREATE TABLE improv_goals_catalog (
    goal_code VARCHAR(50) PRIMARY KEY -- например: Hobby, Career, Experiment
);

-- Переводы целей импровизации
CREATE TABLE improv_goals_translation (
    goal_code VARCHAR(50) REFERENCES improv_goals_catalog(goal_code) ON DELETE CASCADE,
    lang VARCHAR(10) NOT NULL,
    label TEXT NOT NULL,
    description TEXT,
    PRIMARY KEY (goal_code, lang)
);

-- Начальные значения целей импровизации
INSERT INTO improv_goals_catalog (goal_code) VALUES
    ('Hobby'),
    ('Career'),
    ('Experiment');

INSERT INTO improv_goals_translation (goal_code, lang, label, description) VALUES
    ('Hobby', 'en', 'Hobby', 'Doing improv for fun and leisure'),
    ('Hobby', 'ru', 'Хобби', 'Занятие импровом для удовольствия'),
    ('Career', 'en', 'Career', 'Professional interest in improv'),
    ('Career', 'ru', 'Карьера', 'Импровизация как профессиональный путь'),
    ('Experiment', 'en', 'Experiment', 'Trying something new'),
    ('Experiment', 'ru', 'Эксперимент', 'Изучение импрова ради нового опыта');

-- Таблица профиля для импровизации
CREATE TABLE improv_profiles (
    profile_id INT PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
    goal VARCHAR(50) REFERENCES improv_goals_catalog(goal_code),
    looking_for_team BOOLEAN DEFAULT FALSE -- Флаг "Ищу команду" перемещен сюда
);

-- Таблица соответствий профилей и стилей импровизации
CREATE TABLE improv_profile_styles (
    profile_id INT REFERENCES profiles(id) ON DELETE CASCADE,
    style VARCHAR(50) REFERENCES improv_style_catalog(style_code) ON DELETE CASCADE,
    PRIMARY KEY (profile_id, style)
);

-- МУЗЫКА

-- Справочник инструментов
CREATE TABLE music_instrument_catalog (
    instrument_code VARCHAR(100) PRIMARY KEY,
    description TEXT
);

-- Переводы инструментов
CREATE TABLE music_instrument_translation (
    instrument_code VARCHAR(100) REFERENCES music_instrument_catalog(instrument_code) ON DELETE CASCADE,
    lang VARCHAR(10) NOT NULL,
    label TEXT NOT NULL,
    PRIMARY KEY (instrument_code, lang)
);

-- Справочник жанров
CREATE TABLE music_genre_catalog (
    genre_code VARCHAR(100) PRIMARY KEY,
    description TEXT
);

-- Переводы жанров
CREATE TABLE music_genre_translation (
    genre_code VARCHAR(100) REFERENCES music_genre_catalog(genre_code) ON DELETE CASCADE,
    lang VARCHAR(10) NOT NULL,
    label TEXT NOT NULL,
    PRIMARY KEY (genre_code, lang)
);

-- Начальные значения для инструментов
INSERT INTO music_instrument_catalog (instrument_code, description) VALUES
    ('acoustic_guitar', 'Акустическая гитара'),
    ('electric_guitar', 'Электрогитара'),
    ('bass_guitar', 'Бас-гитара'),
    ('piano', 'Фортепиано'),
    ('synthesizer', 'Синтезатор'),
    ('drums', 'Ударные'),
    ('cajon', 'Кахон'),
    ('violin', 'Скрипка'),
    ('cello', 'Виолончель'),
    ('flute', 'Флейта'),
    ('saxophone', 'Саксофон'),
    ('trumpet', 'Труба'),
    ('voice', 'Вокал'),
    ('rap', 'Рэп'),
    ('dj', 'Диджеинг');

INSERT INTO music_instrument_translation (instrument_code, lang, label) VALUES
    ('acoustic_guitar', 'ru', 'Акустическая гитара'),
    ('electric_guitar', 'ru', 'Электрогитара'),
    ('bass_guitar', 'ru', 'Бас-гитара'),
    ('piano', 'ru', 'Фортепиано'),
    ('synthesizer', 'ru', 'Синтезатор'),
    ('drums', 'ru', 'Ударные'),
    ('cajon', 'ru', 'Кахон'),
    ('violin', 'ru', 'Скрипка'),
    ('cello', 'ru', 'Виолончель'),
    ('flute', 'ru', 'Флейта'),
    ('saxophone', 'ru', 'Саксофон'),
    ('trumpet', 'ru', 'Труба'),
    ('voice', 'ru', 'Вокал'),
    ('rap', 'ru', 'Рэп'),
    ('dj', 'ru', 'Диджеинг'),
    ('acoustic_guitar', 'en', 'Acoustic Guitar'),
    ('electric_guitar', 'en', 'Electric Guitar'),
    ('bass_guitar', 'en', 'Bass Guitar'),
    ('piano', 'en', 'Piano'),
    ('synthesizer', 'en', 'Synthesizer'),
    ('drums', 'en', 'Drums'),
    ('cajon', 'en', 'Cajon'),
    ('violin', 'en', 'Violin'),
    ('cello', 'en', 'Cello'),
    ('flute', 'en', 'Flute'),
    ('saxophone', 'en', 'Saxophone'),
    ('trumpet', 'en', 'Trumpet'),
    ('voice', 'en', 'Vocals'),
    ('rap', 'en', 'Rap'),
    ('dj', 'en', 'DJing');

-- Начальные значения для жанров
INSERT INTO music_genre_catalog (genre_code, description) VALUES
    ('rock', 'Рок'),
    ('jazz', 'Джаз'),
    ('classical', 'Классика'),
    ('pop', 'Поп-музыка'),
    ('electronic', 'Электронная музыка');

INSERT INTO music_genre_translation (genre_code, lang, label) VALUES
    ('rock', 'ru', 'Рок'),
    ('jazz', 'ru', 'Джаз'),
    ('classical', 'ru', 'Классика'),
    ('pop', 'ru', 'Поп-музыка'),
    ('electronic', 'ru', 'Электронная музыка'),
    ('rock', 'en', 'Rock'),
    ('jazz', 'en', 'Jazz'),
    ('classical', 'en', 'Classical'),
    ('pop', 'en', 'Pop'),
    ('electronic', 'en', 'Electronic');

-- Таблица соответствий профилей и жанров
CREATE TABLE music_profile_genres (
    profile_id INT REFERENCES profiles(id) ON DELETE CASCADE,
    genre_code VARCHAR(100) REFERENCES music_genre_catalog(genre_code) ON DELETE CASCADE,
    PRIMARY KEY (profile_id, genre_code)
);

-- Таблица соответствий профилей и инструментов
CREATE TABLE music_profile_instruments (
    profile_id INT REFERENCES profiles(id) ON DELETE CASCADE,
    instrument_code VARCHAR(100) REFERENCES music_instrument_catalog(instrument_code) ON DELETE CASCADE,
    PRIMARY KEY (profile_id, instrument_code)
);
