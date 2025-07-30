//go:build darwin && cgo
// +build darwin,cgo

package main

/*
#cgo CFLAGS: -x objective-c -DVERBOSE_LOGGING=0
#cgo LDFLAGS: -L. -laudiounit_inspector -lobjc -framework Foundation -framework AudioToolbox -framework AVFoundation -framework AudioUnit

#include <stdlib.h>
#include "audiounit_inspector.h"
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"unsafe"
)

// Plugin represents an AudioUnit plugin with its parameters
type Plugin struct {
	Name           string      `json:"name"`
	ManufacturerID string      `json:"manufacturerID"`
	Type           string      `json:"type"`
	Subtype        string      `json:"subtype"`
	Parameters     []Parameter `json:"parameters"`
}

// Parameter represents a plugin parameter
type Parameter struct {
	Unit          string   `json:"unit"`
	DisplayName   string   `json:"displayName"`
	Address       uint64   `json:"address"`
	MaxValue      float32  `json:"maxValue"`
	Identifier    string   `json:"identifier"`
	MinValue      float32  `json:"minValue"`
	CanRamp       bool     `json:"canRamp"`
	IsWritable    bool     `json:"isWritable"`
	RawFlags      uint32   `json:"rawFlags"`
	DefaultValue  float32  `json:"defaultValue"`
	CurrentValue  float32  `json:"currentValue"`
	IndexedValues []string `json:"indexedValues,omitempty"`
}

// Global introspection data
var IntrospectionData []Plugin

// IntrospectionResult provides query methods for plugin data
type IntrospectionResult []Plugin

// ExecuteIntrospection runs the native AudioUnit introspection
func ExecuteIntrospection() error {
	log.Println("Executing native AudioUnit introspection...")

	// Call the native function
	jsonPtr := C.IntrospectAudioUnits()
	if jsonPtr == nil {
		return fmt.Errorf("introspection returned null")
	}
	defer C.free(unsafe.Pointer(jsonPtr))

	// Convert C string to Go string
	jsonString := C.GoString(jsonPtr)

	// Parse JSON into plugin data
	var plugins []Plugin
	if err := json.Unmarshal([]byte(jsonString), &plugins); err != nil {
		return fmt.Errorf("failed to parse introspection JSON: %w", err)
	}

	// Store globally
	IntrospectionData = plugins
	log.Printf("Successfully introspected %d AudioUnit plugins", len(plugins))

	return nil
}

// GetIntrospectionJSON returns the raw JSON from introspection
func GetIntrospectionJSON() (string, error) {
	jsonPtr := C.IntrospectAudioUnits()
	if jsonPtr == nil {
		return "", fmt.Errorf("introspection returned null")
	}
	defer C.free(unsafe.Pointer(jsonPtr))

	return C.GoString(jsonPtr), nil
}

// SelectBestPluginForLayout finds the best plugin for layout generation
func (result IntrospectionResult) SelectBestPluginForLayout() *Plugin {
	// Prioritize Neural DSP plugins
	for i := range result {
		if result[i].ManufacturerID == "NDSP" && len(result[i].Parameters) > 0 {
			return &result[i]
		}
	}

	// Fall back to any plugin with a good number of parameters
	var bestPlugin *Plugin
	maxParams := 0

	for i := range result {
		if len(result[i].Parameters) > maxParams {
			maxParams = len(result[i].Parameters)
			bestPlugin = &result[i]
		}
	}

	return bestPlugin
}

// FindPluginsByManufacturer finds all plugins by manufacturer ID
func (result IntrospectionResult) FindPluginsByManufacturer(manufacturerID string) []Plugin {
	var plugins []Plugin
	for _, plugin := range result {
		if plugin.ManufacturerID == manufacturerID {
			plugins = append(plugins, plugin)
		}
	}
	return plugins
}

// GetBooleanParameters returns parameters that appear to be boolean switches
func (p *Plugin) GetBooleanParameters() []Parameter {
	var boolParams []Parameter
	for _, param := range p.Parameters {
		if param.MinValue == 0 && param.MaxValue == 1 && param.Unit == "Boolean" {
			boolParams = append(boolParams, param)
		}
	}
	return boolParams
}

// GetIndexedParameters returns parameters that have discrete indexed values
func (p *Plugin) GetIndexedParameters() []Parameter {
	var indexedParams []Parameter
	for _, param := range p.Parameters {
		if len(param.IndexedValues) > 0 {
			indexedParams = append(indexedParams, param)
		}
	}
	return indexedParams
}

// GetGenericParameters returns parameters that are continuous/generic
func (p *Plugin) GetGenericParameters() []Parameter {
	var genericParams []Parameter
	for _, param := range p.Parameters {
		if len(param.IndexedValues) == 0 && !(param.MinValue == 0 && param.MaxValue == 1 && param.Unit == "Boolean") {
			genericParams = append(genericParams, param)
		}
	}
	return genericParams
}

// GetParameterByAddress finds a parameter by its address
func (p *Plugin) GetParameterByAddress(address uint64) *Parameter {
	for i := range p.Parameters {
		if p.Parameters[i].Address == address {
			return &p.Parameters[i]
		}
	}
	return nil
}

// GetParameterByName finds a parameter by its display name
func (p *Plugin) GetParameterByName(name string) *Parameter {
	for i := range p.Parameters {
		if p.Parameters[i].DisplayName == name {
			return &p.Parameters[i]
		}
	}
	return nil
}

// GetParametersByManufacturerID returns parameters for a specific plugin by manufacturer ID
func (result IntrospectionResult) GetParametersByManufacturerID(manufacturerID string) ([]Parameter, error) {
	for _, plugin := range result {
		if plugin.ManufacturerID == manufacturerID {
			return plugin.Parameters, nil
		}
	}
	return nil, fmt.Errorf("plugin with ManufacturerID '%s' not found", manufacturerID)
}
