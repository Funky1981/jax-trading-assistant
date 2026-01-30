.PHONY: help test lint fmt up down knowledge-up knowledge-down knowledge-schema knowledge-ingest

# Default DSN for knowledge database
KNOWLEDGE_DSN ?= postgres://postgres:postgres@localhost:5432/jax_knowledge?sslmode=disable

help:
	@echo "Targets:"
	@echo "  test              - go test ./..."
	@echo "  lint              - golangci-lint run ./..."
	@echo "  fmt               - gofmt -w ."
	@echo "  up                - docker compose (main postgres)"
	@echo "  down              - docker compose down"
	@echo ""
	@echo "Knowledge Base:"
	@echo "  knowledge-up      - start knowledge Postgres via docker compose"
	@echo "  knowledge-down    - stop knowledge Postgres"
	@echo "  knowledge-schema  - apply schema to jax_knowledge DB"
	@echo "  knowledge-ingest  - run ingestor against knowledge/md"

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

# Knowledge Base targets
knowledge-up:
	docker compose -f tools/docker-compose.yml up -d postgres
	@echo "Waiting for postgres to be ready..."
	@sleep 3
	@docker compose -f tools/docker-compose.yml exec -T postgres pg_isready -U postgres -d jax_knowledge || sleep 2

knowledge-down:
	docker compose -f tools/docker-compose.yml down

knowledge-schema:
	@echo "Applying schema to jax_knowledge..."
	docker compose -f tools/docker-compose.yml exec -T postgres psql -U postgres -d jax_knowledge -f /dev/stdin < tools/sql/schema.sql

knowledge-ingest:
	@echo "Ingesting knowledge from knowledge/md..."
	cd tools && go run ./cmd/ingest --root ../knowledge/md --dsn "$(KNOWLEDGE_DSN)" --dry-run=false

knowledge-ingest-dry:
	@echo "Dry-run ingesting knowledge from knowledge/md..."
	cd tools && go run ./cmd/ingest --root ../knowledge/md --dsn "$(KNOWLEDGE_DSN)" --dry-run=true

