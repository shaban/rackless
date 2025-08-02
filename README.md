# Rackless

**A streamlined, custom audio plugin parameter automation system without the traditional rack constraints.**

## Overview

Rackless is an audio plugin automation system that allows you to map UI controls to multiple plugin parameters across different plugins in your audio chain. Think of it as a custom control surface that can automate any AudioUnit parameter in real-time.

### Core Features

- **AudioUnit Discovery**: Enumerate all installed AudioUnit plugins on macOS
- **Parameter Introspection**: Extract parameter lists, ranges, and types from any AudioUnit
- **Custom Control Mapping**: Map sliders, knobs, and buttons to multiple plugin parameters
- **Real-time Updates**: Asynchronous parameter updates during audio playback
- **Curve Mapping**: Transform control ranges with linear, exponential, or custom curves
- **Threshold Controls**: Special mappings for on/off effects based on control values

## Architecture

### Current Status: Migration in Progress

**From**: Vue.js frontend with multiple UI frameworks (Tailwind, Shoelace, webaudio-controls)
**To**: Go WASM + Templates for simplified state management and better AI development experience

### Tech Stack

#### Backend (Working)
- **Go**: Main application server and audio configuration logic
- **Standalone Tools**: Independent Objective-C command-line utilities

#### Frontend (Migrating)
- **Go WASM**: Application logic in WebAssembly
- **HTML Templates**: Server-side rendering
- **Canvas API**: Custom audio controls (knobs, sliders, waveforms)
- **Web MIDI API**: Direct MIDI controller integration

#### Standalone Tools (Production Ready)
- **Objective-C**: Native AudioUnit API for real-time audio processing
- **Inspector**: AudioUnit plugin discovery and parameter extraction (JSON output)
- **Audio Host**: Real-time guitar processing with interactive command-line interface
- **Devices**: Audio/MIDI device enumeration (JSON output)

## Project Structure

```
rackless/
├── README.md
├── Makefile               # Main build system
├── go.mod
├── *.go                   # Go server application
├── data/
│   └── settings.json     # Configuration
├── docs/
│   ├── architecture.md
│   └── *.md             # Technical documentation
├── frontend/            # Go WASM frontend (migrating)
│   ├── Makefile
│   ├── components/
│   ├── src/
│   └── static/
└── standalone/          # Production-ready Objective-C tools
    ├── audio-host/      # Real-time guitar processing (interactive CLI)
    ├── inspector/       # AudioUnit plugin discovery
    └── devices/         # Audio/MIDI device enumeration
```

## Development

### Prerequisites

- **macOS**: Required for AudioUnit APIs
- **Go 1.21+**: For WASM support and latest features
- **Xcode Command Line Tools**: For Objective-C compilation

### Building

```bash
# Build main Go server
make server              # Build and run server
make server-dev          # Development mode with auto-reload

# Build frontend (WASM migration in progress)
make frontend

# Build standalone tools
make standalone          # All tools
cd standalone/audio-host && make    # Just audio-host
cd standalone/inspector && make     # Just inspector  
cd standalone/devices && make       # Just devices

# Development workflow
make server-dev          # Start server in development
make css-watch          # Watch CSS changes
```

### Interactive Tools

**Audio Host** (`standalone/audio-host/`): Bidirectional interactive command-line interface
- Real-time guitar processing with AudioUnit plugins
- Commands: `start`, `stop`, `load-plugin`, `tone on/off`, `devices audio-input`
- Modes: Interactive (default) or command mode (`--command-mode` for stdin/stdout)

**Other Tools**: JSON output to stdout
- **Inspector**: AudioUnit plugin discovery and parameter extraction
- **Devices**: Audio/MIDI device enumeration

### Known Issues & Current Reality

⚠️ **Validation Gap**: Server validation is currently more restrictive than audio-host capabilities
- Server rejects some valid configurations that audio-host accepts
- See [`docs/audio-validation-reality.md`](docs/audio-validation-reality.md) for details
- Affects buffer sizes and sample rates in particular

## Migration Status

🚧 **Currently migrating from Vue.js to Go WASM architecture**

See [`docs/architecture.md`](docs/architecture.md) for current architecture and design decisions.

### Why Go WASM?

1. **Single Language Consistency**: Go patterns throughout the stack
2. **Shared Data Structures**: Same Go types for frontend/backend communication
3. **Compile-time Safety**: Catch template errors before runtime
4. **AI-Friendly Development**: Predictable debugging, no framework mysteries
5. **Real-time Performance**: Direct parameter updates without framework overhead

## License

GNU Affero General Public License v3.0 - see [LICENSE](LICENSE) file for details.

## Contributing

This project is in active development. Check the [issues](../../issues) for current priorities and [architecture documentation](docs/architecture.md) for design decisions.
