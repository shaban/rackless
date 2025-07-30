//go:build darwin && cgo
// +build darwin,cgo

package introspection

/*
#cgo CFLAGS: -x objective-c -DVERBOSE_LOGGING=0
#cgo LDFLAGS: -L../audio -laudiounit_inspector -framework Foundation -framework AudioToolbox -framework AVFoundation -framework AudioUnit
#include <stdlib.h>
#include "../audio/audiounit_inspector.h"
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// GetAudioUnits performs native AudioUnit introspection
// Relies on 0.5s per-plugin timeout in Objective-C layer for safety
func GetAudioUnits() (IntrospectionResult, error) {
	jsonPtr := C.IntrospectAudioUnits()
	if jsonPtr == nil {
		return nil, fmt.Errorf("AudioUnit introspection failed")
	}
	defer C.free(unsafe.Pointer(jsonPtr))

	// Convert C string to Go string
	jsonString := C.GoString(jsonPtr)

	// Parse JSON into plugin data
	var plugins []Plugin
	if err := json.Unmarshal([]byte(jsonString), &plugins); err != nil {
		return nil, fmt.Errorf("failed to parse introspection JSON: %w", err)
	}

	return IntrospectionResult(plugins), nil
}

// GetAudioUnitsJSON returns the raw JSON from introspection
// Relies on 0.5s per-plugin timeout in Objective-C layer for safety
func GetAudioUnitsJSON() (string, error) {
	jsonPtr := C.IntrospectAudioUnits()
	if jsonPtr == nil {
		return "", fmt.Errorf("AudioUnit introspection failed")
	}
	defer C.free(unsafe.Pointer(jsonPtr))

	return C.GoString(jsonPtr), nil
}
