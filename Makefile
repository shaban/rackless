# Build and Development Makefile for Rackless

.PHONY: help server server-dev server-stop frontend frontend-clean frontend-serve frontend-dev standalone standalone-clean test clean css css-watch css-prod

# Default target
help:
	@echo "Rackless - Audio Plugin Parameter Automation System"
	@echo ""
	@echo "Available targets:"
	@echo "  server             - Build and run the Rackless server"
	@echo "  server-dev         - Run server in development mode (with auto-reload)"
	@echo "  server-stop        - Stop any running Rackless server processes"
	@echo "  frontend           - Build frontend (WASM + static files)"
	@echo "  frontend-serve     - Start frontend development server"
	@echo "  frontend-dev       - Start frontend in development mode"
	@echo "  frontend-clean     - Clean frontend build artifacts"
	@echo "  standalone         - Build all standalone tools"
	@echo "  standalone-clean   - Clean standalone build artifacts"
	@echo "  css                - Build Tailwind CSS"
	@echo "  css-watch          - Watch Tailwind CSS for changes"
	@echo "  css-prod           - Build CSS for production"
	@echo "  test               - Run Go tests"
	@echo "  clean              - Clean all build artifacts"

# Server targets
server: rackless
	@echo "Starting Rackless server..."
	./rackless

server-dev:
	@echo "Starting Rackless server in development mode..."
	@echo "Press Ctrl+C to stop the server"
	go run .

server-stop:
	@echo "Stopping Rackless server processes..."
	@pkill -f "rackless" || echo "No Rackless processes found"
	@pkill -f "go run ." || echo "No development server processes found"

# Build the server binary
rackless: *.go
	@echo "Building Rackless server..."
	go build -o rackless .
	@echo "✅ Rackless server binary created"

# Frontend targets
frontend:
	@echo "Building frontend..."
	@if [ -d "frontend" ]; then cd frontend && $(MAKE); else echo "Frontend directory not found"; fi

frontend-clean:
	@echo "Cleaning frontend..."
	@if [ -d "frontend" ]; then cd frontend && $(MAKE) clean; else echo "Frontend directory not found"; fi

frontend-serve:
	@echo "Starting frontend development server..."
	@if [ -d "frontend" ]; then cd frontend && $(MAKE) serve; else echo "Frontend directory not found"; fi

frontend-dev:
	@echo "Starting frontend in development mode..."
	@if [ -d "frontend" ]; then cd frontend && $(MAKE) dev; else echo "Frontend directory not found"; fi

# Standalone tools targets
standalone:
	@echo "Building all standalone tools..."
	@if [ -d "standalone" ]; then cd standalone && $(MAKE); else echo "Standalone directory not found"; fi

standalone-clean:
	@echo "Cleaning standalone tools..."
	@if [ -d "standalone" ]; then cd standalone && $(MAKE) clean; else echo "Standalone directory not found"; fi

# CSS targets
css:
	@echo "Building Tailwind CSS..."
	./bin/tailwindcss -i frontend/src/input.css -o frontend/static/style.css

css-watch:
	@echo "Watching Tailwind CSS for changes..."
	./bin/tailwindcss -i frontend/src/input.css -o frontend/static/style.css --watch

css-prod:
	@echo "Building Tailwind CSS for production..."
	./bin/tailwindcss -i frontend/src/input.css -o frontend/static/style.css --minify

# Test targets
test:
	@echo "Running Go tests..."
	go test ./...

# Clean build artifacts
clean: frontend-clean standalone-clean
	@echo "Cleaning build artifacts..."
	rm -f frontend/static/style.css
	rm -f rackless
