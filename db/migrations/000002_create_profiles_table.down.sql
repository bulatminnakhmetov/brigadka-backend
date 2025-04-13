-- Удаление таблиц музыкального профиля
DROP TABLE IF EXISTS music_profile_instruments;
DROP TABLE IF EXISTS music_profile_genres;
DROP TABLE IF EXISTS music_genre_translation;
DROP TABLE IF EXISTS music_genre_catalog;
DROP TABLE IF EXISTS music_instrument_translation;
DROP TABLE IF EXISTS music_instrument_catalog;

-- Удаление таблиц импровизационного профиля
DROP TABLE IF EXISTS improv_profile_styles;
DROP TABLE IF EXISTS improv_profiles;
DROP TABLE IF EXISTS improv_style_translation;
DROP TABLE IF EXISTS improv_style_catalog;
DROP TABLE IF EXISTS improv_goals_translation;
DROP TABLE IF EXISTS improv_goals_catalog;

-- Удаление базовой таблицы профилей
DROP TABLE IF EXISTS profiles;

-- Удаление каталога типов активности
DROP TABLE IF EXISTS activity_type_catalog;