.PHONY: run migrate-up migrate-down test build docker-up

DATABASE_URL ?= postgres://campusdesk:campusdesk@localhost:5432/campusdesk?sslmode=disable

run: build
	@pkill -f bin/campusdesk 2>/dev/null; sleep 0.5; true
	./bin/campusdesk

dev: build
	@pkill -f bin/campusdesk 2>/dev/null; sleep 0.5; true
	./bin/campusdesk &
	cd frontend && node_modules/.bin/vite

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
