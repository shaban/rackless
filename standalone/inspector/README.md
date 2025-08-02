# Rackless Inspector - Plugin & Device Discovery

Comprehensive AudioUnit plugin and audio device discovery tool for the Rackless project.

## Features

- **Complete Plugin Database**: Discovers 62+ AudioUnit plugins
- **Parameter Introspection**: Full parameter metadata with automation flags
- **Indexed Parameters**: Extract dropdown/menu values from plugins
- **Device Analysis**: Detailed audio device capabilities
- **JSON Export**: Clean JSON output for integration
- **Comprehensive Metadata**: Names, ranges, units, automation flags

## Building

```bash
make
```

## Usage

### Full System Analysis
```bash
./inspector > system-analysis.json
```

### Plugin Names Only
```bash
./inspector 2>/dev/null | jq '.[] | .name'
```

### Plugin Count
```bash
./inspector 2>/dev/null | jq '. | length'
```

### Find Indexed Parameters
```bash
./inspector 2>/dev/null | jq '.[] | select(.parameters[]?.isIndexed == true) | .name'
```

## Output Format

The inspector outputs a JSON array of plugin objects:

```json
[
  {
    "name": "Plugin Name",
    "manufacturerID": "appl",
    "type": "aufc",
    "subtype": "merg",
    "parameters": [
      {
        "displayName": "Parameter Name",
        "identifier": "0",
        "address": 0,
        "minValue": 0,
        "maxValue": 100,
        "defaultValue": 50,
        "currentValue": 50,
        "unit": "Generic",
        "isWritable": true,
        "canRamp": false,
        "isIndexed": false,
        "rawFlags": 3221225473
      }
    ]
  }
]
```

## Performance

- Full scan takes ~30 seconds (timeout protection included)
- Discovers 62+ plugins with complete parameter metadata
- Uses async processing for reliability
- Comprehensive indexed parameter extraction

## Integration

Use this tool to:
1. Build plugin databases for applications
2. Generate automation mappings
3. Discover available audio capabilities
4. Export system configuration
