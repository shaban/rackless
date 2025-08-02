package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/shaban/rackless/audio"
	"github.com/shaban/rackless/internal/debug"
)

// ConfigChangeRequest represents a request to change audio configuration
type ConfigChangeRequest struct {
	Config audio.AudioConfig `json:"config"`
	Reason string            `json:"reason,omitempty"`
}

// ConfigChangeResponse represents the response to a configuration change
type ConfigChangeResponse struct {
	Success          bool                         `json:"success"`
	Message          string                       `json:"message"`
	ChangeType       string                       `json:"changeType"`
	RequiredRestart  bool                         `json:"requiredRestart"`
	ProcessIDChanged bool                         `json:"processIdChanged"`
	OldPID           int                          `json:"oldPid,omitempty"`
	NewPID           int                          `json:"newPid,omitempty"`
	PreviousConfig   *audio.AudioConfig           `json:"previousConfig,omitempty"`
	NewConfig        *audio.AudioConfig           `json:"newConfig,omitempty"`
	Details          *audio.ReconfigurationResult `json:"details,omitempty"`
}

// Sample rate validation functions
func validateSampleRate(config audio.AudioConfig) error {
	sampleRate := int(config.SampleRate)

	// Check output device sample rate compatibility
	for _, device := range audio.Data.Devices.AudioOutput {
		if device.IsDefault {
			// Check if default output device is online
			if !device.IsOnline {
				return fmt.Errorf("default output device %d (%s) is not online/available",
					device.DeviceID, device.Name)
			}

			supported := false
			for _, supportedRate := range device.SupportedSampleRates {
				if supportedRate == sampleRate {
					supported = true
					break
				}
			}
			if !supported {
				return fmt.Errorf("output device %d (%s) does not support %d Hz. Supported rates: %v",
					device.DeviceID, device.Name, sampleRate, device.SupportedSampleRates)
			}
			break
		}
	}

	// Check input device sample rate compatibility if specified
	if config.AudioInputDeviceID != 0 {
		found := false
		for _, device := range audio.Data.Devices.AudioInput {
			if device.DeviceID == config.AudioInputDeviceID {
				found = true

				// Check if device is online
				if !device.IsOnline {
					return fmt.Errorf("input device %d (%s) is not online/available",
						device.DeviceID, device.Name)
				}

				supported := false
				for _, supportedRate := range device.SupportedSampleRates {
					if supportedRate == sampleRate {
						supported = true
						break
					}
				}
				if !supported {
					return fmt.Errorf("input device %d (%s) does not support %d Hz. Supported rates: %v",
						device.DeviceID, device.Name, sampleRate, device.SupportedSampleRates)
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("input device %d not found", config.AudioInputDeviceID)
		}
	}

	return nil
}

func findCompatibleSampleRate(inputDeviceID, outputDeviceID int) (int, error) {
	var inputSupportedRates []int
	var outputSupportedRates []int

	// Get input device supported rates
	if inputDeviceID != 0 {
		for _, device := range audio.Data.Devices.AudioInput {
			if device.DeviceID == inputDeviceID {
				inputSupportedRates = device.SupportedSampleRates
				break
			}
		}
		if inputSupportedRates == nil {
			return 0, fmt.Errorf("input device %d not found", inputDeviceID)
		}
	}

	// Get output device supported rates (use default if not specified)
	if outputDeviceID != 0 {
		for _, device := range audio.Data.Devices.AudioOutput {
			if device.DeviceID == outputDeviceID {
				outputSupportedRates = device.SupportedSampleRates
				break
			}
		}
	} else {
		// Use default output device
		for _, device := range audio.Data.Devices.AudioOutput {
			if device.IsDefault {
				outputSupportedRates = device.SupportedSampleRates
				break
			}
		}
	}

	if outputSupportedRates == nil {
		return 0, fmt.Errorf("output device not found")
	}

	// Find common sample rates
	var commonRates []int
	for _, outputRate := range outputSupportedRates {
		if inputDeviceID == 0 {
			// No input device, any output rate is fine
			commonRates = append(commonRates, outputRate)
		} else {
			// Check if input device supports this rate
			for _, inputRate := range inputSupportedRates {
				if inputRate == outputRate {
					commonRates = append(commonRates, outputRate)
					break
				}
			}
		}
	}

	if len(commonRates) == 0 {
		return 0, fmt.Errorf("no compatible sample rates found between devices")
	}

	// Prefer standard rates in order: 44100, 48000, 96000, 192000
	preferredRates := []int{44100, 48000, 96000, 192000}
	for _, preferred := range preferredRates {
		for _, common := range commonRates {
			if common == preferred {
				return preferred, nil
			}
		}
	}

	// If no preferred rate found, return the first common rate
	return commonRates[0], nil
}

// Device testing function for simplified boolean approach
func testDeviceConfiguration(config audio.AudioConfig) (bool, string, string) {
	// Step 1: Validate configuration parameters
	if err := validateSampleRate(config); err != nil {
		return false,
			fmt.Sprintf("Device configuration invalid: %v", err),
			"Please select compatible audio devices and sample rate"
	}

	// Step 2: Try to actually start audio-host with these parameters
	// This is the real test - can we initialize the audio system?
	tempProcess, err := audio.StartAudioHostProcess(config)
	if err != nil {
		return false,
			fmt.Sprintf("Audio initialization failed: %v", err),
			"Try different devices or check if audio devices are in use by other applications"
	}

	// Step 3: Audio-host started successfully, clean up immediately
	tempProcess.Stop()

	return true, "", ""
}

// Device switching function - stops current audio-host and starts new one
func switchAudioDevices(config audio.AudioConfig) (bool, string, string, bool, int) {
	// Step 1: Check if audio-host is currently running
	audio.Mutex.RLock()
	wasRunning := audio.Process != nil && audio.Process.IsRunning()
	currentProcess := audio.Process
	audio.Mutex.RUnlock()

	// Step 2: Stop current audio-host if running
	if wasRunning {
		log.Printf("🔄 Stopping current audio-host to switch devices...")
		audio.Mutex.Lock()
		audio.Process = nil
		audio.Mutex.Unlock()

		err := currentProcess.Stop()
		if err != nil {
			return false,
				fmt.Sprintf("Failed to stop current audio-host: %v", err),
				"Try manually stopping audio processes or restart the server",
				wasRunning, 0
		}
		log.Printf("✅ Current audio-host stopped successfully")
	}

	// Step 3: Validate new configuration
	if err := validateSampleRate(config); err != nil {
		return false,
			fmt.Sprintf("New device configuration invalid: %v", err),
			"Please select compatible audio devices and sample rate",
			wasRunning, 0
	}

	// Step 4: Start audio-host with new configuration
	log.Printf("🚀 Starting audio-host with new device configuration...")
	newProcess, err := audio.StartAudioHostProcess(config)
	if err != nil {
		return false,
			fmt.Sprintf("Failed to start audio-host with new devices: %v", err),
			"Check if new devices are available and not in use by other applications",
			wasRunning, 0
	}

	// Step 5: Store the new process
	audio.Mutex.Lock()
	audio.Process = newProcess
	audio.Mutex.Unlock()

	// Update reconfiguration system
	audio.Reconfig.SetCurrentConfig(config)
	audio.Reconfig.SetRunning(true)

	log.Printf("✅ Audio devices switched successfully - new PID %d", newProcess.GetPID())
	return true, "", "", wasRunning, newProcess.GetPID()
}

// API Handlers
func handleDevices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // For WASM development

	if err := json.NewEncoder(w).Encode(audio.Data.Devices); err != nil {
		http.Error(w, "Failed to encode devices data", http.StatusInternalServerError)
		return
	}
}

func handlePlugins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // For WASM development

	if err := json.NewEncoder(w).Encode(audio.Data.Plugins); err != nil {
		http.Error(w, "Failed to encode plugins data", http.StatusInternalServerError)
		return
	}
}

func handlePlugin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Extract plugin ID from path: /api/plugins/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/plugins/")
	pluginID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid plugin ID", http.StatusBadRequest)
		return
	}

	if pluginID < 0 || pluginID >= len(audio.Data.Plugins) {
		http.Error(w, "Plugin not found", http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(audio.Data.Plugins[pluginID]); err != nil {
		http.Error(w, "Failed to encode plugin data", http.StatusInternalServerError)
		return
	}
}

func handleServerData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // For WASM development

	if err := json.NewEncoder(w).Encode(audio.Data); err != nil {
		http.Error(w, "Failed to encode server data", http.StatusInternalServerError)
		return
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	health := map[string]interface{}{
		"status":    "healthy",
		"devices":   len(audio.Data.Devices.AudioInput) + len(audio.Data.Devices.AudioOutput),
		"plugins":   len(audio.Data.Plugins),
		"timestamp": audio.Data.Devices.Timestamp,
	}

	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, "Failed to encode health data", http.StatusInternalServerError)
		return
	}
}

func handleStartAudio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if audio-host is already running
	audio.Mutex.RLock()
	if audio.Process != nil && audio.Process.IsRunning() {
		audio.Mutex.RUnlock()
		response := audio.StartAudioResponse{
			Success: false,
			Message: fmt.Sprintf("Audio-host process is already running (PID %d)", audio.Process.GetPID()),
		}
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}
	audio.Mutex.RUnlock()

	var request audio.StartAudioRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	config := request.Config
	log.Printf("🎯 Starting audio with config: sample rate %.0f Hz, input device %d, buffer size %d",
		config.SampleRate, config.AudioInputDeviceID, config.BufferSize)

	// Validate buffer size (professional audio range: 32-1024 samples)
	if config.BufferSize != 0 && (config.BufferSize < 32 || config.BufferSize > 1024) {
		log.Printf("❌ Invalid buffer size: %d (must be 32-1024 samples)", config.BufferSize)
		response := audio.StartAudioResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid buffer size: %d (must be 32-1024 samples)", config.BufferSize),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Set default buffer size if not specified (256 is good balance of latency vs stability)
	if config.BufferSize == 0 {
		config.BufferSize = 256
		log.Printf("🔧 Using default buffer size: %d samples", config.BufferSize)
	}

	// Validate sample rate compatibility
	if err := validateSampleRate(config); err != nil {
		log.Printf("❌ Sample rate validation failed: %v", err)
		response := audio.StartAudioResponse{
			Success: false,
			Message: fmt.Sprintf("Sample rate validation failed: %v", err),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Start the audio-host process
	process, err := audio.StartAudioHostProcess(config)
	if err != nil {
		log.Printf("❌ Failed to start audio-host: %v", err)
		response := audio.StartAudioResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start audio-host: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Store the process globally
	audio.Mutex.Lock()
	audio.Process = process
	audio.Mutex.Unlock()

	// Update the reconfiguration system with the current configuration
	audio.Reconfig.SetCurrentConfig(config)
	audio.Reconfig.SetRunning(true)

	response := audio.StartAudioResponse{
		Success: true,
		Message: "Audio-host process started successfully with bidirectional communication",
		PID:     process.GetPID(),
	}

	json.NewEncoder(w).Encode(response)
}

func handleStopAudio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	audio.Mutex.Lock()
	process := audio.Process
	audio.Process = nil
	audio.Mutex.Unlock()

	if process == nil || !process.IsRunning() {
		response := map[string]interface{}{
			"success": false,
			"message": "No audio-host process is running",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Stop the process
	err := process.Stop()
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to stop audio-host process: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Audio-host process stopped successfully",
	}

	// Update the reconfiguration system to reflect stopped state
	audio.Reconfig.SetRunning(false)

	json.NewEncoder(w).Encode(response)
}

func handleAudioCommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request audio.AudioCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	audio.Mutex.RLock()
	process := audio.Process
	audio.Mutex.RUnlock()

	if process == nil || !process.IsRunning() {
		response := audio.AudioCommandResponse{
			Success: false,
			Error:   "No audio-host process is running",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("🎯 Sending command to audio-host: %s", request.Command)

	// Send command to audio-host
	output, err := process.SendCommand(request.Command)
	if err != nil {
		log.Printf("❌ Command failed: %v", err)
		response := audio.AudioCommandResponse{
			Success: false,
			Error:   fmt.Sprintf("Command failed: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("✅ Command response: %s", output)

	response := audio.AudioCommandResponse{
		Success: true,
		Output:  output,
	}

	json.NewEncoder(w).Encode(response)
}

func handleAudioStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	audio.Mutex.RLock()
	process := audio.Process
	audio.Mutex.RUnlock()

	status := map[string]interface{}{
		"processRunning": false,
		"engineRunning":  false,
		"pid":            nil,
	}

	if process != nil && process.IsRunning() {
		status["processRunning"] = true
		status["pid"] = process.GetPID()

		// Try to get detailed status from audio-host
		output, err := process.SendCommand("status")
		if err == nil {
			status["details"] = output

			// Parse engine running state from audio-host status
			if strings.Contains(output, "running=true") {
				status["engineRunning"] = true
			}
		}
	}

	json.NewEncoder(w).Encode(status)
}

func handleSuggestSampleRate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get query parameters
	inputDeviceIDStr := r.URL.Query().Get("inputDevice")
	outputDeviceIDStr := r.URL.Query().Get("outputDevice")

	var inputDeviceID, outputDeviceID int
	var err error

	if inputDeviceIDStr != "" {
		inputDeviceID, err = strconv.Atoi(inputDeviceIDStr)
		if err != nil {
			http.Error(w, "Invalid input device ID", http.StatusBadRequest)
			return
		}
	}

	if outputDeviceIDStr != "" {
		outputDeviceID, err = strconv.Atoi(outputDeviceIDStr)
		if err != nil {
			http.Error(w, "Invalid output device ID", http.StatusBadRequest)
			return
		}
	}

	// Find compatible sample rate
	sampleRate, err := findCompatibleSampleRate(inputDeviceID, outputDeviceID)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success":    true,
		"sampleRate": sampleRate,
		"message":    fmt.Sprintf("Recommended sample rate: %d Hz", sampleRate),
	}

	json.NewEncoder(w).Encode(response)
}

func handleTestDevices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request audio.DeviceTestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Build audio.AudioConfig from test request
	config := audio.AudioConfig{
		SampleRate:         request.SampleRate,
		AudioInputDeviceID: request.InputDeviceID,
		BufferSize:         request.BufferSize,
	}

	// Set default buffer size if not specified
	if config.BufferSize == 0 {
		config.BufferSize = 256
	}

	// Use default output device if not specified
	if request.OutputDeviceID != 0 {
		// Note: Current audio-host doesn't support output device selection
		// but we can validate it exists
		found := false
		for _, device := range audio.Data.Devices.AudioOutput {
			if device.DeviceID == request.OutputDeviceID {
				found = true
				break
			}
		}
		if !found {
			response := audio.DeviceTestResponse{
				IsAudioReady:   false,
				ErrorMessage:   fmt.Sprintf("Output device %d not found", request.OutputDeviceID),
				RequiredAction: "Select a valid audio output device",
				TestedConfig:   config,
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	log.Printf("🧪 Testing device configuration: input %d, sample rate %.0f Hz, buffer %d",
		config.AudioInputDeviceID, config.SampleRate, config.BufferSize)

	// Test the configuration
	isReady, errorMsg, action := testDeviceConfiguration(config)

	response := audio.DeviceTestResponse{
		IsAudioReady:   isReady,
		ErrorMessage:   errorMsg,
		RequiredAction: action,
		TestedConfig:   config,
	}

	if isReady {
		log.Printf("✅ Device test successful - audio ready")
	} else {
		log.Printf("❌ Device test failed: %s", errorMsg)
	}

	json.NewEncoder(w).Encode(response)
}

func handleSwitchDevices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request audio.DeviceSwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Build audio.AudioConfig from switch request
	config := audio.AudioConfig{
		SampleRate:         request.SampleRate,
		AudioInputDeviceID: request.InputDeviceID,
		AudioInputChannel:  0, // Default to channel 0
		BufferSize:         request.BufferSize,
		EnableTestTone:     false, // Default to no test tone when switching devices
	}

	// Set default buffer size if not specified
	if config.BufferSize == 0 {
		config.BufferSize = 256
	}

	// Validate output device if specified
	if request.OutputDeviceID != 0 {
		// Note: Current audio-host doesn't support output device selection
		// but we can validate it exists for future compatibility
		found := false
		for _, device := range audio.Data.Devices.AudioOutput {
			if device.DeviceID == request.OutputDeviceID {
				found = true
				break
			}
		}
		if !found {
			response := audio.DeviceSwitchResponse{
				IsAudioReady:   false,
				ErrorMessage:   fmt.Sprintf("Output device %d not found", request.OutputDeviceID),
				RequiredAction: "Select a valid audio output device",
				NewConfig:      config,
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	log.Printf("🔄 Switching to device configuration: input %d, sample rate %.0f Hz, buffer %d",
		config.AudioInputDeviceID, config.SampleRate, config.BufferSize)

	// Switch the devices
	isReady, errorMsg, action, wasRunning, pid := switchAudioDevices(config)

	response := audio.DeviceSwitchResponse{
		IsAudioReady:           isReady,
		ErrorMessage:           errorMsg,
		RequiredAction:         action,
		NewConfig:              config,
		PreviousProcessRunning: wasRunning,
		ProcessRestarted:       wasRunning && isReady, // Only true if something was running and switch succeeded
		PID:                    pid,
	}

	if isReady {
		if wasRunning {
			log.Printf("✅ Device switch successful - audio-host restarted with PID %d", pid)
		} else {
			log.Printf("✅ Device switch successful - audio-host started with PID %d", pid)
		}
	} else {
		log.Printf("❌ Device switch failed: %s", errorMsg)
		if !isReady {
			// If switch failed, make sure we're in a clean state
			audio.Mutex.Lock()
			audio.Process = nil
			audio.Mutex.Unlock()
			audio.Reconfig.SetRunning(false)
		}
	}

	json.NewEncoder(w).Encode(response)
}

func handleDebug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	// Get current audio status
	audio.Mutex.RLock()
	process := audio.Process
	audio.Mutex.RUnlock()

	// Convert AudioDevice slices to debug.Device slices
	inputDevices := make([]debug.Device, len(audio.Data.Devices.AudioInput))
	for i, device := range audio.Data.Devices.AudioInput {
		inputDevices[i] = device
	}

	outputDevices := make([]debug.Device, len(audio.Data.Devices.AudioOutput))
	for i, device := range audio.Data.Devices.AudioOutput {
		outputDevices[i] = device
	}

	// Prepare data for the debug dashboard
	data := debug.DashboardData{
		ProcessRunning: process != nil && process.IsRunning(),
		InputDevices:   inputDevices,
		OutputDevices:  outputDevices,
		PluginCount:    len(audio.Data.Plugins),
		DefaultInput:   audio.Data.Devices.Defaults.DefaultInput,
		DefaultOutput:  audio.Data.Devices.Defaults.DefaultOutput,
		DefaultRate:    audio.Data.Devices.DefaultSampleRate,
		Timestamp:      audio.Data.Devices.Timestamp,
	}

	if data.ProcessRunning {
		data.PID = process.GetPID()
		// Try to get engine status
		output, err := process.SendCommand("status")
		if err == nil {
			data.StatusDetails = output
			data.EngineRunning = strings.Contains(output, "running=true")
		} else {
			data.StatusDetails = fmt.Sprintf("Error getting status: %v", err)
		}
	}

	// Generate and write HTML response
	html := debug.RenderHTML(data)
	w.Write([]byte(html))
}

// handleConfigChange processes intelligent configuration changes
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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("GET /api/devices", handleDevices)
	mux.HandleFunc("GET /api/plugins", handlePlugins)
	mux.HandleFunc("GET /api/plugins/{id}", handlePlugin)
	mux.HandleFunc("GET /api/data", handleServerData)

	// Audio control routes
	mux.HandleFunc("POST /api/audio/start", handleStartAudio)
	mux.HandleFunc("POST /api/audio/stop", handleStopAudio)
	mux.HandleFunc("POST /api/audio/command", handleAudioCommand)
	mux.HandleFunc("GET /api/audio/status", handleAudioStatus)
	mux.HandleFunc("GET /api/audio/suggest-sample-rate", handleSuggestSampleRate)
	mux.HandleFunc("POST /api/audio/config-change", func(w http.ResponseWriter, r *http.Request) {
		handleConfigChange(w, r, audio.Reconfig)
	})
	mux.HandleFunc("POST /api/audio/test-devices", handleTestDevices)
	mux.HandleFunc("POST /api/audio/switch-devices", handleSwitchDevices)

	// Debug/testing routes
	mux.HandleFunc("GET /debug", handleDebug)

	// Static file serving (for WASM app) with no-cache headers for development
	fs := http.FileServer(http.Dir("./frontend/static/"))

	// Wrap the file server to add no-cache headers
	noCacheFS := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add no-cache headers for development
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		// Serve the file
		fs.ServeHTTP(w, r)
	})

	mux.Handle("GET /", noCacheFS)

	return mux
}

// checkPortAvailable checks if the specified port is available for binding
func checkPortAvailable(port string) error {
	// Try to bind to the port to see if it's available
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("port %s is not available: %v", port, err)
	}
	// Close immediately since we're just checking availability
	listener.Close()
	return nil
}

func main() {
	log.Println("🚀 Starting Rackless Audio Server...")

	// Initialize the audio package
	if err := audio.Initialize(); err != nil {
		log.Fatalf("❌ Failed to initialize audio package: %v", err)
	}

	// Check port availability first before doing any expensive operations
	const serverPort = "8080"
	log.Printf("🔍 Checking if port %s is available...", serverPort)
	if err := checkPortAvailable(serverPort); err != nil {
		log.Fatalf("❌ Server startup failed: %v\n\n💡 Possible solutions:\n   • Stop any other process using port %s\n   • Wait a moment and try again\n   • Check with: lsof -i :%s", err, serverPort, serverPort)
	}
	log.Printf("✅ Port %s is available", serverPort)

	// Load device information
	if err := audio.LoadDevices(); err != nil {
		log.Fatalf("❌ Failed to load devices: %v", err)
	}

	// Load plugin information
	if err := audio.LoadPlugins(); err != nil {
		log.Fatalf("❌ Failed to load plugins: %v", err)
	}

	log.Println("🎵 Rackless Audio Server initialized successfully!")
	log.Printf("📊 Server data summary:")
	log.Printf("   • Default audio input: Device %d", audio.Data.Devices.Defaults.DefaultInput)
	log.Printf("   • Default audio output: Device %d", audio.Data.Devices.Defaults.DefaultOutput)
	log.Printf("   • Default sample rate: %.0f Hz", audio.Data.Devices.DefaultSampleRate)
	log.Printf("   • Total plugins available: %d", len(audio.Data.Plugins))

	// Setup routes
	router := setupRoutes()
	handler := corsMiddleware(router)

	log.Printf("🌐 Starting HTTP server on :%s...", serverPort)
	log.Println("📡 API endpoints available:")
	log.Println("   • GET /api/health - Server health status")
	log.Println("   • GET /api/devices - Audio device information")
	log.Println("   • GET /api/plugins - AudioUnit plugin list")
	log.Println("   • GET /api/plugins/{id} - Individual plugin details")
	log.Println("   • GET /api/data - Complete server data")
	log.Println("   • POST /api/audio/start - Start audio-host with validation")
	log.Println("   • POST /api/audio/stop - Stop audio-host")
	log.Println("   • POST /api/audio/command - Send command to running audio-host")
	log.Println("   • GET /api/audio/status - Get audio-host status")
	log.Println("   • GET /api/audio/suggest-sample-rate - Find compatible sample rate")
	log.Println("   • POST /api/audio/test-devices - Test device configuration (returns isAudioReady)")
	log.Println("   • POST /api/audio/switch-devices - Switch audio devices (stops current, starts new)")
	log.Println("   • GET /debug - Debug dashboard (HTML interface)")
	log.Println("   • GET / - Static file serving (web app)")
	log.Println("")
	log.Println("🎯 Smart audio controller ready with bidirectional communication!")
	log.Println("   • Server validates sample rate compatibility before starting audio-host")
	log.Println("   • Audio-host provides clear error messages for any failures")
	log.Println("   • Real-time command communication with running audio-host processes")
	log.Println("   • Automatic process management and cleanup")

	err := http.ListenAndServe(":"+serverPort, handler)
	if err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
