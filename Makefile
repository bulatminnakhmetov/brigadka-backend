# --- Загрузка переменных окружения из .env ---
ifneq ("$(wildcard .env)","")
	include .env
	export
endif

# --- Сборка приложения ---
build-release:
	# Сборка релизной версии (оптимизированная, без отладочной информации)
	CGO_ENABLED=0 go build -tags netgo -ldflags "-s -w" -o bin/app ./cmd/service

build-debug:
	# Сборка отладочной версии (без оптимизаций, с дебаг-инфой)
	CGO_ENABLED=0 go build -gcflags "all=-N -l" -o bin/app-debug ./cmd/service

# --- Запуск приложения ---
run-release: build-release
	# Запуск релизной версии с переменной окружения GIN_MODE=release
	GIN_MODE=release ./bin/app

run-debug: build-debug
	# Запуск отладочной версии
	./bin/app-debug

# --- Тесты ---
run-unit-tests:
	# Запуск юнит-тестов
	go test ./internal/...

run-integration-tests: generate-local-ca
	cp .env.example .env
	# Запуск интеграционных тестов в Docker
	@docker compose --profile test up --build --force-recreate --remove-orphans -d || { \
		echo "❌ Ошибка во время запуска тестов"; exit 1; } ; \
	docker compose logs -f tests & \
	docker compose wait tests; \
	docker compose --profile test down -v --remove-orphans

# --- Миграции базы данных ---
migrate-up:
	# Применить все новые миграции
	go run ./cmd/migrate -up

migrate-down:
	# Откатить последнюю миграцию
	go run ./cmd/migrate -down

migrate-create:
	# Создать новую миграцию (запросит имя)
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir db/migrations -seq $$name

# --- Подключение к базе данных ---
connect-local-db:
	# Подключение к локальной БД в Docker-контейнере
	docker exec -it brigadka-backend-postgres-1 psql -U ${DB_USER} -d ${DB_NAME}

connect-db:
	# Подключение к БД по параметрам из .env
	PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME}

# --- Swagger ---
generate-swagger:
	# Генерация swagger-документации
	swag init -g cmd/service/main.go

# --- Подготовка окружения для отладки ---
prepare-debug-env: generate-local-ca
	# Копируем пример .env в рабочий .env
	cp .env.example .env
	@echo "Detecting Docker environment..."
	# Получаем IP-адрес Colima, если он используется, иначе используем localhost (Docker Desktop)
	@DOCKER_HOST_IP=$$(colima status --json 2>/dev/null | jq -r '.ip_address' || echo ""); \
	if [ -z "$$DOCKER_HOST_IP" ] || [ "$$DOCKER_HOST_IP" = "null" ]; then \
		echo "Colima не найден, используем localhost для Docker Desktop"; \
		DOCKER_HOST_IP=localhost; \
	else \
		echo "Colima IP: $$DOCKER_HOST_IP"; \
	fi; \
	# Абсолютный путь к CA-сертификату
	ABS_CERT_PATH=$$(cd certs/ca && pwd)/ca.crt; \
	# Обновляем DB_HOST в .env на IP Docker-хоста
	echo "Updating .env with DB_HOST=$$DOCKER_HOST_IP"; \
	sed -i.bak "s/^DB_HOST=.*/DB_HOST=$$DOCKER_HOST_IP/" .env && rm .env.bak; \
	# Обновляем B2_ENDPOINT в .env на IP Docker-хоста с портом 9000
	echo "Updating .env with B2_ENDPOINT=$$DOCKER_HOST_IP:9000"; \
	sed -i.bak "s/^B2_ENDPOINT=.*/B2_ENDPOINT=$$DOCKER_HOST_IP:9000/" .env && rm .env.bak; \
	# Обновляем или добавляем SSL_CERT_FILE в .env на путь к CA-сертификату
	echo "Updating .env with SSL_CERT_FILE=$$ABS_CERT_PATH"; \
	if grep -q '^SSL_CERT_FILE=' .env; then \
		sed -i.bak "s|^SSL_CERT_FILE=.*|SSL_CERT_FILE=$$ABS_CERT_PATH|" .env && rm .env.bak; \
	else \
		echo "SSL_CERT_FILE=$$ABS_CERT_PATH" >> .env; \
	fi

start-debug-env: prepare-debug-env
	# Запуск всех сервисов кроме приложения для отладки
	@echo "Starting services except app...";
	@echo "Press Ctrl+C to stop the debug environment";
	@trap 'docker compose --profile debug down -v --remove-orphans; exit' INT; \
	docker compose --profile debug up -d --build --force-recreate && \
	docker compose wait minio-init migrations && \
	echo "✅ \033[1;32mДебаг-окружение готово! Теперь можно запустить сервис (например, make run-debug) и подебажить. Нажмите CTRL + C, чтобы остановить окружение.\033[0m"; \
	while true; do sleep 1; done

.PHONY: prepare-debug-env start-debug-env

# --- Генерация локального CA и сертификатов для MinIO ---
generate-local-ca:
	@echo "🔧 \033[1;34mГенерируем CA и серверные сертификаты...\033[0m"
	# Создаём директории для CA и MinIO сертификатов
	mkdir -p certs/ca certs/minio
	# Генерируем приватный ключ CA
	openssl genrsa -out certs/ca/ca.key 4096
	# Генерируем самоподписанный CA сертификат
	openssl req -x509 -new -nodes -key certs/ca/ca.key -sha256 -days 3650 -out certs/ca/ca.crt -subj "/C=RU/ST=Local/L=Local/O=Local CA/CN=Local CA"
	# Генерируем приватный ключ для MinIO
	openssl genrsa -out certs/minio/private.key 4096
	# Получаем IP-адрес Docker-хоста для сертификата
	@DOCKER_HOST_IP=$$(colima status --json 2>/dev/null | jq -r '.ip_address' || echo ""); \
	if [ -z "$$DOCKER_HOST_IP" ] || [ "$$DOCKER_HOST_IP" = "null" ]; then \
		echo "Colima не найден, используем localhost для Docker Desktop"; \
		DOCKER_HOST_IP=localhost; \
	else \
		echo "Colima IP: $$DOCKER_HOST_IP"; \
	fi; \
	# Подставляем IP в шаблон openssl.cnf для MinIO
	cat certs/minio/openssl.cnf.template | sed "s/{{DOCKER_HOST_IP}}/$$DOCKER_HOST_IP/g" > certs/minio/openssl.cnf; \
	# Генерируем CSR для MinIO с нужным CN и SAN
	openssl req -new -key certs/minio/private.key -out certs/minio/minio.csr -subj "/C=RU/ST=Local/L=Local/O=MinIO/CN=minio" -config certs/minio/openssl.cnf; \
	# Подписываем CSR MinIO нашим CA
	openssl x509 -req -in certs/minio/minio.csr -CA certs/ca/ca.crt -CAkey certs/ca/ca.key -CAcreateserial -out certs/minio/public.crt -days 3650 -sha256 -extensions v3_req -extfile certs/minio/openssl.cnf
	@echo "✅ \033[1;32mГотово! CA и серверные сертификаты лежат в certs/ca и certs/minio\033[0m"

# --- Запуск Github Actions локально через act ---
run-gh-actions:
	@if [ ! -S /var/run/docker.sock ]; then \
		echo "❌ \033[1;31mНе найден /var/run/docker.sock\033[0m"; \
		echo "Если вы используете Colima, создайте симлинк командой:"; \
		echo "  \033[1;33msudo ln -s ~/.colima/default/docker.sock /var/run/docker.sock\033[0m"; \
		echo "Если вы используете Docker Desktop, откройте Docker Desktop → Settings → Advanced и отключите, затем снова включите опцию 'Use the default socket path'.\n"; \
		exit 1; \
	fi; \
	act -j integration-tests --container-architecture linux/amd64