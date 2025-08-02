package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/shaban/rackless/audio"
)

// ConfigChangeRequest represents a request to change audio configuration
type ConfigChangeRequest struct {
	Config audio.AudioConfig `json:"config"`
	Reason string      `json:"reason,omitempty"`
}

// ConfigChangeResponse represents the response to a configuration change
type ConfigChangeResponse struct {
	Success          bool                   `json:"success"`
	Message          string                 `json:"message"`
	ChangeType       string                 `json:"changeType"`
	RequiredRestart  bool                   `json:"requiredRestart"`
	ProcessIDChanged bool                   `json:"processIdChanged"`
	OldPID           int                    `json:"oldPid,omitempty"`
	NewPID           int                    `json:"newPid,omitempty"`
	PreviousConfig   *audio.AudioConfig           `json:"previousConfig,omitempty"`
	NewConfig        *audio.AudioConfig           `json:"newConfig,omitempty"`
	Details          *audio.ReconfigurationResult `json:"details,omitempty"`
}

// handleaudio.ConfigChange processes intelligent configuration changes
func handleConfigChange(w http.ResponseWriter, r *http.Request, audioReconfig *audio.AudioEngineReconfiguration) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request ConfigChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Default reason if not provided
	if request.Reason == "" {
		request.Reason = "Configuration change requested"
	}

	log.Printf("🎯 Config change request: %s", request.Reason)

	// Validate the new configuration first
	if err := validateAudioConfig(request.Config); err != nil {
		response := ConfigChangeResponse{
			Success: false,
			Message: fmt.Sprintf("Configuration validation failed: %v", err),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Apply the configuration change through the reconfiguration manager
	change := audio.ConfigChange{
		NewConfig:    request.Config,
		ChangeReason: request.Reason,
	}

	result, err := audioReconfig.ApplyConfigChange(change)
	if err != nil {
		response := ConfigChangeResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to apply configuration change: %v", err),
			Details: result,
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Convert change type to string
	changeTypeStr := changeTypeToString(result.ChangeType)

	response := ConfigChangeResponse{
		Success:          result.Success,
		Message:          result.Message,
		ChangeType:       changeTypeStr,
		RequiredRestart:  result.RequiredRestart,
		ProcessIDChanged: result.ProcessIDChanged,
		OldPID:           result.OldPID,
		NewPID:           result.NewPID,
		PreviousConfig:   result.PreviousConfig,
		NewConfig:        result.NewConfig,
		Details:          result,
	}

	if result.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(response)
}

// validateAudioConfig performs comprehensive validation of audio configuration
func validateAudioConfig(config audio.AudioConfig) error {
	// Buffer size validation
	if config.BufferSize != 0 && (config.BufferSize < 32 || config.BufferSize > 1024) {
		return fmt.Errorf("invalid buffer size: %d (must be 32-1024 samples)", config.BufferSize)
	}

	// Comprehensive sample rate and device validation
	if err := validateSampleRate(config); err != nil {
		return fmt.Errorf("device/sample rate validation failed: %v", err)
	}

	return nil
}

// changeTypeToString converts audio.ChangeRequirement enum to string
func changeTypeToString(changeType audio.ChangeRequirement) string {
	switch changeType {
	case audio.NoChangeRequired:
		return "no-change"
	case audio.ChainRebuildRequired:
		return "chain-rebuild"
	case audio.ProcessRestartRequired:
		return "process-restart"
	case audio.DynamicChangeOnly:
		return "dynamic-change"
	default:
		return "unknown"
	}
}

// handleGetCurrentConfig returns the current audio configuration
func handleGetCurrentConfig(w http.ResponseWriter, r *http.Request, audioReconfig *audio.AudioEngineReconfiguration) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	currentConfig := audioReconfig.GetCurrentConfig()
	isRunning := audioReconfig.IsRunning()

	response := map[string]interface{}{
		"success":       true,
		"isRunning":     isRunning,
		"currentConfig": currentConfig,
	}

	if audio.Process != nil {
		response["processId"] = audio.Process.GetPID()
	}

	json.NewEncoder(w).Encode(response)
}

// Legacy handleStartAudio updated to use reconfiguration manager
func handleStartAudioWithReconfig(w http.ResponseWriter, r *http.Request, audioReconfig *audio.AudioEngineReconfiguration) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request audio.StartAudioRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	config := request.Config

	changeReq := audio.ConfigChange{
		NewConfig:    config,
		ChangeReason: "Start audio with specific configuration",
	}

	result, err := audioReconfig.ApplyConfigChange(changeReq)
	if err != nil {
		response := ConfigChangeResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to apply configuration: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// If successful, update the reconfiguration manager state
	audioReconfig.SetRunning(true)

	response := ConfigChangeResponse{
		Success:          result.Success,
		Message:          result.Message,
		ChangeType:       changeTypeToString(result.ChangeType),
		RequiredRestart:  result.RequiredRestart,
		ProcessIDChanged: result.ProcessIDChanged,
		OldPID:           result.OldPID,
		NewPID:           result.NewPID,
		PreviousConfig:   result.PreviousConfig,
		NewConfig:        result.NewConfig,
		Details:          result,
	}

	json.NewEncoder(w).Encode(response)
}

// Update stop handler to notify reconfiguration manager
func handleStopAudioWithReconfig(w http.ResponseWriter, r *http.Request, audioReconfig *audio.AudioEngineReconfiguration) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	audio.Mutex.Lock()
	process := audio.Process
	audio.Mutex.Unlock()

	if process == nil {
		response := map[string]interface{}{
			"success": false,
			"message": "No audio-host process running",
		}
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Stop the process
	err := process.Stop()
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to stop audio-host: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Clear global state
	audio.Mutex.Lock()
	audio.Process = nil
	audio.Mutex.Unlock()

	// Notify reconfiguration manager
	audioReconfig.SetRunning(false)

	response := map[string]interface{}{
		"success": true,
		"message": "Audio-host stopped successfully",
	}

	json.NewEncoder(w).Encode(response)
}
