//go:build darwin && cgo

package devices

/*
#cgo CFLAGS: -I../audio
#cgo LDFLAGS: -L../audio -laudiounit_devices -framework CoreAudio -framework CoreMIDI -framework Foundation

#include "audiounit_devices.h"
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unsafe"
)

// nativeDeviceEnumerator implements DeviceEnumerator using CGO
type nativeDeviceEnumerator struct {
	config DeviceEnumerationConfig
}

// NewDeviceEnumerator creates a new device enumerator with default configuration
func NewDeviceEnumerator() DeviceEnumerator {
	return &nativeDeviceEnumerator{
		config: DefaultConfig(),
	}
}

// NewDeviceEnumeratorWithConfig creates a new device enumerator with custom configuration
func NewDeviceEnumeratorWithConfig(config DeviceEnumerationConfig) DeviceEnumerator {
	return &nativeDeviceEnumerator{
		config: config,
	}
}

// GetAudioInputDevices implements DeviceEnumerator.GetAudioInputDevices
func (de *nativeDeviceEnumerator) GetAudioInputDevices() ([]AudioDevice, error) {
	return de.getAudioInputDevicesWithTimeout(de.config.Timeout)
}

// GetAudioOutputDevices implements DeviceEnumerator.GetAudioOutputDevices
func (de *nativeDeviceEnumerator) GetAudioOutputDevices() ([]AudioDevice, error) {
	return de.getAudioOutputDevicesWithTimeout(de.config.Timeout)
}

// GetMIDIInputDevices implements DeviceEnumerator.GetMIDIInputDevices
func (de *nativeDeviceEnumerator) GetMIDIInputDevices() ([]MIDIDevice, error) {
	return de.getMIDIInputDevicesWithTimeout(de.config.Timeout)
}

// GetMIDIOutputDevices implements DeviceEnumerator.GetMIDIOutputDevices
func (de *nativeDeviceEnumerator) GetMIDIOutputDevices() ([]MIDIDevice, error) {
	return de.getMIDIOutputDevicesWithTimeout(de.config.Timeout)
}

// GetDefaultAudioDevices implements DeviceEnumerator.GetDefaultAudioDevices
func (de *nativeDeviceEnumerator) GetDefaultAudioDevices() (DefaultAudioDevices, error) {
	return de.getDefaultAudioDevicesWithTimeout(de.config.Timeout)
}

// GetAllDevices implements DeviceEnumerator.GetAllDevices
func (de *nativeDeviceEnumerator) GetAllDevices() (DeviceEnumerationResult, error) {
	start := time.Now()
	
	ctx, cancel := context.WithTimeout(context.Background(), de.config.Timeout)
	defer cancel()
	
	// Channel to collect results
	type result struct {
		audioInputs    []AudioDevice
		audioOutputs   []AudioDevice
		midiInputs     []MIDIDevice
		midiOutputs    []MIDIDevice
		defaultDevices DefaultAudioDevices
		err            error
	}
	
	resultChan := make(chan result, 1)
	
	// Run enumeration in goroutine with timeout protection
	go func() {
		var r result
		
		// Get audio input devices
		r.audioInputs, r.err = de.getAudioInputDevicesWithTimeout(de.config.Timeout)
		if r.err != nil {
			resultChan <- r
			return
		}
		
		// Get audio output devices
		r.audioOutputs, r.err = de.getAudioOutputDevicesWithTimeout(de.config.Timeout)
		if r.err != nil {
			resultChan <- r
			return
		}
		
		// Get MIDI input devices
		r.midiInputs, r.err = de.getMIDIInputDevicesWithTimeout(de.config.Timeout)
		if r.err != nil {
			resultChan <- r
			return
		}
		
		// Get MIDI output devices
		r.midiOutputs, r.err = de.getMIDIOutputDevicesWithTimeout(de.config.Timeout)
		if r.err != nil {
			resultChan <- r
			return
		}
		
		// Get default devices
		r.defaultDevices, r.err = de.getDefaultAudioDevicesWithTimeout(de.config.Timeout)
		if r.err != nil {
			resultChan <- r
			return
		}
		
		resultChan <- r
	}()
	
	// Wait for result or timeout
	select {
	case r := <-resultChan:
		if r.err != nil {
			return DeviceEnumerationResult{
				Success:         false,
				Error:           r.err.Error(),
				EnumerationTime: time.Since(start),
			}, r.err
		}
		
		// Add "(None Selected)" options for safe defaults
		audioInputsWithNone := append([]AudioDevice{{
			Name:         "(None Selected)",
			UID:          "none",
			DeviceID:     -1,
			ChannelCount: 0,
			IsDefault:    true,
		}}, r.audioInputs...)
		
		midiInputsWithNone := append([]MIDIDevice{{
			Name:       "(None Selected)",
			UID:        "none",
			EndpointID: -1,
			IsOnline:   true,
		}}, r.midiInputs...)
		
		midiOutputsWithNone := append([]MIDIDevice{{
			Name:       "(None Selected)",
			UID:        "none",
			EndpointID: -1,
			IsOnline:   true,
		}}, r.midiOutputs...)
		
		return DeviceEnumerationResult{
			AudioInputs:     audioInputsWithNone,
			AudioOutputs:    r.audioOutputs,
			MIDIInputs:      midiInputsWithNone,
			MIDIOutputs:     midiOutputsWithNone,
			DefaultDevices:  r.defaultDevices,
			Success:         true,
			EnumerationTime: time.Since(start),
		}, nil
		
	case <-ctx.Done():
		return DeviceEnumerationResult{
			Success:         false,
			Error:           "device enumeration timed out",
			EnumerationTime: time.Since(start),
		}, fmt.Errorf("device enumeration timed out after %v", de.config.Timeout)
	}
}

// Internal timeout-protected methods

func (de *nativeDeviceEnumerator) getAudioInputDevicesWithTimeout(timeout time.Duration) ([]AudioDevice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	resultChan := make(chan []AudioDevice, 1)
	errorChan := make(chan error, 1)
	
	go func() {
		cResult := C.getAudioInputDevices()
		if cResult == nil {
			errorChan <- fmt.Errorf("failed to get audio input devices")
			return
		}
		defer C.free(unsafe.Pointer(cResult))
		
		jsonStr := C.GoString(cResult)
		var devices []AudioDevice
		if err := json.Unmarshal([]byte(jsonStr), &devices); err != nil {
			errorChan <- fmt.Errorf("failed to parse audio input devices JSON: %w", err)
			return
		}
		
		resultChan <- devices
	}()
	
	select {
	case devices := <-resultChan:
		return devices, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("audio input device enumeration timed out after %v", timeout)
	}
}

func (de *nativeDeviceEnumerator) getAudioOutputDevicesWithTimeout(timeout time.Duration) ([]AudioDevice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	resultChan := make(chan []AudioDevice, 1)
	errorChan := make(chan error, 1)
	
	go func() {
		cResult := C.getAudioOutputDevices()
		if cResult == nil {
			errorChan <- fmt.Errorf("failed to get audio output devices")
			return
		}
		defer C.free(unsafe.Pointer(cResult))
		
		jsonStr := C.GoString(cResult)
		var devices []AudioDevice
		if err := json.Unmarshal([]byte(jsonStr), &devices); err != nil {
			errorChan <- fmt.Errorf("failed to parse audio output devices JSON: %w", err)
			return
		}
		
		resultChan <- devices
	}()
	
	select {
	case devices := <-resultChan:
		return devices, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("audio output device enumeration timed out after %v", timeout)
	}
}

func (de *nativeDeviceEnumerator) getMIDIInputDevicesWithTimeout(timeout time.Duration) ([]MIDIDevice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	resultChan := make(chan []MIDIDevice, 1)
	errorChan := make(chan error, 1)
	
	go func() {
		cResult := C.getMIDIInputDevices()
		if cResult == nil {
			errorChan <- fmt.Errorf("failed to get MIDI input devices")
			return
		}
		defer C.free(unsafe.Pointer(cResult))
		
		jsonStr := C.GoString(cResult)
		var devices []MIDIDevice
		if err := json.Unmarshal([]byte(jsonStr), &devices); err != nil {
			errorChan <- fmt.Errorf("failed to parse MIDI input devices JSON: %w", err)
			return
		}
		
		resultChan <- devices
	}()
	
	select {
	case devices := <-resultChan:
		return devices, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("MIDI input device enumeration timed out after %v", timeout)
	}
}

func (de *nativeDeviceEnumerator) getMIDIOutputDevicesWithTimeout(timeout time.Duration) ([]MIDIDevice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	resultChan := make(chan []MIDIDevice, 1)
	errorChan := make(chan error, 1)
	
	go func() {
		cResult := C.getMIDIOutputDevices()
		if cResult == nil {
			errorChan <- fmt.Errorf("failed to get MIDI output devices")
			return
		}
		defer C.free(unsafe.Pointer(cResult))
		
		jsonStr := C.GoString(cResult)
		var devices []MIDIDevice
		if err := json.Unmarshal([]byte(jsonStr), &devices); err != nil {
			errorChan <- fmt.Errorf("failed to parse MIDI output devices JSON: %w", err)
			return
		}
		
		resultChan <- devices
	}()
	
	select {
	case devices := <-resultChan:
		return devices, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("MIDI output device enumeration timed out after %v", timeout)
	}
}

func (de *nativeDeviceEnumerator) getDefaultAudioDevicesWithTimeout(timeout time.Duration) (DefaultAudioDevices, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	resultChan := make(chan DefaultAudioDevices, 1)
	errorChan := make(chan error, 1)
	
	go func() {
		cResult := C.getDefaultAudioDevices()
		if cResult == nil {
			errorChan <- fmt.Errorf("failed to get default audio devices")
			return
		}
		defer C.free(unsafe.Pointer(cResult))
		
		jsonStr := C.GoString(cResult)
		var defaults DefaultAudioDevices
		if err := json.Unmarshal([]byte(jsonStr), &defaults); err != nil {
			errorChan <- fmt.Errorf("failed to parse default audio devices JSON: %w", err)
			return
		}
		
		resultChan <- defaults
	}()
	
	select {
	case defaults := <-resultChan:
		return defaults, nil
	case err := <-errorChan:
		return DefaultAudioDevices{}, err
	case <-ctx.Done():
		return DefaultAudioDevices{}, fmt.Errorf("default audio device enumeration timed out after %v", timeout)
	}
}
