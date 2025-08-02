# Makefile for Standalone Audio Host
# Pure Objective-C command line tool

CC = clang
CFLAGS = -x objective-c -fobjc-arc -O2
FRAMEWORKS = -framework Foundation -framework AudioToolbox -framework CoreAudio -framework AVFoundation -framework CoreMIDI -framework AudioUnit
TARGET = audio-host
SOURCE = main.m

.PHONY: all clean run test test-silent test-commands test-enumeration help

all: $(TARGET)

$(TARGET): $(SOURCE)
	@echo "🔨 Building standalone audio host..."
	$(CC) $(CFLAGS) $(FRAMEWORKS) -o $(TARGET) $(SOURCE)
	@echo "✅ Build complete: ./$(TARGET)"

clean:
	@echo "🧹 Cleaning..."
	rm -f $(TARGET)

run: $(TARGET)
	@echo "🎵 Running audio host..."
	./$(TARGET)

test: $(TARGET)
	@echo "🔊 Testing audio host (5 seconds)..."
	@if command -v gtimeout >/dev/null 2>&1; then \
		gtimeout 5s ./$(TARGET) || true; \
	else \
		echo "⚠️  gtimeout not found. Run manually: ./$(TARGET)"; \
		echo "   (install with: brew install coreutils)"; \
	fi

test-silent: $(TARGET)
	@echo "🔇 Testing silent mode (3 seconds)..."
	@if command -v gtimeout >/dev/null 2>&1; then \
		gtimeout 3s ./$(TARGET) --no-tone || true; \
	else \
		echo "⚠️  gtimeout not found. Run manually: ./$(TARGET) --no-tone"; \
		echo "   (install with: brew install coreutils)"; \
	fi

test-commands: $(TARGET)
	@echo "🎛️  Testing command mode..."
	@(echo "start"; sleep 1; echo "status"; sleep 1; echo "tone freq 880"; sleep 1; echo "status"; sleep 1; echo "tone off"; sleep 2; echo "tone on"; sleep 1; echo "quit") | ./$(TARGET) --command-mode

test-enumeration: $(TARGET)
	@echo "🔍 Testing device and plugin enumeration..."
	@(echo "devices audio-input"; echo "devices audio-output"; echo "devices midi-input"; echo "devices midi-output"; echo "plugins"; echo "quit") | ./$(TARGET) --command-mode

test-plugins: $(TARGET)
	@echo "🔌 Getting plugin JSON..."
	@echo "plugins" | ./$(TARGET) --command-mode

help:
	@echo "Standalone Audio Host - Build Options"
	@echo "====================================="
	@echo "  make          - Build the audio host"
	@echo "  make run      - Build and run interactively"
	@echo "  make test     - Build and test for 5 seconds"
	@echo "  make test-silent - Build and test silent mode"
	@echo "  make test-commands - Test command mode"
	@echo "  make test-enumeration - Test device and plugin enumeration"
	@echo "  make clean    - Clean build artifacts"
	@echo ""
	@echo "Runtime Options:"
	@echo "  ./audio-host --help"
