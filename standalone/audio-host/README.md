# Rackless Audio Host

Production-ready Objective-C command-line tool for real-time guitar processing with AudioUnit plugin hosting.

## Build & Run

```bash
# Build
make

# Run interactively with test tone
./audio-host --sample-rate 96000

# Run with guitar input (Mooer Steep II example)
./audio-host --sample-rate 96000 --audio-input-device 145 --audio-input-channel 0

# Run in command mode for server integration
./audio-host --sample-rate 96000 --audio-input-device 145 --command-mode
```

## Command Line Options

```bash
# Required: Set sample rate (must match audio device)
./audio-host --sample-rate 96000

# Audio input configuration
./audio-host --audio-input-device 145 --audio-input-channel 0

# Buffer size (default: 256 samples)
./audio-host --buffer-size 512

# Command mode for programmatic control (stdin/stdout)
./audio-host --command-mode

# Disable test tone (default: off when input device specified)
./audio-host --no-tone

# Help
./audio-host --help
```

## Interactive Commands (Command Mode)

```bash
# Audio engine control
start                    # Start audio processing
stop                     # Stop audio processing
status                   # Get current status

# Test tone control
tone on                  # Enable test tone
tone off                 # Disable test tone
tone freq 1000           # Set frequency (Hz)

# Plugin management
load-plugin aumf:NMAS:NDSP    # Load Neural DSP plugin
unload-plugin                 # Unload current plugin
list-plugins                  # Show loaded plugins

# Device enumeration
devices audio-input      # List audio input devices (JSON)
devices audio-output     # List audio output devices (JSON)
devices midi-input       # List MIDI input devices (JSON)
devices midi-output      # List MIDI output devices (JSON)

# Exit
quit                     # Stop and exit
```

## Features

- ✅ **Real-time Guitar Processing**: Low-latency input → plugin → output
- ✅ **AudioUnit Plugin Hosting**: Load and process guitar amp/effect plugins
- ✅ **Device Selection**: Specify audio input/output devices by ID
- ✅ **Sample Rate Validation**: Automatic device compatibility checking
- ✅ **Command Mode**: stdin/stdout interface for server integration
- ✅ **Test Tone Generator**: Built-in sine wave for testing
- ✅ **JSON Device Enumeration**: Clean API for device discovery

## Critical: Device Switching Limitation

⚠️ **IMPORTANT**: Audio input/output devices **cannot be changed at runtime**. They must be specified at startup via command-line arguments.

**For server integration:**
- To switch audio devices, the server **must kill** the current audio-host process
- Then start a **new audio-host** instance with the desired device parameters
- This is a Core Audio requirement - changing devices requires reinitializing the entire audio graph

```bash
# Example: Switch from device 145 to device 105
# 1. Kill current process
pkill audio-host

# 2. Start new instance with different device
./audio-host --sample-rate 96000 --audio-input-device 105 --command-mode
```

## Audio Configuration

- **Sample Rate**: Must match audio device (typically 44.1kHz, 48kHz, or 96kHz)
- **Audio Format**: 32-bit Float, Stereo
- **Buffer Size**: 256 samples (configurable)
- **Input Processing**: Mono guitar → stereo output
- **Plugin Format**: AudioUnit (.component)

## Example Usage

### Guitar Processing with Neural DSP Plugin
```bash
# Start with guitar interface
./audio-host --sample-rate 96000 --audio-input-device 145 --command-mode

# In command mode:
start
load-plugin aumf:NMAS:NDSP
# Play guitar - processed through Neural DSP Morgan Amps Suite
unload-plugin
stop
quit
```

### Server Integration Pattern
```bash
# Server enumerates devices first
echo "devices audio-input" | ./audio-host --sample-rate 96000 --command-mode

# Server starts audio processing with selected device
./audio-host --sample-rate 96000 --audio-input-device 145 --command-mode << EOF
start
load-plugin aumf:NMAS:NDSP
EOF
```

## Requirements

- macOS with Core Audio
- Xcode command line tools (`clang`)
