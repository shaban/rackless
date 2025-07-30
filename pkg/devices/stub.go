//go:build !darwin || !cgo

package devices

import (
	"time"
)

// stubDeviceEnumerator provides a cross-platform fallback implementation
type stubDeviceEnumerator struct {
	config DeviceEnumerationConfig
}

// NewDeviceEnumerator creates a new device enumerator with default configuration
func NewDeviceEnumerator() DeviceEnumerator {
	return &stubDeviceEnumerator{
		config: DefaultConfig(),
	}
}

// NewDeviceEnumeratorWithConfig creates a new device enumerator with custom configuration
func NewDeviceEnumeratorWithConfig(config DeviceEnumerationConfig) DeviceEnumerator {
	return &stubDeviceEnumerator{
		config: config,
	}
}

// GetAudioInputDevices implements DeviceEnumerator.GetAudioInputDevices
func (de *stubDeviceEnumerator) GetAudioInputDevices() ([]AudioDevice, error) {
	return []AudioDevice{
		{
			Name:                 "Mock Audio Input",
			UID:                  "mock_input",
			DeviceID:             1,
			ChannelCount:         2,
			SupportedSampleRates: []float64{44100, 48000},
			SupportedBitDepths:   []int{16, 24},
			IsDefault:            true,
		},
	}, nil
}

// GetAudioOutputDevices implements DeviceEnumerator.GetAudioOutputDevices
func (de *stubDeviceEnumerator) GetAudioOutputDevices() ([]AudioDevice, error) {
	return []AudioDevice{
		{
			Name:                 "Mock Audio Output",
			UID:                  "mock_output",
			DeviceID:             2,
			ChannelCount:         2,
			SupportedSampleRates: []float64{44100, 48000},
			SupportedBitDepths:   []int{16, 24},
			IsDefault:            true,
		},
	}, nil
}

// GetMIDIInputDevices implements DeviceEnumerator.GetMIDIInputDevices
func (de *stubDeviceEnumerator) GetMIDIInputDevices() ([]MIDIDevice, error) {
	return []MIDIDevice{
		{
			Name:       "Mock MIDI Input",
			UID:        "mock_midi_input",
			EndpointID: 1,
			IsOnline:   true,
		},
	}, nil
}

// GetMIDIOutputDevices implements DeviceEnumerator.GetMIDIOutputDevices
func (de *stubDeviceEnumerator) GetMIDIOutputDevices() ([]MIDIDevice, error) {
	return []MIDIDevice{
		{
			Name:       "Mock MIDI Output",
			UID:        "mock_midi_output",
			EndpointID: 2,
			IsOnline:   true,
		},
	}, nil
}

// GetDefaultAudioDevices implements DeviceEnumerator.GetDefaultAudioDevices
func (de *stubDeviceEnumerator) GetDefaultAudioDevices() (DefaultAudioDevices, error) {
	return DefaultAudioDevices{
		DefaultInput:  1,
		DefaultOutput: 2,
	}, nil
}

// GetAllDevices implements DeviceEnumerator.GetAllDevices
func (de *stubDeviceEnumerator) GetAllDevices() (DeviceEnumerationResult, error) {
	start := time.Now()
	
	audioInputs, _ := de.GetAudioInputDevices()
	audioOutputs, _ := de.GetAudioOutputDevices()
	midiInputs, _ := de.GetMIDIInputDevices()
	midiOutputs, _ := de.GetMIDIOutputDevices()
	defaultDevices, _ := de.GetDefaultAudioDevices()
	
	// Add "(None Selected)" options for safe defaults
	audioInputsWithNone := append([]AudioDevice{{
		Name:         "(None Selected)",
		UID:          "none",
		DeviceID:     -1,
		ChannelCount: 0,
		IsDefault:    true,
	}}, audioInputs...)
	
	midiInputsWithNone := append([]MIDIDevice{{
		Name:       "(None Selected)",
		UID:        "none",
		EndpointID: -1,
		IsOnline:   true,
	}}, midiInputs...)
	
	midiOutputsWithNone := append([]MIDIDevice{{
		Name:       "(None Selected)",
		UID:        "none",
		EndpointID: -1,
		IsOnline:   true,
	}}, midiOutputs...)
	
	return DeviceEnumerationResult{
		AudioInputs:     audioInputsWithNone,
		AudioOutputs:    audioOutputs,
		MIDIInputs:      midiInputsWithNone,
		MIDIOutputs:     midiOutputsWithNone,
		DefaultDevices:  defaultDevices,
		Success:         true,
		EnumerationTime: time.Since(start),
	}, nil
}
