# Загружаем переменные из .env файла
ifneq ("$(wildcard .env)","")
	include .env
	export
endif

run:
	go run ./cmd/service

build:
	go build -o bin/app ./cmd/service

build-release:
	CGO_ENABLED=0 go build -tags netgo -ldflags "-s -w" -o bin/app ./cmd/service

run-release:
	GIN_MODE=release ./bin/app

run-local:
	docker compose up --build

run-unit-tests:
	go test ./internal/...

KEEP ?= false

start-integration-env:
	docker compose -f docker-compose.test.yml up -d --build test-postgres test-migrations test-app

down-integration-env:
	docker compose -f docker-compose.test.yml down -v

run-integration:
	docker compose -f docker-compose.test.yml run --rm tests

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
	docker exec -it brigadka-backend-test-postgres-1 psql -U ${DB_USER} -d ${DB_NAME}