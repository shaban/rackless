# Rackless Architecture

## Overview

Rackless is designed as a **single-binary, dual-target** application that can run as either a native backend server or a WASM frontend, sharing the same Go codebase for consistent behavior and simplified development.

## Core Architecture Principles

### 1. Single Language Consistency
- **Backend**: Go with Objective-C bridge for AudioUnit APIs
- **Frontend**: Go WASM with minimal JavaScript glue
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
│  │  AudioUnit      │  │   Parameter     │  │   Layout     │ │
│  │  Discovery      │  │   Mapping       │  │  Management  │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                    CGO Bridge                               │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │  AudioUnit      │  │  AudioUnit      │                   │
│  │  Devices        │  │  Inspector      │                   │
│  │  (Objective-C)  │  │  (Objective-C)  │                   │
│  └─────────────────┘  └─────────────────┘                   │
├─────────────────────────────────────────────────────────────┤
│                    macOS AudioUnit APIs                     │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### AudioUnit Discovery
1. Go backend calls Objective-C bridge
2. Objective-C enumerates AudioComponents
3. Returns JSON data to Go
4. Go serves structured data to WASM frontend

### Parameter Mapping
1. User creates mapping in WASM UI
2. WASM sends mapping to backend via HTTP
3. Backend stores mapping and validates parameters
4. Real-time updates flow through WebSocket

### Real-time Control
1. MIDI controller or UI control changes value
2. WASM calculates mapped parameter values
3. Direct parameter updates via AudioUnit APIs
4. UI reflects changes immediately

## Migration Strategy

### Phase 1: Foundation ✅
- [x] Repository structure
- [x] Go modules setup
- [x] Build system (Makefile)
- [x] Basic server and WASM stubs

### Phase 2: Audio Bridge (Next)
- [ ] Extract Objective-C AudioUnit bridge
- [ ] Port Go audio device enumeration
- [ ] Port Go AudioUnit introspection
- [ ] Test bridge functionality

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

### Why Single Binary Architecture?
- **Deployment**: Single executable contains everything
- **Development**: Consistent debugging across "backend" and "frontend"
- **Testing**: Test complete application as a unit
- **Distribution**: No complex installation procedures
