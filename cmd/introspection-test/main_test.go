package main

import (
	"testing"
	"time"

	"github.com/shaban/rackless/pkg/introspection"
)

func TestIntrospectionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test that introspection works within a reasonable time
	start := time.Now()

	plugins, err := introspection.GetAudioUnits()
	if err != nil {
		t.Fatalf("Integration test failed: %v", err)
	}

	elapsed := time.Since(start)
	t.Logf("Introspection completed in %v", elapsed)

	// Should complete within reasonable time (adjust based on your system)
	if elapsed > 45*time.Second {
		t.Errorf("Introspection took too long: %v", elapsed)
	}

	if len(plugins) == 0 {
		t.Error("Integration test: no plugins found")
	}

	// Test that we can find parameters
	totalParams := plugins.GetParameterCount()
	if totalParams == 0 {
		t.Error("Integration test: no parameters found across all plugins")
	}

	t.Logf("Integration test successful: %d plugins, %d parameters",
		len(plugins), totalParams)
}

func TestIntrospectionRepeatable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping repeatability test in short mode")
	}

	// Run introspection twice to ensure it's repeatable
	plugins1, err1 := introspection.GetAudioUnits()
	if err1 != nil {
		t.Fatalf("First introspection failed: %v", err1)
	}

	plugins2, err2 := introspection.GetAudioUnits()
	if err2 != nil {
		t.Fatalf("Second introspection failed: %v", err2)
	}

	if len(plugins1) != len(plugins2) {
		t.Errorf("Introspection not repeatable: first run found %d plugins, second run found %d",
			len(plugins1), len(plugins2))
	}

	t.Logf("Repeatability test passed: consistent results across runs")
}
