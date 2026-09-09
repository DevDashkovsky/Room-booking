.PHONY: up down test test-race coverage lint test-integration test-e2e

COMPOSE := $(shell docker compose version >/dev/null 2>&1 && echo 'docker compose' || echo 'docker-compose')

up:
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

test:
	go test ./...

test-race:
	go test -race ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...
	golangci-lint fmt --diff

test-integration:
	@test -n "$$ROOM_BOOKING_TEST_DATABASE_URL" || (echo 'Set ROOM_BOOKING_TEST_DATABASE_URL to a test PostgreSQL database'; exit 1)
	go test -race ./... -count=1

test-e2e:
	sh scripts/test-e2e.sh
