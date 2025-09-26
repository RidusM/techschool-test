FROM golang:1.25.0-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN go test -c ./tests/integration/main -o integration-test

FROM alpine:latest

COPY --from=builder /app/integration-test ./integration-test
COPY --from=builder /app/configs ./configs

CMD ["./integration-test", "-test.v"]