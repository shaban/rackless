package devices

import "time"

// AudioDevice represents an audio input or output device
type AudioDevice struct {
	Name                 string    `json:"name"`
	UID                  string    `json:"uid"`
	DeviceID             int       `json:"deviceId"`
	ChannelCount         int       `json:"channelCount"`
	SupportedSampleRates []float64 `json:"supportedSampleRates"`
	SupportedBitDepths   []int     `json:"supportedBitDepths"`
	IsDefault            bool      `json:"isDefault"`
}

// MIDIDevice represents a MIDI input or output device
type MIDIDevice struct {
	Name       string `json:"name"`
	UID        string `json:"uid"`
	EndpointID int    `json:"endpointId"`
	IsOnline   bool   `json:"isOnline"`
}

// DefaultAudioDevices represents the system's default audio devices
type DefaultAudioDevices struct {
	DefaultInput  int `json:"defaultInput"`
	DefaultOutput int `json:"defaultOutput"`
}

// DeviceEnumerationResult contains all enumerated devices
type DeviceEnumerationResult struct {
	AudioInputs     []AudioDevice       `json:"audioInputs"`
	AudioOutputs    []AudioDevice       `json:"audioOutputs"`
	MIDIInputs      []MIDIDevice        `json:"midiInputs"`
	MIDIOutputs     []MIDIDevice        `json:"midiOutputs"`
	DefaultDevices  DefaultAudioDevices `json:"defaultDevices"`
	EnumerationTime time.Duration       `json:"enumerationTime"`
	Success         bool                `json:"success"`
	Error           string              `json:"error,omitempty"`
}

// DeviceEnumerator interface defines the device enumeration capabilities
type DeviceEnumerator interface {
	// GetAudioInputDevices returns all available audio input devices
	GetAudioInputDevices() ([]AudioDevice, error)

	// GetAudioOutputDevices returns all available audio output devices
	GetAudioOutputDevices() ([]AudioDevice, error)

	// GetMIDIInputDevices returns all available MIDI input devices
	GetMIDIInputDevices() ([]MIDIDevice, error)

	// GetMIDIOutputDevices returns all available MIDI output devices
	GetMIDIOutputDevices() ([]MIDIDevice, error)

	// GetDefaultAudioDevices returns the system's default audio devices
	GetDefaultAudioDevices() (DefaultAudioDevices, error)

	// GetAllDevices returns a comprehensive enumeration of all devices
	GetAllDevices() (DeviceEnumerationResult, error)
}

// DeviceEnumerationConfig holds configuration for device enumeration
type DeviceEnumerationConfig struct {
	IncludeOfflineDevices bool `json:"includeOfflineDevices"`
	IncludeVirtualDevices bool `json:"includeVirtualDevices"`
}

// DefaultConfig returns the default device enumeration configuration
func DefaultConfig() DeviceEnumerationConfig {
	return DeviceEnumerationConfig{
		IncludeOfflineDevices: false,
		IncludeVirtualDevices: true,
	}
}
