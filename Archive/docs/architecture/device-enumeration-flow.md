# Device Enumeration Flow

## Overview
This document maps the complete device discovery and capability detection process in MC-SoFX-Controller.

## Current Implementation Flow

```mermaid
flowchart TD
    A[HTTP Request: /api/devices/audio/input] --> B[DeviceEnumerator.GetAudioInputDevices]
    B --> C[C.getAudioInputDevices]
    C --> D[Core Audio: Get Device Count]
    D --> E[Core Audio: Get Device IDs]
    E --> F[For Each Device]
    F --> G{Has Input Channels?}
    G -->|No| H[Skip Device]
    G -->|Yes| I[Get Capability Info]
    I --> J[Get Sample Rates]
    I --> K[Get Bit Depths]
    I --> L[Get Device Name]
    J --> M[Build Device JSON]
    K --> M
    L --> M
    M --> N[Add to Results Array]
    H --> O[Next Device]
    N --> O
    O --> P{More Devices?}
    P -->|Yes| F
    P -->|No| Q[Serialize to JSON]
    Q --> R[Return to Go]
    R --> S[HTTP Response]
```

## Key Decision Points

### 1. Device Filtering Logic
```mermaid
graph LR
    A[Device ID] --> B{Stream Config Available?}
    B -->|No| C[Skip - No Audio Capability]
    B -->|Yes| D{Input/Output Channels > 0?}
    D -->|No| E[Skip - Wrong Direction]
    D -->|Yes| F[Include Device]
```

### 2. Capability Detection Strategy
```mermaid
graph TD
    A[Device Selected] --> B[Get Sample Rate Ranges]
    B --> C{Range Data Available?}
    C -->|No| D[Use Fallback: 44.1/48kHz]
    C -->|Yes| E[Extract Common Rates]
    E --> F[Get Stream Formats]
    F --> G{Format Data Available?}
    G -->|No| H[Use Fallback: 16/24 bit]
    G -->|Yes| I[Extract Bit Depths]
    D --> J[Build Capability Object]
    H --> J
    I --> J
```

## Error Handling States

```mermaid
stateDiagram-v2
    [*] --> Scanning
    Scanning --> DeviceFound : Device detected
    Scanning --> ScanComplete : No more devices
    DeviceFound --> CapabilityCheck : Get capabilities
    CapabilityCheck --> DeviceValid : Capabilities OK
    CapabilityCheck --> DeviceSkipped : No capabilities
    DeviceValid --> DeviceAdded : Add to results
    DeviceSkipped --> NextDevice : Continue scan
    DeviceAdded --> NextDevice : Continue scan
    NextDevice --> Scanning : More devices
    NextDevice --> ScanComplete : Done
    ScanComplete --> [*]
    
    CapabilityCheck --> ErrorState : Core Audio error
    ErrorState --> DeviceSkipped : Recover and skip
```

## Current Issues & Improvements Needed

### ❌ Current Problems
1. **Default Device Detection**: Still hardcoded values
2. **Error Recovery**: Limited error handling for device failures
3. **Real-time Updates**: No device change notifications
4. **Validation**: No capability intersection checking

### ✅ Next Architecture Steps
1. **Default Device Flow**: Add real system default detection
2. **Device Compatibility Matrix**: Pre-compute valid device combinations
3. **Audio Chain Validation**: Check routing feasibility before AudioUnit load
4. **State Management**: Track device availability changes

## Testing Strategy

Each state transition should have corresponding tests:

- ✅ **Device Enumeration**: Test with known hardware configurations
- ✅ **Capability Detection**: Verify sample rates/bit depths match hardware specs  
- ⚠️ **Error Conditions**: Test device disconnection during enumeration
- ❌ **Default Device Logic**: Test system preference changes
- ❌ **Routing Validation**: Test incompatible device combinations

## Integration Points

This flow integrates with:
- **Frontend Device Selectors**: Populates dropdown lists
- **AudioUnit Host**: Provides constraints for audio processing
- **Preferences Manager**: Stores user device selections
- **Layout Manager**: Determines available I/O for plugin routing

---
*Last Updated: July 29, 2025*
*Implementation Status: Device enumeration ✅, Default detection ⚠️, Validation ❌*
