# Rackless Next Steps: Unified Audio-Host Architecture

## 🎯 **Core Problem Analysis**

### **Current Architecture Flaws**
The current design splits audio functionality across 3 separate, disposable tools:
```
Go Server
├── exec standalone/devices/devices → JSON output, exits
├── exec standalone/inspector/inspector → JSON output, exits  
├── exec standalone/audio-host/audio-host → long-running, but limited capabilities
```

**Critical Issues Identified:**
1. **Process Coordination Complexity** - Server manages 3 different tools with different interfaces
2. **No Dynamic Reconfiguration** - Can't change devices/sample rates without full restart
3. **Information Silos** - Each tool knows different aspects of the audio system
4. **UX Confusion** - Command-line args vs command mode parameters are inconsistent
5. **Missing 1:1 Feature Parity** - Core audio config only available at startup, runtime features only available via commands

### **User Experience Problems**
- Device switching requires killing and restarting processes
- Plugin discovery is separate from audio processing
- Can't pre-load plugins on startup
- Manual engine start required even with perfect configuration
- No runtime reconfiguration of core audio parameters

## 🚀 **Proposed Solution: Unified Audio-Host**

### **Target Architecture**
```
Go Server
└── unified-audio-host (single long-running process)
    ├── Device Discovery & Enumeration
    ├── Plugin Discovery & Introspection  
    ├── Real-time Audio Processing
    ├── Dynamic Chain Reconfiguration
    └── Unified Command Interface
```

### **Key Benefits**
1. **Single Audio Process** - One intelligent process managing all audio functionality
2. **Self-Contained** - Can discover devices, inspect plugins, and process audio
3. **Dynamic Reconfiguration** - Rebuild audio chain without restart
4. **Stateful Intelligence** - Remembers configurations, can optimize transitions
5. **Unified API** - Consistent interface for all audio operations
6. **Better Error Handling** - Centralized error reporting and recovery

## 📋 **Implementation Roadmap**

### **Phase 1: Device Discovery Integration (1-2 weeks)**
**Goal**: Add device enumeration capabilities to audio-host

**Tasks:**
1. **Copy Device Enumeration Code**
   - Move `standalone/devices/audiounit_devices.m` functions into `audio-host/main.m`
   - Add command-line mode: `./audio-host devices`
   - Maintain JSON output compatibility

2. **Add Command Mode Support**
   ```objc
   // New command mode commands:
   "devices audio-input"   → enumerate audio input devices
   "devices audio-output"  → enumerate audio output devices  
   "devices midi-input"    → enumerate MIDI input devices
   "devices midi-output"   → enumerate MIDI output devices
   ```

3. **Test Compatibility**
   - Ensure Go server can still call device enumeration
   - Verify JSON output format matches existing expectations
   - Test both command-line and command mode interfaces

**Success Criteria:**
- `./audio-host devices` produces same JSON as current `./devices/devices`
- Command mode `devices audio-input` works while audio is running
- Go server integration works without changes

### **Phase 2: Plugin Discovery Integration (1 week)**
**Goal**: Add plugin introspection capabilities to audio-host

**Tasks:**
1. **Copy Plugin Introspection Code**
   - Move `standalone/inspector/audiounit_inspector.m` functions into `audio-host/main.m`
   - Add command-line mode: `./audio-host inspect [timeout]`
   - Maintain JSON output compatibility and timeout parameters

2. **Add Command Mode Support**
   ```objc
   // New command mode commands:
   "inspect [timeout]"     → full plugin scan with optional timeout
   "inspect-plugin <id>"   → inspect specific plugin by ID
   "find-plugin <name>"    → search plugins by name/manufacturer
   ```

3. **Runtime Plugin Discovery**
   - Allow plugin scanning while audio engine is running
   - Cache plugin information for faster lookups
   - Support incremental plugin discovery

**Success Criteria:**
- `./audio-host inspect` produces same JSON as current `./inspector/inspector`
- Plugin discovery works during audio processing
- Timeout handling maintained for problematic plugins

### **Phase 3: Runtime Reconfiguration (2-3 weeks)**
**Goal**: Enable dynamic audio chain reconfiguration without restart

**Tasks:**
1. **Dynamic Device Switching**
   ```objc
   // New runtime commands:
   "set-device input <id>"    → hot-swap input device
   "set-device output <id>"   → hot-swap output device
   "set-channel <n>"          → change input channel
   ```

2. **Dynamic Audio Parameters**
   ```objc
   "set-sample-rate <hz>"     → change sample rate (rebuilds chain)
   "set-buffer-size <n>"      → change buffer size (rebuilds chain)
   "rebuild-chain"            → manual chain rebuild with current config
   ```

3. **Smart Chain Management**
   - Graceful audio chain teardown and rebuild
   - Preserve plugin state during reconfiguration where possible
   - Atomic configuration changes (all-or-nothing)
   - Rollback capability if new configuration fails

4. **Enhanced Plugin Management**
   ```objc
   "switch-plugin <id>"       → unload current, load new plugin
   "reload-plugin"            → reload current plugin (reset state)
   "plugin-preset <name>"     → load plugin preset by name
   ```

**Success Criteria:**
- Device switching works without audio dropouts
- Sample rate changes rebuild chain successfully
- Plugin switching preserves audio processing state
- Failed reconfigurations rollback gracefully

### **Phase 4: Server Integration & Cleanup (1 week)**
**Goal**: Update Go server to use unified process, remove legacy tools

**Tasks:**
1. **Update Go Audio Package**
   - Modify `audio/devices.go` to call unified audio-host
   - Update `audio/process.go` for enhanced command interface
   - Remove separate device/inspector tool calls

2. **Enhanced Server APIs**
   ```go
   // New API capabilities:
   POST /api/audio/set-device        → runtime device switching
   POST /api/audio/rebuild-chain     → force chain reconfiguration  
   GET  /api/audio/discover-devices  → live device discovery
   GET  /api/audio/scan-plugins      → live plugin scanning
   ```

3. **Startup Optimization**
   ```bash
   # New startup options:
   --auto-start              → automatically start engine after init
   --load-plugin <id>        → pre-load plugin on startup
   --device input <id>       → set input device
   --device output <id>      → set output device
   ```

4. **Legacy Tool Removal**
   - Archive `standalone/devices/` and `standalone/inspector/`
   - Update build system to only build unified audio-host
   - Update documentation to reflect new architecture

**Success Criteria:**
- Go server works with unified audio-host
- All existing API endpoints continue to work
- New runtime reconfiguration APIs functional
- Legacy tools no longer needed

## 🎯 **Immediate Next Session Goals**

### **Priority 1: Start Phase 1 Implementation**
1. **Copy device enumeration functions** from `standalone/devices/audiounit_devices.m` into `standalone/audio-host/main.m`
2. **Add command-line device mode** to audio-host
3. **Test basic integration** with existing Go server

### **Priority 2: Design Command Interface**
1. **Unify command-line args and command mode parameters**
2. **Design consistent command syntax** for all operations
3. **Plan backward compatibility** during transition

### **Priority 3: Architecture Validation**
1. **Validate that audio chain rebuilding is feasible** without glitches
2. **Test hot device switching** with real hardware
3. **Measure performance impact** of unified process

## 🔄 **Backward Compatibility Strategy**

### **During Transition (Phases 1-3)**
- Keep existing standalone tools working
- Add feature flags to enable/disable unified functionality
- Maintain JSON output format compatibility
- Allow gradual migration of Go server functionality

### **After Migration (Phase 4+)**
- Archive old tools but keep them in git history
- Update documentation with migration guide
- Provide clear upgrade path for any external integrations

## 🧪 **Testing Strategy**

### **Integration Testing**
- Test with real audio hardware (Focusrite, Native Instruments, etc.)
- Validate with multiple plugin formats (Neural DSP, Logic, third-party)
- Test device switching scenarios (USB connect/disconnect)

### **Performance Testing**
- Measure latency impact of unified process
- Test memory usage with large plugin collections
- Validate real-time performance during reconfiguration

### **User Experience Testing**
- Test common workflows (device switching, plugin loading)
- Validate error messages and recovery scenarios
- Ensure intuitive command interface

## 💡 **Implementation Notes**

### **Code Organization**
- Create separate `.m` files for each major function area
- Maintain clear separation between audio engine and discovery functions
- Use consistent error handling patterns across all functionality

### **Configuration Management**
- Consider adding configuration file support for common setups
- Cache discovered devices and plugins for faster startup
- Support for audio hardware profiles (studio, live, headphones, etc.)

### **Future Extensibility**
- Design plugin API for future custom plugins
- Plan for MIDI controller integration
- Consider automation and scripting capabilities

## 🎸 **Why This Matters**

This architectural change will transform Rackless from a proof-of-concept into a professional audio tool capable of:
- **Live Performance** - Real-time device switching for stage use
- **Studio Work** - Dynamic configuration for different recording setups  
- **Sound Design** - Seamless plugin experimentation and A/B testing
- **Education** - Clear, consistent interface for learning audio programming

The wide-reaching changes are not just necessary—they're the foundation for everything we want to build next.

---

## 📡 **Addendum: Go Server as Thin Proxy**

### **Audio-Host as Single Authority**
With the unified audio-host becoming the authoritative source for all audio operations, the Go server should evolve into a **thin proxy layer** that:

1. **Forwards Frontend Commands** - Pass all audio-related requests directly to audio-host
2. **Preserves Error Context** - Return audio-host's exact error codes and messages to frontend
3. **Maintains Single Source of Truth** - Audio-host retains complete authority over audio state
4. **Eliminates Dual Logic** - Remove audio logic duplication between Go server and audio tools

### **Benefits of Proxy Architecture**
- **Consistent Error Handling** - Frontend gets authoritative error responses from actual audio system
- **Reduced Complexity** - Go server focuses on HTTP/WebSocket handling, not audio logic
- **Better Debugging** - All audio errors originate from single source
- **Future-Proof** - Audio-host can evolve independently while maintaining API contract

### **Implementation Strategy**
```go
// Go server becomes thin proxy:
func (h *AudioHandler) SetDevice(w http.ResponseWriter, r *http.Request) {
    // Parse request
    // Send command to audio-host
    // Return audio-host's response directly (status code + body)
    // No interpretation or transformation of audio-host responses
}
```

This ensures audio-host maintains its authority as the single point of truth across all contexts - command-line, API, and future integrations.

---

**Next Session**: Let's dive into Phase 1 and start integrating device discovery into audio-host! 🚀
