package devices

import (
	"testing"
)

// BenchmarkDeviceEnumeration benchmarks complete device enumeration
func BenchmarkDeviceEnumeration(b *testing.B) {
	enumerator := NewDeviceEnumerator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := enumerator.GetAllDevices()
		if err != nil {
			b.Fatalf("Device enumeration failed: %v", err)
		}
		if !result.Success {
			b.Fatalf("Device enumeration reported failure: %s", result.Error)
		}
	}
}

// BenchmarkAudioInputDevices benchmarks audio input device enumeration
func BenchmarkAudioInputDevices(b *testing.B) {
	enumerator := NewDeviceEnumerator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		devices, err := enumerator.GetAudioInputDevices()
		if err != nil {
			b.Fatalf("Audio input enumeration failed: %v", err)
		}
		if devices == nil {
			b.Fatal("Audio input enumeration returned nil")
		}
	}
}

// BenchmarkAudioOutputDevices benchmarks audio output device enumeration
func BenchmarkAudioOutputDevices(b *testing.B) {
	enumerator := NewDeviceEnumerator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		devices, err := enumerator.GetAudioOutputDevices()
		if err != nil {
			b.Fatalf("Audio output enumeration failed: %v", err)
		}
		if devices == nil {
			b.Fatal("Audio output enumeration returned nil")
		}
	}
}

// BenchmarkMIDIInputDevices benchmarks MIDI input device enumeration
func BenchmarkMIDIInputDevices(b *testing.B) {
	enumerator := NewDeviceEnumerator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		devices, err := enumerator.GetMIDIInputDevices()
		if err != nil {
			b.Fatalf("MIDI input enumeration failed: %v", err)
		}
		if devices == nil {
			b.Fatal("MIDI input enumeration returned nil")
		}
	}
}

// BenchmarkMIDIOutputDevices benchmarks MIDI output device enumeration
func BenchmarkMIDIOutputDevices(b *testing.B) {
	enumerator := NewDeviceEnumerator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		devices, err := enumerator.GetMIDIOutputDevices()
		if err != nil {
			b.Fatalf("MIDI output enumeration failed: %v", err)
		}
		if devices == nil {
			b.Fatal("MIDI output enumeration returned nil")
		}
	}
}

// BenchmarkDefaultDevices benchmarks default device detection
func BenchmarkDefaultDevices(b *testing.B) {
	enumerator := NewDeviceEnumerator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		defaults, err := enumerator.GetDefaultAudioDevices()
		if err != nil {
			b.Fatalf("Default device detection failed: %v", err)
		}
		// Validate that we got some response
		_ = defaults
	}
}

// BenchmarkDeviceEnumerationWithConfig benchmarks enumeration with custom config
func BenchmarkDeviceEnumerationWithConfig(b *testing.B) {
	config := DeviceEnumerationConfig{
		IncludeOfflineDevices: false,
		IncludeVirtualDevices: true,
	}
	enumerator := NewDeviceEnumeratorWithConfig(config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := enumerator.GetAllDevices()
		if err != nil {
			b.Fatalf("Device enumeration with config failed: %v", err)
		}
		if !result.Success {
			b.Fatalf("Device enumeration reported failure: %s", result.Error)
		}
	}
}
