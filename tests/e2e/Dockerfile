FROM golang:1.25.0-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY internal/entity/ ./internal/entity/
COPY tests/e2e/ ./tests/e2e/

CMD ["go", "test", "-v", "./tests/e2e/main"]