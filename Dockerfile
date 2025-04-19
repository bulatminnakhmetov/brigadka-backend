FROM golang:1.23-alpine

WORKDIR /app

RUN apk add --no-cache curl ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/service

# Копируем скрипт
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]
CMD ["./main"]
