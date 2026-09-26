.PHONY: build run test lint fmt clean seed seed-data deps migrate-status migrate-up migrate-baseline

build:
	go build ./...

run:
	go run ./cmd/server

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

clean:
	rm -f server
	rm -rf tmp/

seed:
	go run ./cmd/seed

deps:
	go mod tidy

# Migrations are plain, forward-only SQL files in migrations/, applied in
# filename order by cmd/migrate (tracked in the schema_migrations table).
migrate-status:
	go run ./cmd/migrate -status

migrate-up:
	go run ./cmd/migrate

# For a database whose schema was created by hand (e.g. applied with psql):
# record every existing migration as applied without executing it.
migrate-baseline:
	go run ./cmd/migrate -baseline

# Demo data (one tenant + sample learners, guardians, staff, invoices, ...).
# Reuses the migration runner against migrations/seed/, so it is applied once
# and recorded in schema_migrations; cmd/seed then creates the matching
# Supabase Auth users for the demo logins.
seed-data:
	go run ./cmd/migrate -dir migrations/seed
