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
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unsafe"
)

const DefaultIntrospectionTimeout = 30 * time.Second

// GetAudioUnits performs native AudioUnit introspection with timeout
func GetAudioUnits() (IntrospectionResult, error) {
	return GetAudioUnitsWithTimeout(DefaultIntrospectionTimeout)
}

// GetAudioUnitsWithTimeout performs native AudioUnit introspection with custom timeout
func GetAudioUnitsWithTimeout(timeout time.Duration) (IntrospectionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Channel to receive the result
	resultCh := make(chan IntrospectionResult, 1)
	errorCh := make(chan error, 1)

	// Run introspection in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorCh <- fmt.Errorf("introspection panicked: %v", r)
			}
		}()

		jsonPtr := C.IntrospectAudioUnits()
		if jsonPtr == nil {
			errorCh <- fmt.Errorf("AudioUnit introspection failed")
			return
		}
		defer C.free(unsafe.Pointer(jsonPtr))

		// Convert C string to Go string
		jsonString := C.GoString(jsonPtr)

		// Parse JSON into plugin data
		var plugins []Plugin
		if err := json.Unmarshal([]byte(jsonString), &plugins); err != nil {
			errorCh <- fmt.Errorf("failed to parse introspection JSON: %w", err)
			return
		}

		resultCh <- IntrospectionResult(plugins)
	}()

	// Wait for result or timeout
	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errorCh:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("AudioUnit introspection timed out after %v", timeout)
	}
}

// GetAudioUnitsJSON returns the raw JSON from introspection with timeout
func GetAudioUnitsJSON() (string, error) {
	return GetAudioUnitsJSONWithTimeout(DefaultIntrospectionTimeout)
}

// GetAudioUnitsJSONWithTimeout returns the raw JSON from introspection with custom timeout
func GetAudioUnitsJSONWithTimeout(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Channel to receive the result
	resultCh := make(chan string, 1)
	errorCh := make(chan error, 1)

	// Run introspection in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorCh <- fmt.Errorf("introspection panicked: %v", r)
			}
		}()

		jsonPtr := C.IntrospectAudioUnits()
		if jsonPtr == nil {
			errorCh <- fmt.Errorf("AudioUnit introspection failed")
			return
		}
		defer C.free(unsafe.Pointer(jsonPtr))

		resultCh <- C.GoString(jsonPtr)
	}()

	// Wait for result or timeout
	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errorCh:
		return "", err
	case <-ctx.Done():
		return "", fmt.Errorf("AudioUnit introspection timed out after %v", timeout)
	}
}
