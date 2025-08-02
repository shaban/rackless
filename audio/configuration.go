package audio

import (
	"fmt"
	"log"
)

// NewAudioEngineReconfiguration creates a new reconfiguration manager
func NewAudioEngineReconfiguration() *AudioEngineReconfiguration {
	return &AudioEngineReconfiguration{
		currentConfig: nil,
		isRunning:     false,
	}
}

// AnalyzeConfigChange determines what type of reconfiguration is needed
func (r *AudioEngineReconfiguration) AnalyzeConfigChange(newConfig AudioConfig) ChangeRequirement {
	if r.currentConfig == nil {
		// First time configuration - no reconfiguration needed, just start
		return NoChangeRequired
	}

	// Check for changes that require process restart (complete audio-host restart)
	if r.requiresProcessRestart(*r.currentConfig, newConfig) {
		return ProcessRestartRequired
	}

	// Check for changes that require chain rebuild (stop/reconfigure/start audio unit)
	if r.requiresChainRebuild(*r.currentConfig, newConfig) {
		return ChainRebuildRequired
	}

	// Check if it's a dynamic change (can be done while running)
	if r.isDynamicChange(*r.currentConfig, newConfig) {
		return DynamicChangeOnly
	}

	return NoChangeRequired
}

// requiresProcessRestart checks if changes require complete audio-host process restart
func (r *AudioEngineReconfiguration) requiresProcessRestart(current, new AudioConfig) bool {
	// Core audio parameters that require full process restart
	if current.SampleRate != new.SampleRate {
		log.Printf("🔄 Sample rate change detected: %.0f Hz → %.0f Hz (requires process restart)",
			current.SampleRate, new.SampleRate)
		return true
	}

	if current.BufferSize != new.BufferSize {
		log.Printf("🔄 Buffer size change detected: %d → %d samples (requires process restart)",
			current.BufferSize, new.BufferSize)
		return true
	}

	if current.AudioInputDeviceID != new.AudioInputDeviceID {
		log.Printf("🔄 Input device change detected: %d → %d (requires process restart)",
			current.AudioInputDeviceID, new.AudioInputDeviceID)
		return true
	}

	return false
}

// requiresChainRebuild checks if changes require audio chain reconfiguration
func (r *AudioEngineReconfiguration) requiresChainRebuild(current, new AudioConfig) bool {
	// Input channel changes could potentially be done with chain rebuild
	if current.AudioInputChannel != new.AudioInputChannel {
		log.Printf("🔧 Input channel change detected: %d → %d (could use chain rebuild)",
			current.AudioInputChannel, new.AudioInputChannel)
		return false
	}

	// Plugin path changes could be done with chain rebuild
	if current.PluginPath != new.PluginPath {
		log.Printf("🔧 Plugin path change detected: %s → %s (could use chain rebuild)",
			current.PluginPath, new.PluginPath)
		return false
	}

	return false
}

// isDynamicChange checks if changes can be made without stopping audio
func (r *AudioEngineReconfiguration) isDynamicChange(current, new AudioConfig) bool {
	// Test tone enable/disable can be changed dynamically
	if current.EnableTestTone != new.EnableTestTone {
		log.Printf("🎵 Test tone change detected: %t → %t (dynamic change)",
			current.EnableTestTone, new.EnableTestTone)
		return true
	}

	// Plugin loading/unloading can be done dynamically
	if current.PluginPath != new.PluginPath {
		log.Printf("🔌 Plugin change detected: %s → %s (dynamic change possible)",
			current.PluginPath, new.PluginPath)
		return true
	}

	return false
}

// ApplyConfigChange orchestrates the reconfiguration process
func (r *AudioEngineReconfiguration) ApplyConfigChange(change ConfigChange) (*ReconfigurationResult, error) {
	log.Printf("🎯 Analyzing config change: %s", change.ChangeReason)

	requirement := r.AnalyzeConfigChange(change.NewConfig)
	result := &ReconfigurationResult{
		ChangeType:     requirement,
		PreviousConfig: r.currentConfig,
		NewConfig:      &change.NewConfig,
	}

	switch requirement {
	case NoChangeRequired:
		return r.handleNoChange(result, change)

	case ProcessRestartRequired:
		return r.handleProcessRestart(result, change)

	case ChainRebuildRequired:
		return r.handleChainRebuild(result, change)

	case DynamicChangeOnly:
		return r.handleDynamicChange(result, change)

	default:
		return nil, fmt.Errorf("unknown change requirement: %d", requirement)
	}
}

// handleNoChange processes cases where no reconfiguration is needed
func (r *AudioEngineReconfiguration) handleNoChange(result *ReconfigurationResult, change ConfigChange) (*ReconfigurationResult, error) {
	log.Printf("✅ No configuration change required")

	result.Success = true
	result.Message = "Configuration unchanged"
	result.RequiredRestart = false
	result.ProcessIDChanged = false

	// Update current config if this is first time setup
	if r.currentConfig == nil {
		r.currentConfig = &change.NewConfig
		result.Message = "Initial configuration set"
	}

	return result, nil
}

// handleProcessRestart manages complete audio-host process restart
func (r *AudioEngineReconfiguration) handleProcessRestart(result *ReconfigurationResult, change ConfigChange) (*ReconfigurationResult, error) {
	log.Printf("🔄 Process restart required for configuration change")

	var oldPID int

	// Stop current audio-host if running
	if r.isRunning && Process != nil {
		oldPID = Process.pid
		log.Printf("⏹️ Stopping current audio-host (PID %d)", oldPID)

		if err := Process.Stop(); err != nil {
			result.Success = false
			result.Message = fmt.Sprintf("Failed to stop current audio-host: %v", err)
			return result, err
		}

		// Clear global state
		Mutex.Lock()
		Process = nil
		Mutex.Unlock()
		r.isRunning = false
	}

	// Start new audio-host with new configuration
	log.Printf("🚀 Starting audio-host with new configuration")
	newProcess, err := StartAudioHostProcess(change.NewConfig)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Failed to start audio-host with new configuration: %v", err)
		return result, err
	}

	// Update global and local state
	Mutex.Lock()
	Process = newProcess
	Mutex.Unlock()

	r.currentConfig = &change.NewConfig
	r.isRunning = true

	result.Success = true
	result.Message = "Audio-host restarted successfully with new configuration"
	result.RequiredRestart = true
	result.ProcessIDChanged = true
	result.OldPID = oldPID
	result.NewPID = newProcess.pid

	log.Printf("✅ Process restart completed: PID %d → PID %d", oldPID, newProcess.pid)
	return result, nil
}

// handleChainRebuild manages audio chain reconfiguration without process restart
func (r *AudioEngineReconfiguration) handleChainRebuild(result *ReconfigurationResult, change ConfigChange) (*ReconfigurationResult, error) {
	log.Printf("🔧 Audio chain rebuild required (not yet implemented)")

	result.Success = false
	result.Message = "Chain rebuild not yet implemented - falling back to process restart"

	// For now, fall back to process restart
	return r.handleProcessRestart(result, change)
}

// handleDynamicChange manages changes that can be made while audio is running
func (r *AudioEngineReconfiguration) handleDynamicChange(result *ReconfigurationResult, change ConfigChange) (*ReconfigurationResult, error) {
	log.Printf("🎵 Applying dynamic configuration change")

	if !r.isRunning || Process == nil {
		result.Success = false
		result.Message = "Cannot apply dynamic changes - audio-host not running"
		return result, fmt.Errorf("audio-host not running")
	}

	// Handle test tone changes
	if r.currentConfig.EnableTestTone != change.NewConfig.EnableTestTone {
		command := "tone off"
		if change.NewConfig.EnableTestTone {
			command = "tone on"
		}

		_, err := Process.SendCommand(command)
		if err != nil {
			result.Success = false
			result.Message = fmt.Sprintf("Failed to change test tone: %v", err)
			return result, err
		}
		log.Printf("🎵 Test tone changed: %t → %t", r.currentConfig.EnableTestTone, change.NewConfig.EnableTestTone)
	}

	// Handle plugin changes
	if r.currentConfig.PluginPath != change.NewConfig.PluginPath {
		// Unload current plugin if any
		if r.currentConfig.PluginPath != "" {
			_, err := Process.SendCommand("unload-plugin")
			if err != nil {
				log.Printf("⚠️ Warning: Failed to unload current plugin: %v", err)
			}
		}

		// Load new plugin if specified
		if change.NewConfig.PluginPath != "" {
			command := fmt.Sprintf("load-plugin %s", change.NewConfig.PluginPath)
			_, err := Process.SendCommand(command)
			if err != nil {
				result.Success = false
				result.Message = fmt.Sprintf("Failed to load plugin: %v", err)
				return result, err
			}
			log.Printf("🔌 Plugin changed: %s → %s", r.currentConfig.PluginPath, change.NewConfig.PluginPath)
		}
	}

	// Update current configuration
	r.currentConfig = &change.NewConfig

	result.Success = true
	result.Message = "Dynamic configuration change applied successfully"
	result.RequiredRestart = false
	result.ProcessIDChanged = false

	log.Printf("✅ Dynamic change completed successfully")
	return result, nil
}

// GetCurrentConfig returns the current audio configuration
func (r *AudioEngineReconfiguration) GetCurrentConfig() *AudioConfig {
	return r.currentConfig
}

// IsRunning returns whether the audio engine is currently running
func (r *AudioEngineReconfiguration) IsRunning() bool {
	return r.isRunning
}

// SetRunning updates the running state (should be called when audio starts/stops externally)
func (r *AudioEngineReconfiguration) SetRunning(running bool) {
	r.isRunning = running
	if !running {
		log.Printf("🔇 Audio engine marked as stopped")
	}
}

// SetCurrentConfig updates the current configuration (should be called when audio starts)
func (r *AudioEngineReconfiguration) SetCurrentConfig(config AudioConfig) {
	r.currentConfig = &config
	log.Printf("🎯 Audio configuration updated: %.0f Hz, %d samples, device %d",
		config.SampleRate, config.BufferSize, config.AudioInputDeviceID)
}
