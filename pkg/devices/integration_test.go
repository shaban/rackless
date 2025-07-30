//go:build integration

package devices

import (
	"encoding/json"
	"testing"
	"time"
)

// TestIntegrationDeviceEnumeration performs a comprehensive integration test
func TestIntegrationDeviceEnumeration(t *testing.T) {
	t.Log("🎛️  Starting device enumeration integration test...")
	
	enumerator := NewDeviceEnumerator()
	
	// Test complete enumeration
	start := time.Now()
	result, err := enumerator.GetAllDevices()
	elapsed := time.Since(start)
	
	if err != nil {
		t.Fatalf("❌ Device enumeration failed: %v", err)
	}
	
	if !result.Success {
		t.Fatalf("❌ Device enumeration reported failure: %s", result.Error)
	}
	
	t.Logf("✅ Device enumeration completed in %v", elapsed)
	t.Logf("📊 Found: %d audio inputs, %d audio outputs, %d MIDI inputs, %d MIDI outputs",
		len(result.AudioInputs), len(result.AudioOutputs), len(result.MIDIInputs), len(result.MIDIOutputs))
	
	// Validate JSON serialization
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("❌ Failed to marshal device enumeration result to JSON: %v", err)
	}
	
	t.Logf("📄 JSON output size: %.1f KB", float64(len(jsonData))/1024)
	
	// Validate that we can unmarshal it back
	var parsedResult DeviceEnumerationResult
	if err := json.Unmarshal(jsonData, &parsedResult); err != nil {
		t.Fatalf("❌ Failed to unmarshal device enumeration JSON: %v", err)
	}
	
	// Basic validation of parsed result
	if parsedResult.Success != result.Success {
		t.Error("❌ Success flag mismatch after JSON round-trip")
	}
	
	if len(parsedResult.AudioInputs) != len(result.AudioInputs) {
		t.Error("❌ Audio input count mismatch after JSON round-trip")
	}
	
	// Test individual component enumeration
	t.Run("AudioInputs", func(t *testing.T) {
		devices, err := enumerator.GetAudioInputDevices()
		if err != nil {
			t.Fatalf("❌ Audio input enumeration failed: %v", err)
		}
		t.Logf("🎤 Found %d audio input devices", len(devices))
		
		for i, device := range devices {
			if device.Name == "" {
				t.Errorf("❌ Audio input device %d has empty name", i)
			}
			if device.UID == "" {
				t.Errorf("❌ Audio input device %d has empty UID", i)
			}
			t.Logf("  - %s (%d channels)", device.Name, device.ChannelCount)
		}
	})
	
	t.Run("AudioOutputs", func(t *testing.T) {
		devices, err := enumerator.GetAudioOutputDevices()
		if err != nil {
			t.Fatalf("❌ Audio output enumeration failed: %v", err)
		}
		t.Logf("🔊 Found %d audio output devices", len(devices))
		
		for i, device := range devices {
			if device.Name == "" {
				t.Errorf("❌ Audio output device %d has empty name", i)
			}
			if device.UID == "" {
				t.Errorf("❌ Audio output device %d has empty UID", i)
			}
			t.Logf("  - %s (%d channels)", device.Name, device.ChannelCount)
		}
	})
	
	t.Run("MIDIInputs", func(t *testing.T) {
		devices, err := enumerator.GetMIDIInputDevices()
		if err != nil {
			t.Fatalf("❌ MIDI input enumeration failed: %v", err)
		}
		t.Logf("🎹 Found %d MIDI input devices", len(devices))
		
		for i, device := range devices {
			if device.Name == "" {
				t.Errorf("❌ MIDI input device %d has empty name", i)
			}
			if device.UID == "" {
				t.Errorf("❌ MIDI input device %d has empty UID", i)
			}
			t.Logf("  - %s (online: %t)", device.Name, device.IsOnline)
		}
	})
	
	t.Run("MIDIOutputs", func(t *testing.T) {
		devices, err := enumerator.GetMIDIOutputDevices()
		if err != nil {
			t.Fatalf("❌ MIDI output enumeration failed: %v", err)
		}
		t.Logf("🎹 Found %d MIDI output devices", len(devices))
		
		for i, device := range devices {
			if device.Name == "" {
				t.Errorf("❌ MIDI output device %d has empty name", i)
			}
			if device.UID == "" {
				t.Errorf("❌ MIDI output device %d has empty UID", i)
			}
			t.Logf("  - %s (online: %t)", device.Name, device.IsOnline)
		}
	})
	
	t.Run("DefaultDevices", func(t *testing.T) {
		defaults, err := enumerator.GetDefaultAudioDevices()
		if err != nil {
			t.Fatalf("❌ Default device detection failed: %v", err)
		}
		t.Logf("⚙️  Default devices: input ID %d, output ID %d", 
			defaults.DefaultInput, defaults.DefaultOutput)
	})
	
	// Test performance characteristics
	if elapsed > 10*time.Second {
		t.Logf("⚠️  Device enumeration took longer than expected: %v", elapsed)
	}
	
	t.Log("✅ Device enumeration integration test completed successfully")
}
