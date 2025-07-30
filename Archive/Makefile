# Makefile for MC-SoFX Controller

.PHONY: all build server clean help tidy css css-watch css-prod test test-server test-packages test-all test-coverage

# Default target
all: build

# Build all tools
build: build-server css

# Build the consolidated server
build-server:
	@echo "Building consolidated server..."
	go build -o mc-sofx-consolidated .

# Web server commands
server: build-server

# Start the web server
start-server: build-server
	@echo "Starting web server on http://localhost:8080"
	./mc-sofx-consolidated

# Start server on custom port
start-server-port: build-server
	@echo "Usage: make start-server-port PORT=8080"
	@if [ -z "$(PORT)" ]; then echo "Error: PORT parameter is required"; exit 1; fi
	@echo "Starting web server on http://localhost:$(PORT)"
	./mc-sofx-consolidated $(PORT)

# CSS build commands
css:
	@echo "Building Tailwind CSS..."
	./bin/tailwindcss -i frontend/src/input.css -o frontend/static/style.css

css-watch:
	@echo "Watching Tailwind CSS for changes..."
	./bin/tailwindcss -i frontend/src/input.css -o frontend/static/style.css --watch

css-prod:
	@echo "Building Tailwind CSS for production..."
	./bin/tailwindcss -i frontend/src/input.css -o frontend/static/style.css --minify

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f mc-sofx-consolidated
	rm -f frontend/static/style.css

# Run the frontend development server (if needed later)
frontend:
	@echo "Starting frontend development server..."
	# TODO: Add frontend build/serve commands

# Tidy Go modules
tidy:
	go mod tidy

# Testing targets
test: test-packages

test-packages:
	@echo "Running package tests..."
	go test -v ./pkg/...

test-server:
	@echo "Running server tests..."
	go test -v .

test-all: test-packages test-server
	@echo "Running all tests with race detection..."
	go test -race -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Help target
help:
	@echo "Available targets:"
	@echo "  build             - Build all tools (server + CSS)"
	@echo "  build-server      - Build the web server"
	@echo "  server            - Build web server"
	@echo "  start-server      - Start web server on port 8080"
	@echo "  start-server-port - Start web server on custom port (requires PORT=port)"
	@echo "  css               - Build Tailwind CSS"
	@echo "  css-watch         - Watch Tailwind CSS for changes"
	@echo "  css-prod          - Build Tailwind CSS for production (minified)"
	@echo "  test              - Run all tests (packages + server)"
	@echo "  test-packages     - Run package tests only"
	@echo "  test-server       - Run server tests only"
	@echo "  test-all          - Run all tests with race detection"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  clean             - Clean build artifacts"
	@echo "  tidy              - Tidy Go modules"
	@echo "  help              - Show this help message"
