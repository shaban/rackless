package main

import (
	"log"
	"testing"
)

func TestDeviceEnumeration(t *testing.T) {
	if DeviceEnum == nil {
		t.Fatal("DeviceEnum is nil - init() function not called or failed")
	}

	t.Run("AudioInputDevices", func(t *testing.T) {
		devices, err := DeviceEnum.GetAudioInputDevices()
		if err != nil {
			t.Errorf("Failed to get audio input devices: %v", err)
			return
		}
		log.Printf("Found %d audio input devices", len(devices))
		for i, device := range devices {
			log.Printf("  [%d] %s (ID: %d, Channels: %d)", i, device.Name, device.DeviceID, device.ChannelCount)
		}
	})

	t.Run("AudioOutputDevices", func(t *testing.T) {
		devices, err := DeviceEnum.GetAudioOutputDevices()
		if err != nil {
			t.Errorf("Failed to get audio output devices: %v", err)
			return
		}
		log.Printf("Found %d audio output devices", len(devices))
		for i, device := range devices {
			log.Printf("  [%d] %s (ID: %d, Channels: %d)", i, device.Name, device.DeviceID, device.ChannelCount)
		}
	})

	t.Run("MIDIInputDevices", func(t *testing.T) {
		devices, err := DeviceEnum.GetMIDIInputDevices()
		if err != nil {
			t.Errorf("Failed to get MIDI input devices: %v", err)
			return
		}
		log.Printf("Found %d MIDI input devices", len(devices))
		for i, device := range devices {
			log.Printf("  [%d] %s (ID: %d, Online: %t)", i, device.Name, device.EndpointID, device.IsOnline)
		}
	})

	t.Run("MIDIOutputDevices", func(t *testing.T) {
		devices, err := DeviceEnum.GetMIDIOutputDevices()
		if err != nil {
			t.Errorf("Failed to get MIDI output devices: %v", err)
			return
		}
		log.Printf("Found %d MIDI output devices", len(devices))
		for i, device := range devices {
			log.Printf("  [%d] %s (ID: %d, Online: %t)", i, device.Name, device.EndpointID, device.IsOnline)
		}
	})

	t.Run("DefaultAudioDevices", func(t *testing.T) {
		defaults, err := DeviceEnum.GetDefaultAudioDevices()
		if err != nil {
			t.Errorf("Failed to get default audio devices: %v", err)
			return
		}
		log.Printf("Default input: %d, Default output: %d", defaults.DefaultInput, defaults.DefaultOutput)
	})

	t.Run("AllDevices", func(t *testing.T) {
		allDevices, err := DeviceEnum.GetAllDevices()
		if err != nil {
			t.Errorf("Failed to get all devices: %v", err)
			return
		}

		// Check that all expected keys are present
		expectedKeys := []string{"audioInputs", "audioOutputs", "midiInputs", "midiOutputs", "defaultDevices"}
		for _, key := range expectedKeys {
			if _, exists := allDevices[key]; !exists {
				t.Errorf("Missing key in allDevices: %s", key)
			}
		}

		log.Printf("All devices response contains %d categories", len(allDevices))
	})
}

func TestDeviceEnumeratorNilSafety(t *testing.T) {
	// Test what happens if we create a new enumerator
	enum := NewDeviceEnumerator()
	if enum == nil {
		t.Fatal("NewDeviceEnumerator returned nil")
	}

	// Test basic functionality
	devices, err := enum.GetAudioOutputDevices()
	if err != nil {
		t.Errorf("New enumerator failed to get audio output devices: %v", err)
	} else {
		t.Logf("New enumerator found %d audio output devices", len(devices))
	}
}
