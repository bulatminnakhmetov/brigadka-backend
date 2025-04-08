FROM golang:1.22-alpine

WORKDIR /app

# Install curl for healthcheck
RUN apk add --no-cache curl

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/service

EXPOSE 8080

CMD ["./main"] 