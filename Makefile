.PHONY: help test lint fmt up down

help:
	@echo "Targets:"
	@echo "  test  - go test ./..."
	@echo "  lint  - golangci-lint run ./..."
	@echo "  fmt   - gofmt -w ."
	@echo "  up    - docker compose (postgres only for now)"
	@echo "  down  - docker compose down"

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

up:
	docker compose -f db/postgres/docker-compose.yml up -d

down:
	docker compose -f db/postgres/docker-compose.yml down

