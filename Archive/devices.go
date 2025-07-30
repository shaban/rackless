//go:build darwin && cgo
// +build darwin,cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework CoreAudio -framework AudioToolbox -framework CoreMIDI -framework AVFoundation
#include "audiounit_devices.h"
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"unsafe"
)

// Device structures matching the C definitions
type AudioDevice struct {
	Name                 string    `json:"name"`
	UID                  string    `json:"uid"`
	DeviceID             int       `json:"deviceId"`
	ChannelCount         int       `json:"channelCount"`
	SupportedSampleRates []float64 `json:"supportedSampleRates"`
	SupportedBitDepths   []int     `json:"supportedBitDepths"`
	IsDefault            bool      `json:"isDefault"`
}

type MIDIDevice struct {
	Name       string `json:"name"`
	UID        string `json:"uid"`
	EndpointID int    `json:"endpointId"`
	IsOnline   bool   `json:"isOnline"`
}

type DefaultAudioDevices struct {
	DefaultInput  int `json:"defaultInput,omitempty"`
	DefaultOutput int `json:"defaultOutput,omitempty"`
}

// DeviceEnumerator provides device discovery functionality
type DeviceEnumerator struct{}

// NewDeviceEnumerator creates a new device enumerator
func NewDeviceEnumerator() *DeviceEnumerator {
	return &DeviceEnumerator{}
}

// GetAudioInputDevices returns all available audio input devices
func (de *DeviceEnumerator) GetAudioInputDevices() ([]AudioDevice, error) {
	cResult := C.getAudioInputDevices()
	if cResult == nil {
		return nil, fmt.Errorf("failed to get audio input devices")
	}
	defer C.free(unsafe.Pointer(cResult))

	jsonStr := C.GoString(cResult)
	var devices []AudioDevice
	if err := json.Unmarshal([]byte(jsonStr), &devices); err != nil {
		return nil, fmt.Errorf("failed to parse audio input devices JSON: %w", err)
	}

	return devices, nil
}

// GetAudioOutputDevices returns all available audio output devices
func (de *DeviceEnumerator) GetAudioOutputDevices() ([]AudioDevice, error) {
	cResult := C.getAudioOutputDevices()
	if cResult == nil {
		return nil, fmt.Errorf("failed to get audio output devices")
	}
	defer C.free(unsafe.Pointer(cResult))

	jsonStr := C.GoString(cResult)
	var devices []AudioDevice
	if err := json.Unmarshal([]byte(jsonStr), &devices); err != nil {
		return nil, fmt.Errorf("failed to parse audio output devices JSON: %w", err)
	}

	return devices, nil
}

// GetDefaultAudioDevices returns the system default audio devices
func (de *DeviceEnumerator) GetDefaultAudioDevices() (*DefaultAudioDevices, error) {
	cResult := C.getDefaultAudioDevices()
	if cResult == nil {
		return nil, fmt.Errorf("failed to get default audio devices")
	}
	defer C.free(unsafe.Pointer(cResult))

	jsonStr := C.GoString(cResult)
	var defaults DefaultAudioDevices
	if err := json.Unmarshal([]byte(jsonStr), &defaults); err != nil {
		return nil, fmt.Errorf("failed to parse default audio devices JSON: %w", err)
	}

	return &defaults, nil
}

// GetMIDIInputDevices returns all available MIDI input devices
func (de *DeviceEnumerator) GetMIDIInputDevices() ([]MIDIDevice, error) {
	cResult := C.getMIDIInputDevices()
	if cResult == nil {
		return nil, fmt.Errorf("failed to get MIDI input devices")
	}
	defer C.free(unsafe.Pointer(cResult))

	jsonStr := C.GoString(cResult)
	var devices []MIDIDevice
	if err := json.Unmarshal([]byte(jsonStr), &devices); err != nil {
		return nil, fmt.Errorf("failed to parse MIDI input devices JSON: %w", err)
	}

	return devices, nil
}

// GetMIDIOutputDevices returns all available MIDI output devices
func (de *DeviceEnumerator) GetMIDIOutputDevices() ([]MIDIDevice, error) {
	cResult := C.getMIDIOutputDevices()
	if cResult == nil {
		return nil, fmt.Errorf("failed to get MIDI output devices")
	}
	defer C.free(unsafe.Pointer(cResult))

	jsonStr := C.GoString(cResult)
	var devices []MIDIDevice
	if err := json.Unmarshal([]byte(jsonStr), &devices); err != nil {
		return nil, fmt.Errorf("failed to parse MIDI output devices JSON: %w", err)
	}

	return devices, nil
}

// GetAllDevices returns a comprehensive list of all available devices
func (de *DeviceEnumerator) GetAllDevices() (map[string]interface{}, error) {
	audioInputs, err := de.GetAudioInputDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to get audio input devices: %w", err)
	}

	audioOutputs, err := de.GetAudioOutputDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to get audio output devices: %w", err)
	}

	midiInputs, err := de.GetMIDIInputDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to get MIDI input devices: %w", err)
	}

	midiOutputs, err := de.GetMIDIOutputDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to get MIDI output devices: %w", err)
	}

	defaults, err := de.GetDefaultAudioDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to get default audio devices: %w", err)
	}

	// Add "(None Selected)" options for safe defaults
	audioInputsWithNone := append([]AudioDevice{{
		Name:         "(None Selected)",
		UID:          "none",
		DeviceID:     -1,
		ChannelCount: 0,
		IsDefault:    true, // This will be our safe default
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

	return map[string]interface{}{
		"audioInputs":    audioInputsWithNone,
		"audioOutputs":   audioOutputs,
		"midiInputs":     midiInputsWithNone,
		"midiOutputs":    midiOutputsWithNone,
		"defaultDevices": defaults,
	}, nil
}

// Global device enumerator instance
var DeviceEnum *DeviceEnumerator

// Initialize device enumeration
func init() {
	DeviceEnum = NewDeviceEnumerator()
	log.Println("🎛️  Device enumerator initialized")
}
