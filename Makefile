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

run-local:
	docker compose up --build --remove-orphans

run-unit-tests:
	go test ./internal/...

KEEP ?= false

start-integration-env:
	docker compose -f docker-compose.test.yml up -d --force-recreate --build test-postgres test-migrations test-app --remove-orphans

down-integration-env:
	docker compose -f docker-compose.test.yml down -v --remove-orphans

run-integration:
	docker compose -f docker-compose.test.yml run --build --rm tests

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
	docker compose down -v --remove-orphans
	docker compose up postgres migrations --remove-orphans --build --force-recreate

.PHONY: prepare-debug-env start-debug-env