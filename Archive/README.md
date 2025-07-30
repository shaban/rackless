# MC-SoFX Controller

A comprehensive controller interface for Neural DSP Morgan M Suite, featuring parameter introspection, flexible control layouts, and a modern web-based UI.

## Features

- **Automated Parameter Introspection**: Go program that executes the MC-SoFX tool and deserializes Neural DSP parameter data
- **Rich Parameter Data**: Extracts 128 parameters including indexed values, boolean controls, and generic parameters
- **Flexible Layout System**: JSON-based control layouts with grid positioning, multiple control types, and MIDI mapping
- **Layout Management Tools**: Command-line utilities for creating, validating, and managing control layouts
- **Auto-Generation**: Automatically generate layouts from introspection data
- **Modern Web UI**: Vue.js frontend with Tailwind CSS styling
- **Structured Output**: JSON data saved to `data/` directories for easy consumption

## Project Structure

```
MC-SoFX-Controller/
├── main.go                     # Main server application (all Go code in main package)
├── devices.go                  # Device enumeration with CGO
├── introspection.go            # AudioUnit introspection with CGO
├── layout_manager.go           # Layout system management
├── settings.go                 # Application settings
├── audiounit_devices.h/.m      # Native device enumeration (Objective-C)
├── audiounit_inspector.h/.m    # Native AudioUnit introspection (Objective-C)
├── libaudiounit_devices.a      # Static library for device functions
├── libaudiounit_inspector.a    # Static library for introspection functions
├── frontend/
│   └── app.html                # Vue.js frontend with Tailwind CSS
├── data/
│   ├── layouts/                # Control layout definitions 
│   └── mappings/               # MIDI mapping configurations
├── docs/                       # Documentation
├── bin/                        # Build tools (tailwindcss)
└── Makefile                    # Build automation
```

## Requirements

- Go 1.19 or later
- macOS (for Neural DSP Audio Unit)
- Neural DSP Morgan M Suite installed

## Quick Start

### 1. Build and run introspection
```bash
make introspection
```

### 2. Work with layouts
```bash
# List available layouts
make list-layouts

# Validate a layout
make validate-layout FILE=default-morgan-suite.json

# Generate a layout from introspection data
make generate-layout NAME=my-custom-layout

# View layout details
make show-layout FILE=default-morgan-suite.json
```

### 3. View the results
- Parameter data: `data/introspection/neural_dsp_parameters.json`
- Layout files: `data/layouts/*.json`
- Console output shows statistics

### 4. Start the web server
```bash
make start-server
# Then open http://localhost:8080 in your browser
```

## Parameter Types Extracted

- **Generic Parameters (88)**: Continuous controls like gain, EQ, etc.
- **Boolean Parameters (28)**: On/off switches for effects and sections  
- **Indexed Parameters (12)**: Dropdown selections with predefined values

## Control Layout System

The layout system provides flexible, grid-based control interfaces:

- **Grid Layouts**: Organize controls in rows and columns
- **Control Groups**: Logical groupings with custom styling
- **Multiple Control Types**: Knobs, sliders, buttons, radio groups
- **MIDI Integration**: Map controls to MIDI CC messages
- **Auto-Generation**: Create layouts from introspection data

See **[Layout System Documentation](docs/layouts.md)** for complete details.

### Example Parameters

```json
{
  "displayName": "Compressor Release",
  "address": 11,
  "unit": "Indexed",
  "indexedValues": ["Fast", "Slow"],
  "minValue": 0,
  "maxValue": 1
}
```

## Usage

### Command Line

```bash
# Build all tools
make build

# Run parameter extraction
make introspection

# Layout management
make list-layouts                              # List all layouts
make validate-layout FILE=layout.json         # Validate a layout
make generate-layout NAME=my-layout           # Generate from data
make show-layout FILE=layout.json             # Show layout details

# Clean build artifacts
make clean
```

### Programmatic Usage

#### Introspection
```go
package main

import (
    "mc-sofx-controller/pkg/introspection"
)

func main() {
    // Introspection is now built-in via CGO, no external tool needed
    runner := introspection.NewRunner("", "")
    result, err := runner.Execute()
    if err != nil {
        panic(err)
    }
    
    // Access parameters
    inputGain := result.GetParameterByName("Input Gain")
    booleanParams := result.GetBooleanParameters()
    indexedParams := result.GetIndexedParameters()
}
```

#### Layout Management
```go
package main

import (
    "mc-sofx-controller/pkg/layout"
)

func main() {
    manager := layout.NewLayoutManager("data/layouts")
    
    // Load and validate layout
    layout, err := manager.LoadLayout("default-morgan-suite.json")
    if err != nil {
        panic(err)
    }
    
    // Access layout components
    grid := layout.Grid
    groups := layout.Groups
}
```

## Development

The project uses:
- **Go**: Backend introspection, layout management, and data processing
- **Vue.js 3**: Frontend framework  
- **Tailwind CSS**: Utility-first CSS framework
- **JSON**: Data interchange format for parameters and layouts

## Documentation

- **[Layout System](docs/layouts.md)**: Complete guide to the control layout system
- **[AI Context](docs/ai-context.md)**: Comprehensive technical architecture and development context

## Output Format

The introspection generates comprehensive JSON data including:
- Parameter addresses, names, and identifiers
- Min/max values and current states
- Parameter units (Generic, Boolean, Indexed)
- Indexed parameter value lists
- Ramping capabilities and write permissions
- Raw Audio Unit flags

## Next Steps

- [ ] Frontend integration with layout system
- [ ] MIDI controller integration  
- [ ] Real-time parameter control via Audio Unit
- [ ] Visual layout editor
- [ ] Preset management system
- [ ] Advanced UI components for different parameter types
- [ ] Theme system for custom styling
