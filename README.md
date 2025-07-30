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
- **Go**: Main application logic
- **Objective-C**: Native AudioUnit API bridge
- **CGO**: C bridge for Go ↔ Objective-C integration

#### Frontend (Migrating)
- **Go WASM**: Application logic in WebAssembly
- **HTML Templates**: Server-side rendering
- **Canvas API**: Custom audio controls (knobs, sliders, waveforms)
- **Web MIDI API**: Direct MIDI controller integration

## Project Structure

```
rackless/
├── README.md
├── Makefile
├── go.mod
├── go.sum
├── cmd/
│   ├── server/          # Native backend server
│   └── wasm/           # WASM frontend application
├── pkg/
│   ├── audio/          # Objective-C AudioUnit bridge
│   ├── introspection/  # AudioUnit parameter logic
│   ├── mapping/        # Parameter mapping engine
│   └── ui/            # WASM UI components
├── web/
│   ├── static/        # CSS, assets
│   └── templates/     # Go HTML templates
├── Archive/           # Previous Vue.js implementation (reference)
└── docs/
    ├── architecture.md
    ├── migration.md
    └── api.md
```

## Development

### Prerequisites

- **macOS**: Required for AudioUnit APIs
- **Go 1.21+**: For WASM support and latest features
- **Xcode Command Line Tools**: For Objective-C compilation

### Building

```bash
# Build native backend
make build

# Build WASM frontend
make wasm

# Development with hot reload
make dev
```

## Migration Status

🚧 **Currently migrating from Vue.js to Go WASM architecture**

See [`docs/migration.md`](docs/migration.md) for detailed migration plan and rationale.

### Why Go WASM?

1. **Single Language Consistency**: Go patterns throughout the stack
2. **No State Synchronization**: Backend and frontend share memory space
3. **Compile-time Safety**: Catch template errors before runtime
4. **AI-Friendly Development**: Predictable debugging, no framework mysteries
5. **Real-time Performance**: Direct parameter updates without framework overhead

## License

[Choose your license - MIT, Apache 2.0, etc.]

## Contributing

This project is in active development. Check the [issues](../../issues) for current priorities and the [migration plan](docs/migration.md) for architectural decisions.
