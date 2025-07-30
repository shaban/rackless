# MC-SoFX Controller - Architecture Migration Plan

**Date**: July 30, 2025  
**Context**: Migration from Vue.js frontend to Go WASM + Templates architecture  
**Reason**: Simplify state management, improve AI development experience, leverage Go expertise

---

## Migration Rationale

### Current Vue.js Pain Points

The current Vue.js architecture creates several development friction points:

1. **State Synchronization Hell**: 
   ```javascript
   // This debugging nightmare we just experienced
   console.log('Main app: currentLayout state:', this.currentLayout);
   console.log('Groups after push:', this.currentLayout.groups.length);
   // But Vue doesn't see the change...
   ```

2. **Template Logic Mysteries**:
   ```html
   <!-- Why isn't this working? -->
   <div v-if="currentLayout && currentLayout.groups && currentLayout.groups.length > 0">
   ```

3. **Component Communication Complexity**:
   ```javascript
   // Event chain hell
   EditorToolbar -> emit('add-group') -> App -> addNewGroup() -> PropertyInspector update
   ```

4. **Reactivity Gotchas**: Vue Proxy objects, force updates, template recompilation errors

### Proposed Go WASM + Templates Solution

#### **For AI Development: Go WASM + Templates WINS**

**Reasoning advantages AI would gain:**
1. **No Mental Model Translation**: Backend structs = template logic
2. **Predictable Debugging**: Go debugging tools, not Vue DevTools mysteries  
3. **Compile-time Safety**: Catch template errors before runtime
4. **Single Language Consistency**: Go patterns throughout
5. **No Framework Wrestling**: No Vue reactivity, component lifecycle, etc.

#### **Development Velocity Impact Comparison**

**Current Vue Approach:**
```
User: "Groups not rendering in editor"
AI: "Let me debug Vue reactivity, check computed properties, investigate template conditions, add force updates..."
*3 hours of debugging state synchronization*
```

**Go WASM Approach:**
```
User: "Groups not rendering in editor"  
AI: "Let me check the template condition and struct values"
*30 minutes - either template logic is wrong or struct isn't populated*
```

---

## Proposed Architecture

### **Go Templates + WASM Advantages**

#### ✅ **Much Better for AI Development**
```go
// Backend struct
type LayoutEditor struct {
    CurrentLayout *Layout
    SelectedItem  *SelectedItem
    HasUnsavedChanges bool
}

// Frontend mirrors exactly - no translation layer
{{.CurrentLayout.Name}}
{{range .CurrentLayout.Groups}}
    <div data-group-id="{{.ID}}">{{.Name}}</div>
{{end}}
```

**Why this is superior for AI:**
1. **Single Source of Truth**: Same structs, same logic, same validation
2. **Compile-time Safety**: Template compilation catches errors AI might miss
3. **No State Synchronization**: Backend and frontend share the same memory space
4. **Predictable Flow**: Template logic is deterministic, no Vue reactivity mysteries
5. **Easier Debugging**: Console shows Go types directly, not proxy objects

#### ✅ **Perfect Match for Existing Architecture**
```go
// Existing pattern would extend naturally
type AudioDevice struct {
    Name         string `json:"name"`
    DeviceID     string `json:"deviceId"`
    IsDefault    bool   `json:"isDefault"`
}

// Template would use the SAME struct
{{range .AudioDevices}}
    <option value="{{.DeviceID}}" {{if .IsDefault}}selected{{end}}>
        {{.Name}}
    </option>
{{end}}
```

### **Clean Architecture Implementation**

#### **Backend State Management**
```go
// main.go - Single binary handles everything
func (app *App) AddGroup(groupName string) {
    group := &Group{
        ID: generateID(),
        Name: groupName,
        // ... rest of struct
    }
    app.CurrentLayout.Groups = append(app.CurrentLayout.Groups, group)
    app.Render() // Re-render templates
}

// Exposed to JS
//go:export addGroup
func addGroup(groupName string) {
    app.AddGroup(groupName)
}
```

#### **JavaScript Helpers (Minimal)**
```javascript
// Just event binding and DOM manipulation
document.getElementById('add-group-btn').addEventListener('click', () => {
    const name = prompt('Group name:');
    if (name) {
        addGroup(name); // Calls Go function directly
    }
});
```

#### **Templates (Type-safe)**
```html
{{define "layout-editor"}}
<div class="layout-editor">
    {{if .CurrentLayout}}
        {{range .CurrentLayout.Groups}}
            <div class="group" data-id="{{.ID}}" onclick="selectGroup('{{.ID}}')">
                {{.Name}}
            </div>
        {{end}}
    {{else}}
        <p>Loading layout...</p>
    {{end}}
</div>
{{end}}
```

---

## Single Endpoint Hierarchical API Design

### **Current Multi-Endpoint Complexity**
```go
// What we have now - endpoint explosion
GET    /api/devices/audio/input
GET    /api/devices/audio/output  
GET    /api/devices/midi/input
GET    /api/devices/midi/output
GET    /api/layouts
GET    /api/layouts/{name}
POST   /api/layouts/save
PUT    /api/layouts/{name}
GET    /api/settings
POST   /api/settings
// ... and growing
```

### **Proposed Single-Endpoint Design**

#### **One Hierarchical Data Structure**
```go
type AppState struct {
    Audio struct {
        Devices struct {
            Input  []AudioDevice `json:"input"`
            Output []AudioDevice `json:"output"`
        } `json:"devices"`
        CurrentInput  *AudioDevice `json:"currentInput"`
        CurrentOutput *AudioDevice `json:"currentOutput"`
    } `json:"audio"`
    
    MIDI struct {
        Devices struct {
            Input  []MIDIDevice `json:"input"`
            Output []MIDIDevice `json:"output"`  
        } `json:"devices"`
        CurrentInput *MIDIDevice `json:"currentInput"`
    } `json:"midi"`
    
    Layouts struct {
        Available []string `json:"available"`
        Current   *Layout  `json:"current"`
        Recent    []string `json:"recent"`
    } `json:"layouts"`
    
    Settings struct {
        Audio AudioSettings `json:"audio"`
        UI    UISettings    `json:"ui"`
        MIDI  MIDISettings  `json:"midi"`
    } `json:"settings"`
    
    Editor struct {
        SelectedItem   *SelectedItem `json:"selectedItem"`
        UnsavedChanges bool         `json:"unsavedChanges"`
        DraftLayout    *Layout      `json:"draftLayout"`
    } `json:"editor"`
}
```

#### **Single API Endpoint**
```go
// ONE endpoint handles everything
GET    /api/state           // Get full app state
GET    /api/state?path=audio.devices.input  // Get specific subtree
DELETE /api/state?path=layouts.current.groups[2]  // Delete specific item
PUT    /api/state?path=layouts.current.groups     // Add new group
POST   /api/state?path=editor.selectedItem        // Edit selection
```

### **Implementation Benefits**

#### **1. Unified State Management**
```go
type Server struct {
    state     *AppState
    stateMux  sync.RWMutex
    // ... other fields
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
    path := r.URL.Query().Get("path")
    
    switch r.Method {
    case "GET":
        s.handleGetState(w, path)
    case "PUT":
        s.handleAddToState(w, r, path)
    case "POST":
        s.handleUpdateState(w, r, path)
    case "DELETE":
        s.handleDeleteFromState(w, path)
    }
}
```

#### **2. Path-Based Operations**
```go
func (s *Server) handleGetState(w http.ResponseWriter, path string) {
    s.stateMux.RLock()
    defer s.stateMux.RUnlock()
    
    if path == "" {
        // Return full state
        json.NewEncoder(w).Encode(s.state)
        return
    }
    
    // Return specific subtree
    value := getValueByPath(s.state, path)
    json.NewEncoder(w).Encode(value)
}
```

#### **3. Reactive Updates**
```go
func (s *Server) handleUpdateState(w http.ResponseWriter, r *http.Request, path string) {
    s.stateMux.Lock()
    defer s.stateMux.Unlock()
    
    var update interface{}
    json.NewDecoder(r.Body).Decode(&update)
    
    // Update specific path
    setValueByPath(s.state, path, update)
    
    // Trigger any side effects
    s.handleStateChange(path, update)
    
    // Return updated subtree
    value := getValueByPath(s.state, path)
    json.NewEncoder(w).Encode(value)
}
```

### **Go WASM Perfect Match**

#### **Shared State Structure**
```go
// Same AppState struct in WASM frontend
var appState *AppState

// Initialize from backend
func initializeApp() {
    resp := fetch("GET", "/api/state", nil)
    json.Unmarshal(resp, &appState)
    render()
}

// Update specific path
func updateAudioDevice(deviceID string) {
    device := findDevice(appState.Audio.Devices.Output, deviceID)
    appState.Audio.CurrentOutput = device
    
    // Sync to backend
    fetch("POST", "/api/state?path=audio.currentOutput", device)
    
    // Re-render affected templates
    renderAudioSection()
}
```

#### **Template Direct Binding**
```html
{{define "audio-settings"}}
<div class="audio-settings">
    <select onchange="updateAudioDevice(this.value)">
        {{range .Audio.Devices.Output}}
            <option value="{{.DeviceID}}" 
                    {{if eq $.Audio.CurrentOutput.DeviceID .DeviceID}}selected{{end}}>
                {{.Name}}
            </option>
        {{end}}
    </select>
    
    <div class="current-device">
        {{if .Audio.CurrentOutput}}
            Using: {{.Audio.CurrentOutput.Name}} 
            ({{.Audio.CurrentOutput.ChannelCount}} channels)
        {{end}}
    </div>
</div>
{{end}}
```

### **Localhost Performance Advantages**

#### **1. Zero Network Latency**
```go
// Localhost pipe - sub-millisecond responses
appState.Editor.SelectedItem = &SelectedItem{Type: "group", ID: groupID}
fetch("POST", "/api/state?path=editor.selectedItem", selectedItem)
// Response time: ~0.1ms vs ~10-50ms over network
```

#### **2. Bulk Operations**
```go
// Update multiple things atomically
update := map[string]interface{}{
    "layouts.current": newLayout,
    "editor.unsavedChanges": false,
    "layouts.recent": append(recent, newLayout.Name),
}
fetch("POST", "/api/state", update)
```

#### **3. Memory Sharing**
```go
// Backend and frontend can share large datasets efficiently
type PluginDatabase struct {
    AllPlugins    []AudioUnitPlugin `json:"allPlugins"`    // 62 plugins
    FilteredBy    map[string][]int  `json:"filteredBy"`    // Category indexes
    SearchIndex   map[string][]int  `json:"searchIndex"`   // Search indexes
}
// 1MB+ dataset - efficient over localhost, painful over network
```

### **API Simplification Examples**

#### **Current Complex Operations**
```javascript
// Vue.js nightmare we're dealing with
async addGroup(name) {
    // 1. Update local state
    this.currentLayout.groups.push(newGroup);
    
    // 2. Sync to backend (multiple calls)
    await fetch('/api/layouts/update', {...});
    await fetch('/api/settings', {currentLayout: this.currentLayout.name});
    
    // 3. Handle reactivity issues
    this.$forceUpdate();
    
    // 4. Update other components
    this.$emit('layout-changed');
}
```

#### **Go WASM Single Operation**
```go
// Simple, atomic, type-safe
func AddGroup(name string) {
    group := &Group{ID: generateID(), Name: name}
    appState.Layouts.Current.Groups = append(appState.Layouts.Current.Groups, group)
    appState.Editor.UnsavedChanges = true
    
    syncToBackend("layouts.current.groups", appState.Layouts.Current.Groups)
    renderCanvas()
}
```

---

## Implementation Strategy

### **Phase 1: Unified State Structure**
```go
// Create the mega-struct
type AppState struct { /* ... */ }

// Initialize from existing systems
func (s *Server) initializeState() {
    s.state = &AppState{}
    
    // Populate from existing systems
    s.state.Audio.Devices.Input = s.deviceEnum.GetAudioInputDevices()
    s.state.Audio.Devices.Output = s.deviceEnum.GetAudioOutputDevices()
    s.state.Layouts.Available = s.layoutManager.ListLayouts()
    // ...
}
```

### **Phase 2: Path-Based API**
```go
// Replace all existing endpoints with single handler
http.HandleFunc("/api/state", s.handleState)

// Redirect old endpoints during transition
http.HandleFunc("/api/devices/audio/input", func(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/api/state?path=audio.devices.input", http.MovedPermanently)
})
```

### **Phase 3: WASM Frontend**
```go
// Single WASM binary that mirrors backend state
//go:build wasm
func main() {
    initializeApp()
    registerEventHandlers()
    render()
}
```

---

## Migration Assessment

### **For AI Development**
1. **Single Mental Model**: One AppState struct instead of dozens of endpoints
2. **Path-Based Logic**: `updateValue(path, value)` instead of endpoint-specific handlers
3. **Type Safety**: Go compiler catches errors in state structure
4. **Atomic Operations**: One request updates multiple related fields
5. **No State Sync Issues**: Backend and frontend share identical structure

### **For Human Development**
1. **Localhost Performance**: Sub-millisecond responses enable real-time UX
2. **Go Expertise**: Leverage existing Go skills for both backend and frontend
3. **Single Language**: No JavaScript/Vue complexity
4. **Atomic Updates**: No partial state corruption
5. **Easy Testing**: Mock the single AppState, not dozens of endpoints

### **Trade-offs**

#### **Go WASM Disadvantages:**
- **Ecosystem**: Fewer UI component libraries
- **Bundle Size**: WASM files are larger than Vue
- **Learning Curve**: Less familiar for web developers
- **DOM Manipulation**: More manual work vs Vue's reactivity

#### **But for THIS Project:**
- **Audio Apps Don't Need Rich Ecosystems**: Custom knobs, sliders, canvas - all custom anyway
- **Desktop App**: Bundle size less critical than mobile web
- **Go-first Architecture**: Expertise aligns with Go patterns
- **AI Logic**: Much easier for AI to reason about single-language flow

---

## Conclusion

**The hierarchical single-endpoint approach combined with Go WASM frontend is not only feasible but ideal for localhost applications where network latency isn't a concern and maximum development velocity is desired.**

This architecture would eliminate 90% of the debugging complexity experienced with Vue state synchronization, provide compile-time safety, and create a single mental model for both backend and frontend development.

**Recommendation**: Proceed with planning the transition to Go WASM + Templates + Single Hierarchical API architecture.

---

*Migration plan documented: July 30, 2025*  
*Status: Architectural analysis complete, ready for implementation planning*
