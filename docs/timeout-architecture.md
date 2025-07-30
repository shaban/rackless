# AudioUnit Introspection Timeout Architecture

## Overview

Rackless uses a **dual-timeout architecture** to ensure robust AudioUnit plugin introspection without deadlocks or indefinite hangs.

## Architecture

```
Go Application Layer
    ↕ (30s overall timeout)
Go Introspection Package (pkg/introspection)  
    ↕ (context.WithTimeout + goroutines)
CGO Bridge
    ↕ (channels for async communication)
Objective-C AudioUnit Bridge (pkg/audio)
    ↕ (0.5s per-plugin timeout)
macOS AudioUnit APIs
```

## Dual-Timeout Strategy

### Level 1: Plugin Initialization Timeout (Objective-C)

**Location**: `pkg/audio/audiounit_inspector.m`
```objectivec
#define INSPECTION_TIMEOUT_SECONDS 0.5  // Quick timeout for plugin initialization
```

**Purpose**: 
- Skip individual plugins that don't initialize quickly
- Prevent problematic plugins from blocking the entire scan
- Allow well-behaved plugins to complete normally

**Behavior**:
- Each plugin gets 0.5 seconds to initialize and expose parameters
- If a plugin exceeds this timeout, it's skipped and marked as "timed out"
- The scan continues with the next plugin
- Results include all successfully scanned plugins

### Level 2: Overall Process Timeout (Go)

**Location**: `pkg/introspection/native.go`
```go
const DefaultIntrospectionTimeout = 30 * time.Second

func GetAudioUnitsWithTimeout(timeout time.Duration) (IntrospectionResult, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    
    // Run C function in goroutine with channels for async communication
    // Return timeout error if entire process exceeds limit
}
```

**Purpose**:
- Prevent the entire introspection process from hanging indefinitely
- Provide configurable timeout for different scenarios (testing vs production)
- Clean up resources if CGO bridge hangs

**Behavior**:
- Entire introspection process must complete within 30 seconds (default)
- Uses Go context for clean cancellation
- Returns proper error if timeout exceeded
- Prevents deadlocks in testing environments

## Performance Characteristics

### Normal Operation
- **Total time**: ~6-7 seconds
- **Plugin timeout**: 0.5s is sufficient for well-behaved plugins
- **Overall timeout**: 30s provides generous buffer

### Testing Environment
- **Benchmark stress**: Can cause plugins to take longer
- **Level 1**: Individual plugins still timeout at 0.5s (prevents hangs)
- **Level 2**: Overall process completes within 30s (prevents test deadlocks)

### Error Scenarios
- **Broken plugin**: Skipped after 0.5s, scan continues
- **System overload**: Process times out cleanly after 30s
- **CGO bridge hang**: Go timeout prevents deadlock

## Configuration

### Compile-time (Objective-C)
```c
// Override at compile time if needed
#define INSPECTION_TIMEOUT_SECONDS 0.5
```

### Runtime (Go)
```go
// Use default timeout
plugins, err := introspection.GetAudioUnits()

// Use custom timeout
plugins, err := introspection.GetAudioUnitsWithTimeout(60 * time.Second)
```

## Results Consistency

Despite timeouts, the system produces **consistent results**:
- **62 plugins** with parameters discovered
- **1,759 total parameters** across all plugins
- **627.6 KB JSON** output
- **Neural DSP Morgan Amps Suite** identified as best demo plugin (128 parameters)

## Why This Works

1. **Fast Path**: Well-behaved plugins complete in < 0.5s
2. **Robust Handling**: Problematic plugins are skipped, not failed
3. **Resource Safety**: Go context ensures cleanup
4. **Testing Safe**: Benchmarks complete within reasonable time
5. **Production Ready**: Consistent performance and results

## Implementation Notes

- **CGO Safety**: All C memory properly freed with `defer C.free()`
- **Goroutine Communication**: Channels used for async C function calls
- **Error Propagation**: Both timeout levels return appropriate errors
- **Context Handling**: Proper cancellation prevents resource leaks

## Historical Context

- **Original Issue**: Some plugins would hang indefinitely during initialization
- **First Attempt**: Single 60s timeout - too slow for testing
- **Second Attempt**: Single 0.5s timeout - worked but no deadlock protection
- **Current Solution**: Dual timeouts provide both speed and safety

This architecture ensures Rackless is both **performant for users** and **reliable for development**.
