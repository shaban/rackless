# Rackless Frontend Build System

GOOS := js
GOARCH := wasm
GO_SRC := $(wildcard src/*.go src/**/*.go)
WASM_TARGET := static/main.wasm
WASM_EXEC_JS := static/wasm_exec.js

.PHONY: all clean serve dev watch help

# Default target
all: $(WASM_TARGET) $(WASM_EXEC_JS)

# Build WASM binary
$(WASM_TARGET): $(GO_SRC)
	@echo "🔨 Building WASM frontend..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(WASM_TARGET) ./src
	@echo "✅ WASM build complete: $(WASM_TARGET)"

# Copy WASM runtime
$(WASM_EXEC_JS):
	@echo "📋 Copying WASM runtime..."
	cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" $(WASM_EXEC_JS)

# Development server
serve: all
	@echo "🌐 Starting development server on http://localhost:8081"
	@echo "📁 Serving from: $(PWD)/static"
	cd static && python3 -m http.server 8081

# Development mode (rebuild on changes)
dev: all
	@echo "👀 Watching for changes... (Ctrl+C to stop)"
	@while true; do \
		$(MAKE) -q all || $(MAKE) all; \
		sleep 1; \
	done

# Clean build artifacts
clean:
	@echo "🧹 Cleaning frontend build..."
	rm -f $(WASM_TARGET) $(WASM_EXEC_JS)

# Help
help:
	@echo "Rackless Frontend Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all     - Build WASM and copy runtime"
	@echo "  serve   - Start development server"
	@echo "  dev     - Watch mode (rebuild on changes)"
	@echo "  clean   - Clean build artifacts"
	@echo "  help    - Show this help"
