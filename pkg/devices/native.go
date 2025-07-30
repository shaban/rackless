//go:build darwin && cgo

package devices

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework CoreAudio -framework AudioToolbox -framework CoreMIDI
#include <stdlib.h>
#include "device_enumerator.h"
*/
import "C"

import (
	"encoding/json"
	"time"
	"unsafe"
)

// nativeDeviceEnumerator provides the Darwin/macOS implementation
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
	cResult := C.enumerateAudioInputDevices()
	defer C.free(unsafe.Pointer(cResult))
	
	result := C.GoString(cResult)
	
	var devices []AudioDevice
	if err := json.Unmarshal([]byte(result), &devices); err != nil {
		return nil, err
	}
	
	return devices, nil
}

// GetAudioOutputDevices implements DeviceEnumerator.GetAudioOutputDevices
func (de *nativeDeviceEnumerator) GetAudioOutputDevices() ([]AudioDevice, error) {
	cResult := C.enumerateAudioOutputDevices()
	defer C.free(unsafe.Pointer(cResult))
	
	result := C.GoString(cResult)
	
	var devices []AudioDevice
	if err := json.Unmarshal([]byte(result), &devices); err != nil {
		return nil, err
	}
	
	return devices, nil
}

// GetMIDIInputDevices implements DeviceEnumerator.GetMIDIInputDevices
func (de *nativeDeviceEnumerator) GetMIDIInputDevices() ([]MIDIDevice, error) {
	cResult := C.enumerateMIDIInputDevices()
	defer C.free(unsafe.Pointer(cResult))
	
	result := C.GoString(cResult)
	
	var devices []MIDIDevice
	if err := json.Unmarshal([]byte(result), &devices); err != nil {
		return nil, err
	}
	
	return devices, nil
}

// GetMIDIOutputDevices implements DeviceEnumerator.GetMIDIOutputDevices
func (de *nativeDeviceEnumerator) GetMIDIOutputDevices() ([]MIDIDevice, error) {
	cResult := C.enumerateMIDIOutputDevices()
	defer C.free(unsafe.Pointer(cResult))
	
	result := C.GoString(cResult)
	
	var devices []MIDIDevice
	if err := json.Unmarshal([]byte(result), &devices); err != nil {
		return nil, err
	}
	
	return devices, nil
}

// GetDefaultAudioDevices implements DeviceEnumerator.GetDefaultAudioDevices
func (de *nativeDeviceEnumerator) GetDefaultAudioDevices() (DefaultAudioDevices, error) {
	cResult := C.getDefaultAudioDevices()
	defer C.free(unsafe.Pointer(cResult))
	
	result := C.GoString(cResult)
	
	var devices DefaultAudioDevices
	if err := json.Unmarshal([]byte(result), &devices); err != nil {
		return DefaultAudioDevices{}, err
	}
	
	return devices, nil
}

// GetAllDevices implements DeviceEnumerator.GetAllDevices
func (de *nativeDeviceEnumerator) GetAllDevices() (DeviceEnumerationResult, error) {
	start := time.Now()
	
	cResult := C.enumerateAllDevices()
	defer C.free(unsafe.Pointer(cResult))
	
	result := C.GoString(cResult)
	
	var deviceResult DeviceEnumerationResult
	if err := json.Unmarshal([]byte(result), &deviceResult); err != nil {
		return DeviceEnumerationResult{
			Success:         false,
			Error:           err.Error(),
			EnumerationTime: time.Since(start),
		}, err
	}
	
	deviceResult.EnumerationTime = time.Since(start)
	deviceResult.Success = true
	
	return deviceResult, nil
}
