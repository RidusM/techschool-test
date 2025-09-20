FROM golang:1.25.0-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY tests/integration/ ./tests/integration/

CMD ["go", "test", "-v", "./tests/integration/main"]