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
	docker-compose up --build

run-unit-tests:
	go test ./internal/...
	
KEEP ?= false

run-integration-tests:
	@bash -c '\
	set -e; \
	docker-compose -f docker-compose.test.yml up -d test-postgres test-migrations test-app; \
	trap " \
		if [ "$$KEEP" != "true" ]; then \
			echo Cleaning up containers...; \
			docker-compose -f docker-compose.test.yml down; \
		else \
			echo Skipping docker-compose down because KEEP=true; \
		fi" EXIT; \
	docker-compose -f docker-compose.test.yml run --rm tests; \
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
db-connect:
	docker exec -it brigadka-backend-postgres-1 psql -U ${DB_USER} -d ${DB_NAME}

