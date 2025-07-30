//go:build !js && !wasm

package devices

import (
	"testing"
)

// TestNewDeviceEnumerator tests basic enumerator creation
func TestNewDeviceEnumerator(t *testing.T) {
	enumerator := NewDeviceEnumerator()
	if enumerator == nil {
		t.Fatal("NewDeviceEnumerator() returned nil")
	}
}

// TestNewDeviceEnumeratorWithConfig tests enumerator creation with custom config
func TestNewDeviceEnumeratorWithConfig(t *testing.T) {
	config := DeviceEnumerationConfig{
		IncludeOfflineDevices: true,
		IncludeVirtualDevices: false,
	}
	
	enumerator := NewDeviceEnumeratorWithConfig(config)
	if enumerator == nil {
		t.Fatal("NewDeviceEnumeratorWithConfig() returned nil")
	}
}

// TestDefaultConfig tests the default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.IncludeOfflineDevices != false {
		t.Errorf("Expected IncludeOfflineDevices to be false by default")
	}
	
	if config.IncludeVirtualDevices != true {
		t.Errorf("Expected IncludeVirtualDevices to be true by default")
	}
}

// TestGetAudioInputDevices tests audio input device enumeration
func TestGetAudioInputDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping device enumeration test in short mode")
	}
	
	enumerator := NewDeviceEnumerator()
	devices, err := enumerator.GetAudioInputDevices()
	if err != nil {
		t.Fatalf("GetAudioInputDevices() failed: %v", err)
	}
	
	// Should at least return an empty slice, not nil
	if devices == nil {
		t.Fatal("GetAudioInputDevices() returned nil")
	}
	
	t.Logf("Found %d audio input devices", len(devices))
	
	// Validate device structures
	for i, device := range devices {
		if device.Name == "" {
			t.Errorf("Device %d has empty name", i)
		}
		if device.UID == "" {
			t.Errorf("Device %d has empty UID", i)
		}
		if device.ChannelCount < 0 {
			t.Errorf("Device %d has negative channel count: %d", i, device.ChannelCount)
		}
	}
}

// TestGetAudioOutputDevices tests audio output device enumeration
func TestGetAudioOutputDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping device enumeration test in short mode")
	}
	
	enumerator := NewDeviceEnumerator()
	devices, err := enumerator.GetAudioOutputDevices()
	if err != nil {
		t.Fatalf("GetAudioOutputDevices() failed: %v", err)
	}
	
	// Should at least return an empty slice, not nil
	if devices == nil {
		t.Fatal("GetAudioOutputDevices() returned nil")
	}
	
	t.Logf("Found %d audio output devices", len(devices))
	
	// Validate device structures
	for i, device := range devices {
		if device.Name == "" {
			t.Errorf("Device %d has empty name", i)
		}
		if device.UID == "" {
			t.Errorf("Device %d has empty UID", i)
		}
		if device.ChannelCount < 0 {
			t.Errorf("Device %d has negative channel count: %d", i, device.ChannelCount)
		}
	}
}

// TestGetMIDIInputDevices tests MIDI input device enumeration
func TestGetMIDIInputDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping device enumeration test in short mode")
	}
	
	enumerator := NewDeviceEnumerator()
	devices, err := enumerator.GetMIDIInputDevices()
	if err != nil {
		t.Fatalf("GetMIDIInputDevices() failed: %v", err)
	}
	
	// Should at least return an empty slice, not nil
	if devices == nil {
		t.Fatal("GetMIDIInputDevices() returned nil")
	}
	
	t.Logf("Found %d MIDI input devices", len(devices))
	
	// Validate device structures
	for i, device := range devices {
		if device.Name == "" {
			t.Errorf("Device %d has empty name", i)
		}
		if device.UID == "" {
			t.Errorf("Device %d has empty UID", i)
		}
	}
}

// TestGetMIDIOutputDevices tests MIDI output device enumeration
func TestGetMIDIOutputDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping device enumeration test in short mode")
	}
	
	enumerator := NewDeviceEnumerator()
	devices, err := enumerator.GetMIDIOutputDevices()
	if err != nil {
		t.Fatalf("GetMIDIOutputDevices() failed: %v", err)
	}
	
	// Should at least return an empty slice, not nil
	if devices == nil {
		t.Fatal("GetMIDIOutputDevices() returned nil")
	}
	
	t.Logf("Found %d MIDI output devices", len(devices))
	
	// Validate device structures
	for i, device := range devices {
		if device.Name == "" {
			t.Errorf("Device %d has empty name", i)
		}
		if device.UID == "" {
			t.Errorf("Device %d has empty UID", i)
		}
	}
}

// TestGetDefaultAudioDevices tests default device detection
func TestGetDefaultAudioDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping device enumeration test in short mode")
	}
	
	enumerator := NewDeviceEnumerator()
	defaults, err := enumerator.GetDefaultAudioDevices()
	if err != nil {
		t.Fatalf("GetDefaultAudioDevices() failed: %v", err)
	}
	
	t.Logf("Default input: %d, output: %d", defaults.DefaultInput, defaults.DefaultOutput)
	
	// Default device IDs can be 0 (meaning no default), but shouldn't be negative
	if defaults.DefaultInput < 0 {
		t.Errorf("Default input device ID is negative: %d", defaults.DefaultInput)
	}
	if defaults.DefaultOutput < 0 {
		t.Errorf("Default output device ID is negative: %d", defaults.DefaultOutput)
	}
}

// TestGetAllDevices tests comprehensive device enumeration
func TestGetAllDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comprehensive device enumeration test in short mode")
	}
	
	enumerator := NewDeviceEnumerator()
	result, err := enumerator.GetAllDevices()
	if err != nil {
		t.Fatalf("GetAllDevices() failed: %v", err)
	}
	
	if !result.Success {
		t.Fatalf("GetAllDevices() reported failure: %s", result.Error)
	}
	
	if result.EnumerationTime <= 0 {
		t.Error("EnumerationTime should be positive")
	}
	
	t.Logf("Device enumeration completed in %v", result.EnumerationTime)
	t.Logf("Found: %d audio inputs, %d audio outputs, %d MIDI inputs, %d MIDI outputs",
		len(result.AudioInputs), len(result.AudioOutputs), len(result.MIDIInputs), len(result.MIDIOutputs))
	
	// Should have at least the "(None Selected)" options
	if len(result.AudioInputs) == 0 {
		t.Error("Expected at least one audio input device (None Selected)")
	}
	if len(result.MIDIInputs) == 0 {
		t.Error("Expected at least one MIDI input device (None Selected)")
	}
	if len(result.MIDIOutputs) == 0 {
		t.Error("Expected at least one MIDI output device (None Selected)")
	}
	
	// Check for "(None Selected)" options
	foundNoneAudioInput := false
	for _, device := range result.AudioInputs {
		if device.Name == "(None Selected)" && device.UID == "none" {
			foundNoneAudioInput = true
			break
		}
	}
	if !foundNoneAudioInput {
		t.Error("Missing '(None Selected)' option in audio inputs")
	}
	
	foundNoneMIDIInput := false
	for _, device := range result.MIDIInputs {
		if device.Name == "(None Selected)" && device.UID == "none" {
			foundNoneMIDIInput = true
			break
		}
	}
	if !foundNoneMIDIInput {
		t.Error("Missing '(None Selected)' option in MIDI inputs")
	}
}
