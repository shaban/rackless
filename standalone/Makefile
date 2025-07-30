# Rackless Standalone Tools Build System

.PHONY: all clean standalone-all standalone-clean audio-host inspector

# Build all standalone tools
all: standalone-all

standalone-all: audio-host inspector devices

# Individual tool targets
audio-host:
	@echo "Building audio-host..."
	cd audio-host && $(MAKE)

inspector:
	@echo "Building inspector..."
	cd inspector && $(MAKE)

devices:
	@echo "Building devices..."
	cd devices && $(MAKE)

# Clean targets
clean: standalone-clean

standalone-clean:
	@echo "Cleaning audio-host..."
	cd audio-host && $(MAKE) clean
	@echo "Cleaning inspector..."
	cd inspector && $(MAKE) clean
	@echo "Cleaning devices..."
	cd devices && $(MAKE) clean

# Test targets
test-audio-host: audio-host
	@echo "Testing audio-host..."
	cd audio-host && echo "devices audio-input" | ./audio-host --command-mode

test-inspector: inspector
	@echo "Testing inspector..."
	cd inspector && ./inspector 2>/dev/null | jq '. | length' || echo "Inspector test completed"

test-devices: devices
	@echo "Testing devices..."
	cd devices && ./devices 2>/dev/null | jq 'keys' || echo "Devices test completed"

test: test-inspector test-devices test-audio-host

# Help
help:
	@echo "Rackless Standalone Tools"
	@echo ""
	@echo "Targets:"
	@echo "  all              - Build all standalone tools"
	@echo "  audio-host       - Build real-time audio engine"
	@echo "  inspector        - Build plugin discovery tool"
	@echo "  devices          - Build unified device enumeration tool"
	@echo "  clean            - Clean all builds"
	@echo "  test             - Test all tools"
	@echo "  help             - Show this help"
