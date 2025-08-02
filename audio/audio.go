package audio

import (
	"sync"
)

// Global audio package variables for simple access
var (
	Data     ServerData                  // Main data container
	Process  *AudioHostProcess           // Audio process management
	Mutex    sync.RWMutex                // Global mutex for thread safety
	Reconfig *AudioEngineReconfiguration // Configuration manager
)

// Initialize sets up the audio package
func Initialize() error {
	// Create the configuration manager
	Reconfig = NewAudioEngineReconfiguration()

	// Load initial data
	if err := LoadDevices(); err != nil {
		return err
	}

	if err := LoadPlugins(); err != nil {
		return err
	}

	return nil
}

// Shutdown cleans up audio resources
func Shutdown() error {
	if Process != nil {
		// Stop any running process
		// TODO: implement process stopping
	}
	return nil
}
