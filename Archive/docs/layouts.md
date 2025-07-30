# MC-SoFX Controller - Layout System

The MC-SoFX Controller includes a comprehensive layout system for defining control interfaces for Neural DSP plugins and other audio units.

## Overview

The layout system consists of:

- **Control Layouts**: JSON files defining the arrangement and behavior of controls
- **Layout Manager**: Go functions (in main package) for loading, saving, and validating layouts
- **Layout Tool**: Command-line utility for working with layouts

## Layout Structure

A layout defines a grid-based interface with groups of controls:

```
Layout
├── Grid (rows × columns)
├── Groups (positioned on grid)
│   ├── Controls (knobs, sliders, buttons)
│   │   └── Targets (parameter mappings, MIDI CC)
│   └── Background styling
└── Metadata (name, version, description)
```

### Grid System

Layouts use a flexible grid system:
- **Rows & Columns**: Define the overall layout dimensions
- **Gutter**: Spacing between grid cells
- **Spanning**: Groups can span multiple cells

### Control Types

#### Switch Controls
- **toggle_button**: On/off button
- **checkbox**: Checkbox input

#### Radio Controls  
- **radio_group**: Traditional radio button group
- **toggle_button_group**: Mutually-exclusive toggle buttons
- **stepped_slider**: Slider with discrete steps

#### Range Controls
- **rotary_knob**: Circular knob control
- **ranged_slider**: Linear slider with range

### Targeting System

Each control can target multiple endpoints:

#### Parameter Targeting
- **Parameter Address**: Direct AU parameter address
- **Parameter Name**: Human-readable parameter name

#### MIDI Targeting  
- **CC Number**: MIDI Control Change number (1-127)
- **Channel**: MIDI channel (1-16)
- **Stepped**: Whether to send discrete vs continuous values

## File Structure

```
data/
└── layouts/
    ├── default-morgan-suite.json    # Hand-crafted default layout
    └── auto-generated-morgan.json   # Auto-generated from introspection
```

## Layout Tool Usage

### Build Tools
```bash
make build              # Build all tools
make build-layouts      # Build just the layout tool
```

### List Layouts
```bash
make list-layouts
```

### Validate Layout
```bash
make validate-layout FILE=default-morgan-suite.json
```

### Show Layout Details  
```bash
make show-layout FILE=default-morgan-suite.json
```

### Generate Layout from Introspection
```bash
# Use default introspection data
make generate-layout NAME=my-layout

# Specify custom data file
make generate-layout NAME=my-layout DATA=custom-data.json OUTPUT=my-layout.json
```

## Layout Format Specification

### Layout Root Object
```json
{
  "name": "Layout Name",
  "description": "Layout description", 
  "version": "1.0.0",
  "grid": {
    "rows": 3,
    "columns": 4,
    "gutter": 10
  },
  "groups": [...]
}
```

### Group Object
```json
{
  "label": "Group Name",
  "id": "unique-group-id",
  "bgType": "color|image|gradient",
  "bgValue": "#2a2a2a",
  "order": 1,
  "colSpan": 1,
  "rowSpan": 1, 
  "x": 0,
  "y": 0,
  "controls": [...]
}
```

### Control Object
```json
{
  "label": "Control Name",
  "id": "unique-control-id",
  "type": "switch|radio|range",
  "implementation": "toggle_button|rotary_knob|etc",
  "x": 10,
  "y": 10,
  "width": 50,
  "height": 50,
  "minValue": 0.0,
  "maxValue": 1.0,
  "targets": [...]
}
```

### Target Object
```json
{
  "parameterAddress": 42,
  "parameterName": "Gain",
  "ccMidi": 7,
  "channel": 1,
  "stepped": false
}
```

## Layout Manager API

### Creating a Manager
```go
import "mc-sofx-controller/pkg/layout"

manager := layout.NewLayoutManager("data/layouts")
```

### Loading Layouts
```go
// Load specific layout
layout, err := manager.LoadLayout("default-morgan-suite.json")

// Load all layouts in directory
err := manager.LoadAllLayouts()

// Get cached layout
layout := manager.GetLayout("Layout Name")
```

### Validating Layouts
```go
err := manager.ValidateLayout(layout)
```

### Generating Layouts
```go  
import "mc-sofx-controller/pkg/introspection"

// Load introspection data
var result introspection.IntrospectionResult
// ... load data ...

// Generate layout
layout := manager.GenerateLayoutFromIntrospection(&result, "My Layout")

// Save layout
err := manager.SaveLayout(layout, "my-layout.json")
```

## Best Practices

### Layout Design
- **Group Related Controls**: Organize controls by function (Input, Amp, Effects, etc.)
- **Use Appropriate Control Types**: Match control type to parameter behavior
- **Consider Screen Space**: Balance detail with usability
- **Provide MIDI Mapping**: Enable external controller integration

### File Organization
- **Descriptive Names**: Use clear, descriptive filenames
- **Version Control**: Include version numbers in layouts
- **Validation**: Always validate layouts before deployment
- **Documentation**: Include descriptions for complex layouts

### Performance
- **Cache Layouts**: Use the layout manager's caching system
- **Validate Once**: Validate layouts at load time, not runtime
- **Batch Operations**: Load multiple layouts efficiently

## Examples

See the following example layouts:

- **`default-morgan-suite.json`**: Hand-crafted layout showcasing best practices
- **`auto-generated-morgan.json`**: Auto-generated layout showing all 128 parameters

## Integration

The layout system integrates with:

- **Introspection System**: Auto-generate layouts from parameter data
- **Frontend**: Vue.js components render controls based on layout specs
- **MIDI System**: Route MIDI CC messages to appropriate parameters
- **Audio Units**: Map controls to AU parameter addresses

## Future Enhancements

Planned improvements include:

- **Visual Layout Editor**: GUI tool for creating/editing layouts
- **Template System**: Reusable layout templates
- **Dynamic Layouts**: Runtime layout modification
- **Theme Support**: Custom styling and themes
- **Preset Integration**: Layout-aware preset management
