# Build and Development Makefile for Rackless

.PHONY: help build wasm dev clean test

# Default target
help:
	@echo "Rackless - Audio Plugin Parameter Automation System"
	@echo ""
	@echo "Available targets:"
	@echo "  build     - Build native backend server"
	@echo "  wasm      - Build WASM frontend"
	@echo "  dev       - Start development server with hot reload"
	@echo "  test      - Run all tests"
	@echo "  clean     - Clean build artifacts"
	@echo "  css       - Build Tailwind CSS (legacy)"
	@echo "  css-watch - Watch and build Tailwind CSS (legacy)"

# Go build settings
GO_FILES := $(shell find . -name '*.go' -not -path './Archive/*')
WASM_DIR := web/static
WASM_FILE := $(WASM_DIR)/app.wasm

# Build native backend
build:
	@echo "Building native backend..."
	go build -o bin/rackless ./cmd/server

# Build WASM frontend
wasm: $(WASM_FILE)

$(WASM_FILE): $(GO_FILES)
	@echo "Building WASM frontend..."
	@mkdir -p $(WASM_DIR)
	GOOS=js GOARCH=wasm go build -o $(WASM_FILE) ./cmd/wasm

# Development server with hot reload
dev: wasm
	@echo "Starting development server..."
	./bin/rackless -dev

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f $(WASM_FILE)
	rm -f *.o *.a

# Legacy CSS build targets (keep for reference during migration)
css:
	@echo "Building Tailwind CSS (legacy)..."
	./bin/tailwindcss -i frontend/src/input.css -o frontend/static/style.css

css-watch:
	@echo "Watching Tailwind CSS (legacy)..."
	./bin/tailwindcss -i frontend/src/input.css -o frontend/static/style.css --watch

css-prod:
	@echo "Building production CSS (legacy)..."
	./bin/tailwindcss -i frontend/src/input.css -o frontend/static/style.css --minify
