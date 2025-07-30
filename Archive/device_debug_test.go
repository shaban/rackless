package main

import (
	"testing"
)

func TestCoreAudioAccess(t *testing.T) {
	t.Skip("Skipping Core Audio test - devices.go disabled for main package build testing")
}

func TestDeviceStubbed(t *testing.T) {
	// Test that the main package builds without the device enumeration
	t.Log("Testing main package builds without device enumeration")

	if DeviceEnum != nil {
		t.Error("DeviceEnum should be nil when devices.go is disabled")
	}
}
