.PHONY: build run test lint fmt clean seed

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

migrate-up:
	# Apply pending migrations (requires a migration runner)
	@echo "Run migrations via your migration tool"

migrate-down:
	# Rollback last migration
	@echo "Run rollback via your migration tool"
