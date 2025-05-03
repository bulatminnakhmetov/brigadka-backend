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