# AudioUnit Introspection Timeout Architecture

## Overview

Rackless uses a **single-layer timeout architecture** to ensure robust AudioUnit plugin introspection without deadlocks or indefinite hangs. After analysis, we determined that the Go-level timeout was superfluous, and the Objective-C plugin-level timeout provides sufficient protection.

## Simplified Architecture

```
Go Application Layer
    ↕ (direct synchronous call)
Go Introspection Package (pkg/introspection)  
    ↕ (simple CGO bridge)
Objective-C AudioUnit Bridge (pkg/audio)
    ↕ (0.5s per-plugin timeout)
macOS AudioUnit APIs
```

## Single-Timeout Strategy

### Plugin Initialization Timeout (Objective-C)

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

**Why This Is Sufficient**:
- **Real-world data**: 62 plugins complete in ~6-7 seconds total
- **Plugin filtering**: Only plugins with parameters are processed
- **0.5s per plugin**: Even 200+ plugins would complete in reasonable time
- **Theoretical max**: With heavy plugin loads, still completes well under any reasonable timeout
- **Future caching**: Will make subsequent scans nearly instantaneous

## Performance Characteristics

### Normal Operation
- **Total time**: ~6-7 seconds for 62 plugins
- **Plugin timeout**: 0.5s is sufficient for well-behaved plugins
- **No overhead**: No goroutines, channels, or context management

### Extreme Scenarios
- **200+ plugins**: ~100s maximum (200 × 0.5s), but realistic filtering reduces this significantly
- **Many quick plugins**: Complete in milliseconds each
- **Mixed load**: Fast plugins don't wait for slow plugins

### Error Scenarios
- **Broken plugin**: Skipped after 0.5s, scan continues
- **System overload**: Each plugin still gets 0.5s maximum
- **Multiple problematic plugins**: Each handled individually, no cascade failure

## Removed Complexity

### What We Eliminated
- ❌ Go context timeouts
- ❌ Goroutines for async execution  
- ❌ Channel-based communication
- ❌ Complex timeout error handling
- ❌ 30-second overall timeout constant

### Benefits of Simplification
- ✅ **Cleaner code**: Direct synchronous calls
- ✅ **Better performance**: No goroutine overhead
- ✅ **Simpler debugging**: Straightforward call stack
- ✅ **More predictable**: No context cancellation edge cases
- ✅ **Easier maintenance**: Less complex timeout logic

## Configuration

### Compile-time (Objective-C)
```c
// Override at compile time if needed
#define INSPECTION_TIMEOUT_SECONDS 0.5
```

### Runtime (Go)
```go
// Simple synchronous call - no timeout configuration needed
plugins, err := introspection.GetAudioUnits()

// Or get raw JSON
jsonData, err := introspection.GetAudioUnitsJSON()
```

## Current Results

### System Performance
- **62 plugins discovered** in ~6-7 seconds
- **1,759 total parameters** across all plugins
- **627.6 KB JSON** output
- **Neural DSP Morgan Amps Suite** identified as best demo plugin (128 parameters)

## Why This Single-Timeout Approach Works

1. **Fast Path**: Well-behaved plugins complete in milliseconds
2. **Robust Handling**: Problematic plugins are skipped after 0.5s, not failed
3. **Predictable Performance**: Total time is reasonably bounded by plugin count
4. **Simple Implementation**: Direct synchronous calls, easy to debug
5. **Production Ready**: Consistent performance and results

## Implementation Notes

- **CGO Safety**: All C memory properly freed with `defer C.free()`
- **Direct Calls**: No goroutines or channels needed
- **Error Propagation**: Clean error handling from Objective-C layer
- **Memory Management**: Proper cleanup prevents leaks

## Historical Context

- **Original Issue**: Some plugins would hang indefinitely during initialization
- **First Solution**: 0.5s plugin timeout in Objective-C - **essential protection**
- **Added Complexity**: 30s Go timeout with goroutines - **analysis showed superfluous**
- **Current Solution**: Single plugin timeout provides both speed and safety

**Key Insight**: The 0.5s plugin timeout is the **critical protection**. Go-level timeouts added complexity without meaningful benefit since:
- Real-world plugin loads complete well within any reasonable timeout
- Plugin filtering reduces actual processing time
- Individual plugin timeouts prevent cascade failures
- Future caching will make this even faster

This simplified architecture ensures Rackless is both **performant for users** and **maintainable for developers**.

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
