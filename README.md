# Brigadka Backend

This repository contains the backend service for Brigadka application. The service is built in Go and provides API endpoints for client applications.

## Features

- Authentication and user management
- Profile management
- Messaging
- Media handling
- Catalog services
- Push notifications

## Prerequisites

- Go 1.22 or later
- Docker and Docker Compose
- Make
- PostgreSQL (for local development without Docker)
- MinIO (or compatible S3 storage for local development without Docker)

## Getting Started

### Setting up the development environment

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd brigadka-backend
   ```

2. If you use Colima, run the following to ensure containers are accessible via localhost:
   ```bash
   colima stop
   colima start --network-address
   ```

3. Start the development environment:
   ```bash
   make start-debug-env
   ```
   This starts all required services (PostgreSQL, MinIO) in Docker containers.

4. Run the application:
   ```bash
   go run cmd/service/main.go
   ```

### Configuration

Configuration is done through environment variables. See .env.debug and .env.docker for examples.

Key configuration options:
- Database connection (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)
- S3 storage (B2_ACCESS_KEY_ID, B2_SECRET_ACCESS_KEY, B2_ENDPOINT, B2_BUCKET_NAME)
- Application settings (APP_PORT)

## Development

### Build commands

- Build release version: `make build-release`
- Build debug version: `make build-debug`
- Run release version: `make run-release`
- Run debug version: `make run-debug`

### Database migrations

- Apply migrations: `make migrate-up`
- Rollback last migration: `make migrate-down`
- Create new migration: `make migrate-create`
- Connect to the database: `make connect-db`

### API Documentation

Generate Swagger documentation:
```bash
make generate-swagger
```

This will create documentation in the http directory.

### Testing

- Run unit tests: `make run-unit-tests`
- Run integration tests: `make run-integration-tests`
- Run integration tests with debug logs: `make run-integration-tests DEBUG-ENV=1`

## Docker Support

The application is containerized and can be run using Docker Compose:

- Development profile: `docker compose --profile debug up`
- Testing profile: `docker compose --profile test up`

## Certificate Management

For local development with secure connections:

- Generate certificates: `make generate-local-ca`
- Install CA certificate in Android emulator: `make install-ca-android`

## CI/CD

The project uses GitHub Actions for continuous integration. You can test GitHub Actions locally:

```bash
make run-gh-actions
```
