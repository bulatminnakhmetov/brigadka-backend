# Brigadka Backend

## Описание

Бэкенд-сервис для проекта Brigadka. Использует PostgreSQL, MinIO (S3-совместимое хранилище), Swagger, миграции и локальные сертификаты для разработки.

---

## Быстрый старт

### 1. Клонируйте репозиторий

```sh
git clone https://github.com/your-org/brigadka-backend.git
cd brigadka-backend
```

### 2. Установите зависимости

- Go 1.21+
- Docker + Docker Compose
- [Colima](https://github.com/abiosoft/colima) (для Mac)
- [jq](https://stedolan.github.io/jq/) (для парсинга JSON)
- [swag](https://github.com/swaggo/swag) (для генерации Swagger)

### 3. Запуск окружения для разработки

```sh
make start-debug-env
```
- Это создаст сертификаты, .env, поднимет все сервисы кроме приложения.
- Для остановки окружения нажмите `Ctrl+C` — все сервисы будут корректно остановлены.

### 4. Запуск приложения

В отдельном терминале:
```sh
make run-debug
```
или запустите из вашей IDE (см. ниже).

---

## Работа с .env и сертификатами

- Файл `.env` создаётся автоматически при запуске `make prepare-debug-env` или `make start-debug-env`.
- В `.env` автоматически прописываются переменные `SSL_CERT_FILE` и `SSL_CERT_DIR` с абсолютными путями к сертификатам.
- **Важно:** Для корректной работы с MinIO по HTTPS используйте только абсолютный путь в `SSL_CERT_FILE`.

---

## Использование в IDE

### VS Code

- Используйте конфиг `.vscode/launch.json`.
- Убедитесь, что рабочая директория (`cwd`) выставлена в корень проекта (`${workspaceFolder}`).

### GoLand

- В настройках Run/Debug Configuration установите **Working directory** в корень проекта.
- Пример: `/Users/yourname/code/brigadka-backend`

---

## Миграции

Создать новую миграцию:
```sh
make migrate-create
```

Применить миграции:
```sh
make migrate-up
```

Откатить миграции:
```sh
make migrate-down
```

---

## Swagger

Сгенерировать Swagger:
```sh
make generate-swagger
```

---

## Тесты

Запуск unit-тестов:
```sh
make run-unit-tests
```

Запуск интеграционных тестов:
```sh
make run-integration-tests
```

---

## Важно

- Все ключи и сертификаты игнорируются `.gitignore`, кроме шаблона `certs/minio/openssl.cnf.template`.
- Не храните приватные ключи в репозитории!
- Если меняется IP Colima, пересоздайте сертификаты:  
  ```sh
  make generate-local-ca
  ```

---

## Контакты

Вопросы и предложения — в Issues или к мейнтейнерам проекта.