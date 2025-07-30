# Standalone Audio Host

Pure Objective-C command-line tool for testing Core Audio functionality.

## Build & Run

```bash
# Build
make

# Run interactively
make run
# or
./audio-host

# Test for 5 seconds (requires gtimeout)
make test

# Test silent mode
make test-silent
```

## Command Line Options

```bash
# Basic usage with test tone
./audio-host

# Silent mode (no test tone)
./audio-host --no-tone

# Custom sample rate
./audio-host --sample-rate 48000

# Custom buffer size
./audio-host --buffer-size 512

# Help
./audio-host --help
```

## Features

- ✅ **Core Audio Integration**: Uses AudioUnit for real-time audio output
- ✅ **Test Tone Generator**: 440Hz sine wave for testing
- ✅ **Configurable**: Sample rate, buffer size, test tone on/off
- ✅ **Real-time Processing**: Proper Core Audio render callback
- ✅ **Clean Shutdown**: Ctrl+C handling

## Audio Configuration

- **Default Sample Rate**: 44100 Hz
- **Audio Format**: 32-bit Float, Stereo
- **Default Buffer**: 256 samples
- **Output Device**: System default

## Next Steps

1. **Add Audio Input**: Microphone → output passthrough
2. **Plugin Hosting**: Load and process AudioUnit plugins
3. **Parameter Control**: Real-time plugin parameter automation
4. **MIDI Integration**: MIDI CC → plugin parameter mapping

## Requirements

- macOS with Core Audio
- Xcode command line tools (`clang`)
