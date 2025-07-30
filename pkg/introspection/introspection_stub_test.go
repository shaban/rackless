//go:build !darwin || !cgo
// +build !darwin !cgo

package introspection

import (
	"testing"
)

func TestGetAudioUnitsStub(t *testing.T) {
	plugins, err := GetAudioUnits()
	if err != nil {
		t.Fatalf("GetAudioUnits() stub failed: %v", err)
	}

	if len(plugins) == 0 {
		t.Error("Stub should return at least one mock plugin")
	}

	// Validate mock plugin structure
	if len(plugins) > 0 {
		plugin := plugins[0]

		if plugin.Name != "Mock AudioUnit" {
			t.Errorf("Expected mock plugin name 'Mock AudioUnit', got '%s'", plugin.Name)
		}

		if plugin.ManufacturerID != "MOCK" {
			t.Errorf("Expected mock manufacturer 'MOCK', got '%s'", plugin.ManufacturerID)
		}

		if len(plugin.Parameters) == 0 {
			t.Error("Mock plugin should have at least one parameter")
		}
	}

	t.Logf("Stub test passed with %d mock plugins", len(plugins))
}

func TestGetAudioUnitsJSONStub(t *testing.T) {
	jsonData, err := GetAudioUnitsJSON()
	if err != nil {
		t.Fatalf("GetAudioUnitsJSON() stub failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Stub should return non-empty JSON")
	}

	t.Logf("Stub JSON length: %d bytes", len(jsonData))
}
