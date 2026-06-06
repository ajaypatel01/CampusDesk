.PHONY: run migrate-up migrate-down test build docker-up

DATABASE_URL ?= postgres://postgres:root@localhost:5432/school?sslmode=disable

run:
	go run ./cmd/server

build:
	go build -o bin/campusdesk ./cmd/server

test:
	go test ./...

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

docker-up:
	docker compose up -d
