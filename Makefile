# Strength Tracker Makefile
# Usage: make [target]

# Variables
APP_NAME=stren
DB_FILE=strength_tracker.db
SEED_FILE=seed.sql
PORT=8080

# Default target
.PHONY: help
help:
	@echo "Strength Tracker - Available commands:"
	@echo ""
	@echo "  make build       - Build the application"
	@echo "  make run         - Build and run the application"
	@echo "  make dev         - Run with auto-restart (requires entr)"
	@echo "  make seed        - Seed the database with exercises"
	@echo "  make db-reset    - Delete and recreate database with seed data"
	@echo "  make test        - Run tests"
	@echo "  make test-cover  - Run tests with coverage report"
	@echo "  make templ       - Regenerate Templ templates"
	@echo "  make clean       - Remove built binary and database"
	@echo ""

# Build the application
.PHONY: build
build: templ
	@echo "Building $(APP_NAME)..."
	go build -o $(APP_NAME) ./cmd/main.go
	@echo "✓ Build complete: ./$(APP_NAME)"

# Build and run
.PHONY: run
run: build
	@echo "Starting server on http://localhost:$(PORT)"
	./$(APP_NAME)

# Development mode with auto-restart (requires entr)
.PHONY: dev
dev:
	@if command -v entr >/dev/null 2>&1; then \
		echo "Watching for changes..."; \
		find . -name '*.go' -o -name '*.templ' | entr -r $(MAKE) run; \
	else \
		echo "Error: 'entr' not installed. Install with: brew install entr (macOS) or apt-get install entr (Linux)"; \
		exit 1; \
	fi

# Regenerate Templ templates
.PHONY: templ
templ:
	@echo "Generating Templ templates..."
	@which templ > /dev/null || (echo "Installing templ..." && go install github.com/a-h/templ/cmd/templ@latest)
	templ generate
	@echo "✓ Templates generated"

# Seed database
.PHONY: seed
seed:
	@echo "Seeding database..."
	@if [ ! -f $(DB_FILE) ]; then \
		echo "Creating database..."; \
	fi
	sqlite3 $(DB_FILE) < $(SEED_FILE)
	@echo "✓ Database seeded"

# Reset database (delete and recreate)
.PHONY: db-reset
db-reset:
	@echo "Resetting database..."
	@rm -f $(DB_FILE)
	sqlite3 $(DB_FILE) < $(SEED_FILE)
	@echo "✓ Database reset complete"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-cover
test-cover:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"
