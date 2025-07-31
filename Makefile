# Build and Development Makefile for Rackless

.PHONY: help build wasm frontend frontend-clean frontend-serve frontend-dev dev clean test test-unit test-integration test-bench introspection-test compile-objc

# Default target
help:
	@echo "Rackless - Audio Plugin Parameter Automation System"
	@echo ""
	@echo "Available targets:"
	@echo "  build              - Build native backend server"
	@echo "  wasm               - Build WASM frontend"
	@echo "  frontend           - Build frontend (WASM + static files)"
	@echo "  frontend-serve     - Start frontend development server"
	@echo "  frontend-dev       - Start frontend in development mode"
	@echo "  dev                - Start development server with hot reload"
	@echo "  test               - Run all tests"
	@echo "  test-unit          - Run unit tests only"
	@echo "  test-integration   - Run integration tests"
	@echo "  test-bench         - Run benchmarks (single iteration)"
	@echo "  introspection-test - Test AudioUnit introspection"
	@echo "  device-test        - Test device enumeration"
	@echo "  audiohost-test     - Test Go audio host controller"
	@echo "  audio-host         - Build standalone audio host tool"
	@echo "  clean              - Clean all build artifacts"

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
test: compile-objc
	@echo "Running all tests..."
	go test ./...

# Run unit tests only
test-unit: compile-objc
	@echo "Running unit tests..."
	go test -short ./...

# Run integration tests
test-integration: compile-objc
	@echo "Running integration tests..."
	go test -run Integration ./...

# Run benchmarks (single iteration for expensive operations)
test-bench: compile-objc
	@echo "Running benchmarks..."
	go test -bench=. -benchtime=1x ./pkg/introspection
	go test -bench=. -benchtime=1x ./pkg/devices

# Test AudioUnit introspection
introspection-test: compile-objc
	@echo "Building and running AudioUnit introspection test..."
	go build -o bin/introspection-test ./cmd/introspection-test
	./bin/introspection-test

# Test device enumeration
device-test: compile-objc
	@echo "Building and running device enumeration test..."
	go build -o bin/device-test ./cmd/device-test
	./bin/device-test

# Test Go audio host controller
audiohost-test: audio-host
	@echo "Building and running Go audio host controller test..."
	go build -o bin/audiohost-test ./cmd/audiohost-test
	./bin/audiohost-test

# Build standalone audio host tool
audio-host:
	@echo "Building standalone audio host..."
	$(MAKE) -C standalone-audio-host

# Compile Objective-C bridge code
compile-objc:
	@echo "Compiling Objective-C AudioUnit bridge..."
	@mkdir -p bin
	# Compile AudioUnit introspection bridge
	clang -c -x objective-c -o pkg/audio/audiounit_inspector.o pkg/audio/audiounit_inspector.m \
		-framework Foundation -framework AudioToolbox -framework AVFoundation -framework AudioUnit
	ar rcs pkg/audio/libaudiounit_inspector.a pkg/audio/audiounit_inspector.o
	# Compile device enumeration bridge
	clang -c -x objective-c -o pkg/audio/device_enumerator.o pkg/audio/device_enumerator.m \
		-framework Foundation -framework CoreAudio -framework AudioToolbox -framework CoreMIDI
	ar rcs pkg/audio/libaudiounit_devices.a pkg/audio/device_enumerator.o
	# Compile audio host bridge
	clang -c -x objective-c -o pkg/audio/audiounit_host.o pkg/audio/audiounit_host.m \
		-framework Foundation -framework CoreAudio -framework AudioToolbox -framework AVFoundation
	ar rcs pkg/audio/libaudiounit_host.a pkg/audio/audiounit_host.o

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

# Clean build artifacts
clean: frontend-clean
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f $(WASM_FILE)
	rm -f pkg/audio/*.o pkg/audio/*.a
	rm -f *.o *.a
