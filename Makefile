run:
	go run ./cmd/service

build:
	go build -o bin/app ./cmd/service

build-release:
	CGO_ENABLED=0 go build -tags netgo -ldflags "-s -w" -o bin/app ./cmd/service

run-release:
	GIN_MODE=release ./bin/app

test:
	go test ./...

	