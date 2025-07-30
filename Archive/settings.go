//go:build darwin && cgo
// +build darwin,cgo

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Settings represents the application configuration
type Settings struct {
	Version      string         `json:"version"`
	Audio        Audio          `json:"audio"`
	Layout       LayoutSettings `json:"layout"`
	UI           UI             `json:"ui"`
	MIDI         MIDI           `json:"midi"`
	Server       ServerCfg      `json:"server"`
	LastModified *time.Time     `json:"lastModified"`
	FirstRun     bool           `json:"firstRun"`
}

type Audio struct {
	InputDeviceID    *string `json:"inputDeviceId"`
	InputDeviceName  string  `json:"inputDeviceName"`
	OutputDeviceID   *string `json:"outputDeviceId"`
	OutputDeviceName string  `json:"outputDeviceName"`
	SampleRate       int     `json:"sampleRate"`
	BufferSize       int     `json:"bufferSize"`
}

type LayoutSettings struct {
	CurrentLayoutName string   `json:"currentLayoutName"`
	CurrentLayoutPath *string  `json:"currentLayoutPath"`
	RecentLayouts     []string `json:"recentLayouts"`
}

type UI struct {
	Theme            string `json:"theme"`
	ShowAdvancedTabs bool   `json:"showAdvancedTabs"`
	LastActiveTab    string `json:"lastActiveTab"`
}

type MIDI struct {
	LearnMode       bool    `json:"learnMode"`
	InputDeviceID   *string `json:"inputDeviceId"`
	InputDeviceName string  `json:"inputDeviceName"`
}

type ServerCfg struct {
	Port      int    `json:"port"`
	AutoStart bool   `json:"autoStart"`
	LogLevel  string `json:"logLevel"`
}

// SettingsManager handles loading, saving, and managing application settings
type SettingsManager struct {
	settings   *Settings
	filePath   string
	mutex      sync.RWMutex
	watchers   []func(*Settings) // Callbacks for settings changes
	deviceEnum *DeviceEnumerator // Added for default device detection
}

// NewSettingsManager creates a new settings manager
func NewSettingsManager(filePath string, deviceEnum *DeviceEnumerator) *SettingsManager {
	return &SettingsManager{
		filePath:   filePath,
		watchers:   make([]func(*Settings), 0),
		deviceEnum: deviceEnum,
	}
}

// Load reads settings from file or creates defaults
func (sm *SettingsManager) Load() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check if settings file exists
	if _, err := os.Stat(sm.filePath); os.IsNotExist(err) {
		log.Println("📄 Settings file not found, creating with defaults...")
		sm.settings = sm.getDefaultSettings()
		return sm.saveInternal()
	}

	// Read existing settings file
	data, err := os.ReadFile(sm.filePath)
	if err != nil {
		return fmt.Errorf("failed to read settings file: %w", err)
	}

	// Parse JSON
	settings := &Settings{}
	if err := json.Unmarshal(data, settings); err != nil {
		log.Printf("⚠️  Settings file corrupted, recreating with defaults: %v", err)
		sm.settings = sm.getDefaultSettings()
		return sm.saveInternal()
	}

	sm.settings = settings

	// Update output device to default on first run after fresh install
	if sm.settings.FirstRun {
		sm.settings.Audio.OutputDeviceName = "Default Audio Device"
		sm.settings.FirstRun = false
		if err := sm.saveInternal(); err != nil {
			log.Printf("Failed to update first run settings: %v", err)
		}
	}

	log.Printf("✅ Settings loaded from %s", sm.filePath)
	return nil
}

// Save persists current settings to file
func (sm *SettingsManager) Save() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	return sm.saveInternal()
}

// saveInternal performs the actual save without locking (internal use)
func (sm *SettingsManager) saveInternal() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(sm.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	// Update last modified timestamp
	now := time.Now()
	sm.settings.LastModified = &now

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(sm.settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write to file
	if err := os.WriteFile(sm.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	// Notify watchers
	for _, watcher := range sm.watchers {
		go watcher(sm.settings)
	}

	return nil
}

// Get returns a copy of current settings (thread-safe)
func (sm *SettingsManager) Get() Settings {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if sm.settings == nil {
		return *sm.getDefaultSettings()
	}

	// Return a copy to prevent external modification
	return *sm.settings
}

// Update modifies settings and saves them
func (sm *SettingsManager) Update(updateFunc func(*Settings)) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.settings == nil {
		sm.settings = sm.getDefaultSettings()
	}

	// Apply updates
	updateFunc(sm.settings)

	// Save changes
	return sm.saveInternal()
}

// UpdateAudioInput sets the audio input device
func (sm *SettingsManager) UpdateAudioInput(deviceID *string, deviceName string) error {
	return sm.Update(func(s *Settings) {
		s.Audio.InputDeviceID = deviceID
		s.Audio.InputDeviceName = deviceName
	})
}

// UpdateAudioOutput sets the audio output device
func (sm *SettingsManager) UpdateAudioOutput(deviceID *string, deviceName string) error {
	return sm.Update(func(s *Settings) {
		s.Audio.OutputDeviceID = deviceID
		s.Audio.OutputDeviceName = deviceName
	})
}

// UpdateCurrentLayout sets the current layout and adds it to recent layouts
func (sm *SettingsManager) UpdateCurrentLayout(layoutName, layoutPath string) error {
	return sm.Update(func(s *Settings) {
		s.Layout.CurrentLayoutName = layoutName
		if layoutPath != "" {
			s.Layout.CurrentLayoutPath = &layoutPath
		}

		// Add to recent layouts (keep max 10)
		recent := make([]string, 0, 10)
		recent = append(recent, layoutName)

		for _, name := range s.Layout.RecentLayouts {
			if name != layoutName && len(recent) < 10 {
				recent = append(recent, name)
			}
		}
		s.Layout.RecentLayouts = recent
	})
}

// UpdateMIDIInput sets the MIDI input device
func (sm *SettingsManager) UpdateMIDIInput(deviceID *string, deviceName string) error {
	return sm.Update(func(s *Settings) {
		s.MIDI.InputDeviceID = deviceID
		s.MIDI.InputDeviceName = deviceName
	})
}

// UpdateUISettings updates UI-related settings
func (sm *SettingsManager) UpdateUISettings(theme, lastActiveTab string, showAdvancedTabs bool) error {
	return sm.Update(func(s *Settings) {
		if theme != "" {
			s.UI.Theme = theme
		}
		if lastActiveTab != "" {
			s.UI.LastActiveTab = lastActiveTab
		}
		s.UI.ShowAdvancedTabs = showAdvancedTabs
	})
}

// AddWatcher registers a callback for settings changes
func (sm *SettingsManager) AddWatcher(callback func(*Settings)) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.watchers = append(sm.watchers, callback)
}

// getDefaultSettings returns the default application settings
func (sm *SettingsManager) getDefaultSettings() *Settings {
	// Detect default audio output device
	var outputDeviceID *string
	outputDeviceName := "Default Audio Device"

	if sm.deviceEnum != nil {
		if defaultDevices, err := sm.deviceEnum.GetDefaultAudioDevices(); err == nil {
			if defaultDevices.DefaultOutput != 0 {
				deviceIDStr := fmt.Sprintf("%d", defaultDevices.DefaultOutput)
				outputDeviceID = &deviceIDStr

				// Try to get the actual device name
				if outputDevices, err := sm.deviceEnum.GetAudioOutputDevices(); err == nil {
					for _, device := range outputDevices {
						if device.DeviceID == int(defaultDevices.DefaultOutput) {
							outputDeviceName = device.Name
							break
						}
					}
				}
			}
		}
	}

	return &Settings{
		Version: "1.0.0",
		Audio: Audio{
			InputDeviceID:    nil,
			InputDeviceName:  "Not Selected",
			OutputDeviceID:   outputDeviceID,
			OutputDeviceName: outputDeviceName,
			SampleRate:       44100,
			BufferSize:       512,
		},
		Layout: LayoutSettings{
			CurrentLayoutName: "Not Selected",
			CurrentLayoutPath: nil,
			RecentLayouts:     make([]string, 0),
		},
		UI: UI{
			Theme:            "dark",
			ShowAdvancedTabs: true,
			LastActiveTab:    "viewer",
		},
		MIDI: MIDI{
			LearnMode:       false,
			InputDeviceID:   nil,
			InputDeviceName: "Not Selected",
		},
		Server: ServerCfg{
			Port:      8080,
			AutoStart: true,
			LogLevel:  "info",
		},
		LastModified: nil,
		FirstRun:     true,
	}
}

// GetCurrentLayoutName returns the current layout name, with fallback logic
func (sm *SettingsManager) GetCurrentLayoutName() string {
	settings := sm.Get()

	// If no layout selected, trigger sample layout use
	if settings.Layout.CurrentLayoutName == "Not Selected" || settings.Layout.CurrentLayoutName == "" {
		return "sample_layout" // This will cause the server to use the sample layout
	}

	return settings.Layout.CurrentLayoutName
}

// IsFirstRun returns true if this is the first time the application is running
func (sm *SettingsManager) IsFirstRun() bool {
	settings := sm.Get()
	return settings.FirstRun
}
