# Загружаем переменные из .env файла
ifneq ("$(wildcard .env)","")
	include .env
	export
endif

build-release:
	CGO_ENABLED=0 go build -tags netgo -ldflags "-s -w" -o bin/app ./cmd/service

build-debug:
	CGO_ENABLED=0 go build -gcflags "all=-N -l" -o bin/app-debug ./cmd/service

run-release:
	GIN_MODE=release ./bin/app

run-unit-tests:
	go test ./internal/...


INTEGRATION_DETACH ?= true

start-integration-env:
	docker compose -f docker-compose.test.yml up $(if $(filter true,$(INTEGRATION_DETACH)),-d,) --force-recreate --build test-postgres test-migrations test-app test-minio --remove-orphans

down-integration-env:
	docker compose -f docker-compose.test.yml down -v --remove-orphans

run-integration:
	docker compose -f docker-compose.test.yml run --build --rm tests

KEEP ?= false

run-integration-tests:
	@bash -c '\
	set -e; \
	$(MAKE) start-integration-env; \
	trap " \
		if [ "$$KEEP" != "true" ]; then \
			echo Cleaning up containers...; \
			$(MAKE) down-integration-env; \
		else \
			echo Skipping docker compose down because KEEP=true; \
		fi" EXIT; \
	$(MAKE) run-integration; \
	'

# Миграции
migrate-up:
	go run ./cmd/migrate -up

migrate-down:
	go run ./cmd/migrate -down

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir db/migrations -seq $$name

# База данных
connect-local-db:
	docker exec -it brigadka-backend-postgres-1 psql -U ${DB_USER} -d ${DB_NAME}

connect-local-test-db:
	docker exec -it brigadka-backend-test-postgres-1 psql -U test_user -d test_db

connect-db:
	PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME}

generate-swagger:
	swag init -g cmd/service/main.go

prepare-debug-env:
	@echo "Fetching Colima IP..."
	@COLIMA_IP=$$(colima status --json | jq -r '.ip_address'); \
	if [ -z "$$COLIMA_IP" ]; then \
		echo "Failed to fetch Colima IP. Ensure Colima is running."; \
		exit 1; \
	fi; \
	echo "Colima IP: $$COLIMA_IP"; \
	echo "Updating .env with DB_HOST=$$COLIMA_IP"; \
	sed -i.bak "s/^DB_HOST=.*/DB_HOST=$$COLIMA_IP/" .env && rm .env.bak

start-debug-env: prepare-debug-env
	@echo "Starting services except app..."
	@echo "Press Ctrl+C to stop the debug environment"
	docker compose down -f docker-compose.local.yml -v --remove-orphans
	docker compose up -f docker-compose.local.yml postgres migrations --remove-orphans --build --force-recreate

.PHONY: prepare-debug-env start-debug-env

generate-local-ca:
	@echo "Создаём директорию certs и подпапки..."
	mkdir -p certs/ca certs/minio
	@echo "Генерируем CA ключ и сертификат..."
	openssl genrsa -out certs/ca/ca.key 4096
	openssl req -x509 -new -nodes -key certs/ca/ca.key -sha256 -days 3650 -out certs/ca/ca.crt -subj "/C=RU/ST=Local/L=Local/O=Local CA/CN=Local CA"
	@echo "Создаём конфиг для SAN..."
	echo "[req]" > certs/minio/openssl.cnf
	echo "distinguished_name = req_distinguished_name" >> certs/minio/openssl.cnf
	echo "req_extensions = v3_req" >> certs/minio/openssl.cnf
	echo "[req_distinguished_name]" >> certs/minio/openssl.cnf
	echo "[v3_req]" >> certs/minio/openssl.cnf
	echo "subjectAltName = @alt_names" >> certs/minio/openssl.cnf
	echo "[alt_names]" >> certs/minio/openssl.cnf
	echo "DNS.1 = localhost" >> certs/minio/openssl.cnf
	echo "DNS.2 = test-minio" >> certs/minio/openssl.cnf
	@echo "Генерируем ключ для MinIO..."
	openssl genrsa -out certs/minio/private.key 4096
	@echo "Генерируем CSR для MinIO с SAN..."
	openssl req -new -key certs/minio/private.key -out certs/minio/minio.csr -subj "/C=RU/ST=Local/L=Local/O=MinIO/CN=test-minio" -config certs/minio/openssl.cnf
	@echo "Подписываем сертификат MinIO нашим CA с SAN..."
	openssl x509 -req -in certs/minio/minio.csr -CA certs/ca/ca.crt -CAkey certs/ca/ca.key -CAcreateserial -out certs/minio/public.crt -days 3650 -sha256 -extensions v3_req -extfile certs/minio/openssl.cnf
	@echo "Готово! CA и серверные сертификаты лежат в certs/ca и certs/minio"