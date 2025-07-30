# Device Enumeration Architecture

This document explains the device enumeration system architecture for the rackless audio plugin host, following the same dual-timeout pattern established for AudioUnit introspection.

## Overview

The device enumeration system provides fast, reliable discovery of:
- **Audio Input Devices**: Microphones, line inputs, virtual audio devices
- **Audio Output Devices**: Speakers, headphones, virtual audio destinations  
- **MIDI Input Devices**: MIDI controllers, virtual MIDI sources
- **MIDI Output Devices**: MIDI synthesizers, virtual MIDI destinations
- **Default Devices**: System-assigned default input/output devices

## Architecture Components

### 1. Objective-C Bridge (`pkg/audio/audiounit_devices.*`)

**Purpose**: Native macOS Core Audio and Core MIDI API access
**Frameworks**: `CoreAudio`, `CoreMIDI`, `Foundation`
**Performance**: ~14ms for complete enumeration

```c
// Main enumeration functions
char* getAudioInputDevices(void);
char* getAudioOutputDevices(void);
char* getMIDIInputDevices(void);
char* getMIDIOutputDevices(void);
char* getDefaultAudioDevices(void);
```

**Key Features**:
- JSON serialization of device properties
- Sample rate and bit depth detection
- Online/offline status for MIDI devices
- Channel count validation
- Comprehensive error handling

### 2. Go Interface (`pkg/devices/`)

**Purpose**: Type-safe Go interface with timeout protection
**Pattern**: Same dual-timeout architecture as AudioUnit introspection

```go
type DeviceEnumerator interface {
    GetAudioInputDevices() ([]AudioDevice, error)
    GetAudioOutputDevices() ([]AudioDevice, error)
    GetMIDIInputDevices() ([]MIDIDevice, error)
    GetMIDIOutputDevices() ([]MIDIDevice, error)
    GetDefaultAudioDevices() (DefaultAudioDevices, error)
    GetAllDevices() (DeviceEnumerationResult, error)
}
```

### 3. Data Types

**AudioDevice**:
```go
type AudioDevice struct {
    Name                 string    `json:"name"`
    UID                  string    `json:"uid"`
    DeviceID             int       `json:"deviceId"`
    ChannelCount         int       `json:"channelCount"`
    SupportedSampleRates []float64 `json:"supportedSampleRates"`
    SupportedBitDepths   []int     `json:"supportedBitDepths"`
    IsDefault            bool      `json:"isDefault"`
}
```

**MIDIDevice**:
```go
type MIDIDevice struct {
    Name       string `json:"name"`
    UID        string `json:"uid"`
    EndpointID int    `json:"endpointId"`
    IsOnline   bool   `json:"isOnline"`
}
```

## Timeout Architecture

### Level 1: Go Context Timeout (Default: 30s)
- **Purpose**: Prevent CGO deadlocks and infinite hangs
- **Implementation**: `context.WithTimeout()` wrapping all CGO calls
- **Behavior**: Cancels enumeration if any component takes too long

### Level 2: Individual Component Timeouts
- **Audio Inputs**: ~5ms typical
- **Audio Outputs**: ~4ms typical  
- **MIDI Inputs**: ~2ms typical
- **MIDI Outputs**: ~1.6ms typical
- **Default Devices**: ~0.13ms typical

### Configuration

```go
type DeviceEnumerationConfig struct {
    Timeout              time.Duration // Default: 30s
    IncludeOfflineDevices bool          // Default: false
    IncludeVirtualDevices bool          // Default: true
}
```

## Performance Characteristics

**Benchmarks** (Apple M1, typical system):
- Complete enumeration: ~14ms
- Individual components: 0.13ms - 5.4ms
- JSON serialization: ~8.6KB output
- Memory usage: Minimal (all JSON strings freed)

**Comparison to AudioUnit Introspection**:
- Device enumeration: ~14ms
- Plugin introspection: ~6-7s
- **~420x faster** (much simpler API calls)

## Error Handling

### Graceful Degradation
- Missing devices return empty arrays, not errors
- Offline devices properly marked as `IsOnline: false`
- Invalid device IDs handled gracefully
- JSON parsing errors return empty results

### Safety Features
- "(None Selected)" options automatically added
- Device ID validation prevents negative values
- Channel count validation prevents invalid configurations
- Memory management with `defer C.free()`

## Cross-Platform Support

### macOS (CGO Build)
- Full native Core Audio/Core MIDI integration
- All device types supported
- Real-time device property detection

### Other Platforms (Stub Build)
- Mock devices for testing/development
- Consistent API interface
- Graceful fallback behavior

## Integration Points

### Build System Integration
```makefile
# Compile device enumeration bridge
clang -c -x objective-c -o pkg/audio/audiounit_devices.o pkg/audio/audiounit_devices.m \
    -framework Foundation -framework CoreAudio -framework CoreMIDI
ar rcs pkg/audio/libaudiounit_devices.a pkg/audio/audiounit_devices.o
```

### Testing Integration
- Unit tests: Basic functionality validation
- Integration tests: Real device discovery
- Benchmarks: Performance characterization
- Timeout tests: Edge case handling

### CGO Linking
```go
/*
#cgo CFLAGS: -I../audio
#cgo LDFLAGS: -L../audio -laudiounit_devices -framework CoreAudio -framework CoreMIDI -framework Foundation
*/
```

## Usage Examples

### Basic Device Discovery
```go
enumerator := devices.NewDeviceEnumerator()
result, err := enumerator.GetAllDevices()
if err != nil {
    log.Fatalf("Device enumeration failed: %v", err)
}

fmt.Printf("Found %d audio inputs, %d audio outputs\n", 
    len(result.AudioInputs), len(result.AudioOutputs))
```

### Custom Configuration
```go
config := devices.DeviceEnumerationConfig{
    Timeout:              5 * time.Second,
    IncludeOfflineDevices: false,
    IncludeVirtualDevices: true,
}
enumerator := devices.NewDeviceEnumeratorWithConfig(config)
```

## Future Enhancements

### Planned Features
- Device change notifications
- Real-time device hot-plugging detection
- Advanced device capability detection
- Device preference persistence

### Performance Optimizations
- Caching for rapid repeated calls
- Incremental updates vs full enumeration
- Background device monitoring

## Troubleshooting

### Common Issues
1. **Timeout errors**: Reduce timeout or check system load
2. **Empty device lists**: Check system permissions
3. **Invalid device IDs**: Devices may have been disconnected

### Debug Output
Enable verbose logging with NSLog output to see detailed enumeration steps:
```
🎤 Enumerating audio input devices...
✅ Found 9 audio input devices
🔊 Enumerating audio output devices...
✅ Found 10 audio output devices
```

## Architecture Notes

This device enumeration system follows the same proven patterns as the AudioUnit introspection system:

1. **Objective-C bridge**: Direct native API access
2. **Go timeout wrapper**: Process-level safety
3. **Cross-platform stubs**: Development/testing support
4. **Comprehensive testing**: Unit, integration, benchmarks
5. **JSON serialization**: Consistent data exchange

The key difference is performance: device enumeration is ~420x faster than plugin introspection because it doesn't need to instantiate and analyze audio plugins, just query system device lists.
