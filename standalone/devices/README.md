# Devices - Unified Device Enumeration

Comprehensive audio and MIDI device discovery tool using Archive device enumeration functions.

## Features

- **Complete Device Discovery**: Audio input/output and MIDI input/output devices
- **Rich Device Information**: Sample rates, bit depths, channel counts, UIDs
- **Default Device Detection**: System default input/output devices
- **Unified JSON Structure**: Single command gives complete system overview
- **Integration Ready**: Clean JSON output for consumption by other tools

## Building

```bash
make
```

## Usage

### Complete System Overview
```bash
./devices > system-devices.json
```

### Device Categories
```bash
./devices 2>/dev/null | jq 'keys'
```

### Device Counts
```bash
./devices 2>/dev/null | jq '{totalAudioInputDevices, totalAudioOutputDevices, totalMIDIInputDevices, totalMIDIOutputDevices}'
```

### Sample Device Information
```bash
./devices 2>/dev/null | jq '.audioInput[0]'
```

### Default Devices
```bash
./devices 2>/dev/null | jq '.defaults'
```

## Output Structure

The devices tool outputs a unified JSON structure:

```json
{
  "audioInput": [
    {
      "name": "Device Name",
      "uid": "device_123",
      "deviceId": 123,
      "channelCount": 2,
      "supportedSampleRates": [44100, 48000, 96000],
      "supportedBitDepths": [16, 24, 32],
      "isDefault": false
    }
  ],
  "audioOutput": [...],
  "midiInput": [
    {
      "name": "MIDI Device",
      "uid": "midi_456",
      "endpointId": 456,
      "isOnline": true
    }
  ],
  "midiOutput": [...],
  "defaults": {
    "defaultInput": 123,
    "defaultOutput": 87
  },
  "totalAudioInputDevices": 9,
  "totalAudioOutputDevices": 10,
  "totalMIDIInputDevices": 10,
  "totalMIDIOutputDevices": 9,
  "timestamp": "2025-07-30 19:30:00 +0000"
}
```

## Integration

This tool is designed to provide complete device information for:

1. **Go Server Applications**: REST API endpoints for device lists
2. **Web Frontend**: Device selection dropdowns and capabilities
3. **Audio Applications**: Device configuration and routing
4. **Automation Systems**: Device monitoring and selection

## Archive Compatibility

Uses the proven device enumeration functions from `Archive/audiounit_devices.m` ensuring:
- Complete device discovery
- Robust error handling
- Detailed device capabilities
- System default detection
