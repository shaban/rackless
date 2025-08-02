# Rackless Standalone Tools

This directory contains standalone command-line tools for audio plugin management and real-time audio processing.

## Tools

### `audio-host/`
Real-time audio processing engine for live performance and plugin hosting.

**Features:**
- Real-time audio I/O with low latency
- AudioUnit plugin chain processing  
- MIDI parameter automation
- Command-line interface for live control
- Device enumeration and management
- Test tone generation

**Usage:**
```bash
# Build and run
make audio-host
cd audio-host
echo "devices audio-input" | ./audio-host --command-mode
echo "tone freq 440" | ./audio-host --command-mode
```

### `inspector/`
Comprehensive AudioUnit plugin discovery tool.

**Features:**
- Complete AudioUnit plugin database generation (62+ plugins)
- Detailed parameter introspection with full metadata
- Audio device capability analysis
- JSON export for integration with other tools
- Indexed parameter value extraction

**Usage:**
```bash
# Build and run
make inspector
cd inspector
./inspector > system-analysis.json
./inspector 2>/dev/null | jq '.[] | .name' # List plugin names
```

### `devices/`
Unified audio and MIDI device enumeration tool using Archive device functions.

**Features:**
- Complete system device discovery (audio input/output, MIDI input/output)
- Detailed device capabilities (sample rates, bit depths, channel counts)
- Default device identification
- Unified JSON structure for easy integration
- Rich metadata for each device type

**Usage:**
```bash
# Build and run  
make devices
cd devices
./devices > all-devices.json
./devices 2>/dev/null | jq 'keys' # Show device categories
./devices 2>/dev/null | jq '.totalAudioInputDevices' # Device counts
```

## Quick Start

```bash
# Build all tools
make all

# Test everything works
make test

# Clean builds
make clean
```

## Architecture

These tools are designed to work together:

1. **`inspector`** - Run once to discover available plugins and their capabilities (62 plugins)
2. **`devices`** - Run to get comprehensive device information for all audio/MIDI devices  
3. **`audio-host`** - Use for real-time audio processing with discovered plugins and devices

## Integration

All tools output JSON that can be consumed by:
- Go server applications
- Web frontend interfaces  
- MIDI controller mapping software
- Automation systems

**Example Integration Workflow:**
```bash
# 1. Discover all plugins
./inspector/inspector > plugins.json

# 2. Discover all devices  
./devices/devices > devices.json

# 3. Start real-time audio with specific device
echo "devices audio-input" | ./audio-host/audio-host --command-mode
```
