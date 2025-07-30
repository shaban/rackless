//go:build darwin && cgo
// +build darwin,cgo

package introspection

import (
	"testing"
)

func TestGetAudioUnits(t *testing.T) {
	plugins, err := GetAudioUnits()
	if err != nil {
		t.Fatalf("GetAudioUnits() failed: %v", err)
	}

	if len(plugins) == 0 {
		t.Error("Expected at least one AudioUnit plugin, got none")
	}

	t.Logf("Found %d AudioUnit plugins", len(plugins))

	// Validate structure of first plugin
	if len(plugins) > 0 {
		plugin := plugins[0]

		if plugin.Name == "" {
			t.Error("First plugin has empty name")
		}

		if plugin.ManufacturerID == "" {
			t.Error("First plugin has empty manufacturer ID")
		}

		t.Logf("First plugin: %s (%s) with %d parameters",
			plugin.Name, plugin.ManufacturerID, len(plugin.Parameters))
	}
}

func TestGetAudioUnitsJSON(t *testing.T) {
	jsonData, err := GetAudioUnitsJSON()
	if err != nil {
		t.Fatalf("GetAudioUnitsJSON() failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Expected non-empty JSON data")
	}

	// Basic JSON validation - should start with [
	if jsonData[0] != '[' {
		t.Error("JSON data should start with '[' for array")
	}

	t.Logf("JSON data length: %d bytes", len(jsonData))
}

func TestIntrospectionResultMethods(t *testing.T) {
	plugins, err := GetAudioUnits()
	if err != nil {
		t.Fatalf("GetAudioUnits() failed: %v", err)
	}

	// Test GetParameterCount
	paramCount := plugins.GetParameterCount()
	if paramCount < 0 {
		t.Error("Parameter count should not be negative")
	}
	t.Logf("Total parameter count: %d", paramCount)

	// Test SelectBestPluginForLayout
	bestPlugin := plugins.SelectBestPluginForLayout()
	if bestPlugin != nil {
		if bestPlugin.Name == "" {
			t.Error("Best plugin should have a name")
		}
		t.Logf("Best plugin for layout: %s with %d parameters",
			bestPlugin.Name, len(bestPlugin.Parameters))
	}

	// Test FindPluginByName (if we have plugins)
	if len(plugins) > 0 {
		firstPlugin := plugins[0]
		foundPlugin := plugins.FindPluginByName(firstPlugin.Name)
		if foundPlugin == nil {
			t.Errorf("Should find plugin by name: %s", firstPlugin.Name)
		}

		// Test non-existent plugin
		notFound := plugins.FindPluginByName("NonExistentPlugin12345")
		if notFound != nil {
			t.Error("Should not find non-existent plugin")
		}
	}
}
