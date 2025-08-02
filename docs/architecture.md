# Rackless Architecture

## Overview

Rackless is designed as a **single-binary, dual-target** application that can run as either a native backend server or a WASM frontend, sharing the same Go codebase for consistent behavior and simplified development.

## Core Architecture Principles

### 1. Single Language Consistency
- **Backend**: Go with JSON communication to standalone Objective-C tools
- **Frontend**: Go WASM with minimal JavaScript glue
- **AudioUnit Integration**: Standalone command-line tools (no CGO bridge)
- **Shared Types**: Same structs for data, API, and templates

### 2. Direct State Management
- No JSON serialization between backend/frontend
- WASM and native backend share identical data structures
- Real-time parameter updates without framework overhead

### 3. Custom UI Controls
- Canvas-based knobs, sliders, and waveform displays
- Direct DOM manipulation for performance
- Web MIDI API integration for hardware controllers

## Component Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Browser                               │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │   Go WASM       │  │   Canvas UI     │  │  Web MIDI    │ │
│  │   Application   │  │   Controls      │  │   API        │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                    HTTP/WebSocket                           │
├─────────────────────────────────────────────────────────────┤
│                     Go Backend Server                       │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │  JSON Commands  │  │   Parameter     │  │   Layout     │ │
│  │  via exec.Cmd   │  │   Mapping       │  │  Management  │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│              Command-Line JSON Interface                    │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │  audio-host     │  │   inspector     │  │   devices    │ │
│  │  (Real-time     │  │   (Plugin       │  │  (Device     │ │
│  │   Processing)   │  │   Discovery)    │  │   Enum)      │ │
│  │  (Objective-C)  │  │  (Objective-C)  │  │ (Objective-C)│ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                    macOS AudioUnit APIs                     │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### AudioUnit Discovery
1. Go backend calls standalone command-line tools via exec.Command()
2. Objective-C tools enumerate AudioComponents and return JSON to stdout
3. Go parses JSON and serves structured data to WASM frontend

### Parameter Mapping
1. User creates mapping in WASM UI
2. WASM sends mapping to backend via HTTP
3. Backend stores mapping and validates parameters
4. Real-time updates flow through WebSocket

**Note**: See [Audio Validation Reality](audio-validation-reality.md) for current validation behavior vs audio-host capabilities.

### Real-time Control
1. MIDI controller or UI control changes value
2. WASM calculates mapped parameter values  
3. Backend sends commands to audio-host via stdin/stdout pipes
4. UI reflects changes immediately

## Migration Strategy

### Phase 1: Foundation ✅
- [x] Repository structure
- [x] Go modules setup
- [x] Build system (Makefile)
- [x] Basic server and WASM stubs

### Phase 2: Audio Integration (Current)
- [x] Extract standalone Objective-C command-line tools
- [x] Port Go audio device enumeration via JSON interface
- [x] Port Go AudioUnit introspection via JSON interface
- [x] Bidirectional command-line audio-host for real-time processing

### Phase 3: WASM Frontend
- [ ] Basic WASM application structure
- [ ] Canvas-based UI controls
- [ ] Parameter mapping interface
- [ ] Real-time parameter updates

### Phase 4: Advanced Features
- [ ] Layout management system
- [ ] Curve editing for parameter mapping
- [ ] MIDI controller integration
- [ ] Preset system

## Key Design Decisions

### Why Go WASM over Vue.js?
- **State Synchronization**: Eliminated Vue reactivity debugging
- **Type Safety**: Compile-time checking for UI logic
- **Performance**: Direct memory access, no framework overhead
- **Consistency**: Single language reduces context switching

### Why Custom Canvas Controls?
- **Audio-Specific**: Knobs, waveforms need custom rendering
- **Performance**: Direct GPU acceleration via Canvas
- **Flexibility**: Complete control over appearance and behavior
- **Responsiveness**: Sub-millisecond parameter updates

### Why Standalone Tools over CGO Bridge?
- **Process Isolation**: Audio crashes don't crash Go server
- **Development Simplicity**: No CGO complexity, cross-compilation issues
- **Tool Reusability**: Command-line tools useful for debugging/testing  
- **JSON Interface**: Clean, typed communication between Go and Objective-C
- **Real-time Performance**: Direct AudioUnit access in dedicated process
