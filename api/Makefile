.PHONY: dev migrate seed test lint build

# Load .env if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

# Start Go API with hot reload (air) + Next.js dev server in parallel
dev:
	@echo "Starting API with hot reload..."
	@if command -v air > /dev/null 2>&1; then \
		air & \
	else \
		echo "air not found. Installing..."; \
		go install github.com/air-verse/air@latest; \
		air & \
	fi
	@if [ -d "../web" ] && [ -f "../web/package.json" ]; then \
		echo "Starting Next.js dev server..."; \
		cd ../web && npm run dev & \
	fi
	@wait

# Apply all SQL migrations in order against DATABASE_URL
migrate:
	@echo "Applying migrations..."
	for f in migrations/0*.sql; do \
		echo "Applying $$f..."; \
		psql "$(DATABASE_URL)" -f "$$f" || exit 1; \
	done
	@echo "Migrations applied successfully."

# Insert demo data
seed:
	@echo "Seeding demo data..."
	psql "$(DATABASE_URL)" -f migrations/seed/seed.sql
	@echo "Seed data inserted successfully."

# Run tests with race detector
test:
	go test ./... -race -count=1

# Run linter
lint:
	golangci-lint run

# Build Go binary + Next.js production build
build:
	go build -o bin/server ./cmd/server
	@if [ -d "../web" ] && [ -f "../web/package.json" ]; then \
		echo "Building Next.js..."; \
		cd ../web && npm run build; \
	fi
	@echo "Build complete."