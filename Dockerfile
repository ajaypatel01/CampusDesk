# syntax=docker/dockerfile:1

# ── Stage 1: build Go binary ──────────────────────────────────────────────────
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/campusdesk ./cmd/server

# ── Stage 2: minimal runtime image ────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/campusdesk .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./campusdesk"]
