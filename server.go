package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Device structures based on standalone/devices output
type AudioDevice struct {
	DeviceID             int    `json:"deviceId"`
	UID                  string `json:"uid"`
	SupportedSampleRates []int  `json:"supportedSampleRates"`
	ChannelCount         int    `json:"channelCount"`
	IsDefault            bool   `json:"isDefault"`
	IsOnline             bool   `json:"isOnline"`
	Name                 string `json:"name"`
	SupportedBitDepths   []int  `json:"supportedBitDepths"`
}

type MIDIDevice struct {
	UID        string `json:"uid"`
	Name       string `json:"name"`
	EndpointID int    `json:"endpointId"`
	IsOnline   bool   `json:"isOnline"`
}

type DefaultDevices struct {
	DefaultInput  int `json:"defaultInput"`
	DefaultOutput int `json:"defaultOutput"`
}

type DevicesData struct {
	TotalMIDIInputDevices   int            `json:"totalMIDIInputDevices"`
	MIDIInput               []MIDIDevice   `json:"midiInput"`
	Defaults                DefaultDevices `json:"defaults"`
	TotalAudioInputDevices  int            `json:"totalAudioInputDevices"`
	AudioInput              []AudioDevice  `json:"audioInput"`
	AudioOutput             []AudioDevice  `json:"audioOutput"`
	TotalMIDIOutputDevices  int            `json:"totalMIDIOutputDevices"`
	Timestamp               string         `json:"timestamp"`
	MIDIOutput              []MIDIDevice   `json:"midiOutput"`
	TotalAudioOutputDevices int            `json:"totalAudioOutputDevices"`
	DefaultSampleRate       float64        `json:"defaultSampleRate"`
}

// Plugin structures based on standalone/inspector output
type PluginParameter struct {
	DisplayName         string   `json:"displayName"`
	DefaultValue        float64  `json:"defaultValue"`
	CurrentValue        float64  `json:"currentValue"`
	Address             int      `json:"address"`
	MaxValue            float64  `json:"maxValue"`
	Unit                string   `json:"unit"`
	Identifier          string   `json:"identifier"`
	MinValue            float64  `json:"minValue"`
	CanRamp             bool     `json:"canRamp"`
	IsWritable          bool     `json:"isWritable"`
	RawFlags            int64    `json:"rawFlags"`
	IndexedValues       []string `json:"indexedValues,omitempty"`
	IndexedValuesSource string   `json:"indexedValuesSource,omitempty"`
}

type Plugin struct {
	Parameters     []PluginParameter `json:"parameters"`
	ManufacturerID string            `json:"manufacturerID"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Subtype        string            `json:"subtype"`
}

// Server data - holds the results of both tools
type ServerData struct {
	Devices DevicesData `json:"devices"`
	Plugins []Plugin    `json:"plugins"`
}

// Audio configuration for starting audio-host
type AudioConfig struct {
	SampleRate         float64 `json:"sampleRate"`
	BufferSize         int     `json:"bufferSize,omitempty"`
	AudioInputDeviceID int     `json:"audioInputDeviceID,omitempty"`
	AudioInputChannel  int     `json:"audioInputChannel,omitempty"`
	EnableTestTone     bool    `json:"enableTestTone,omitempty"`
	PluginPath         string  `json:"pluginPath,omitempty"`
}

// Audio start request
type StartAudioRequest struct {
	Config AudioConfig `json:"config"`
}

// Audio start response
type StartAudioResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	PID     int    `json:"pid,omitempty"`
}

// Audio command request
type AudioCommandRequest struct {
	Command string `json:"command"`
}

// Audio command response
type AudioCommandResponse struct {
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Device test request for simplified boolean approach
type DeviceTestRequest struct {
	InputDeviceID  int     `json:"inputDeviceID"`
	OutputDeviceID int     `json:"outputDeviceID,omitempty"`
	SampleRate     float64 `json:"sampleRate"`
	BufferSize     int     `json:"bufferSize,omitempty"`
}

// Device test response with boolean ready state
type DeviceTestResponse struct {
	IsAudioReady    bool   `json:"isAudioReady"`
	ErrorMessage    string `json:"errorMessage,omitempty"`
	RequiredAction  string `json:"requiredAction,omitempty"`
	TestedConfig    AudioConfig `json:"testedConfig"`
}

// Device switch request for changing audio devices
type DeviceSwitchRequest struct {
	InputDeviceID  int     `json:"inputDeviceID"`
	OutputDeviceID int     `json:"outputDeviceID,omitempty"`
	SampleRate     float64 `json:"sampleRate"`
	BufferSize     int     `json:"bufferSize,omitempty"`
}

// Device switch response with boolean ready state
type DeviceSwitchResponse struct {
	IsAudioReady     bool   `json:"isAudioReady"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
	RequiredAction   string `json:"requiredAction,omitempty"`
	NewConfig        AudioConfig `json:"newConfig"`
	PreviousRunning  bool   `json:"previousRunning"`
	ProcessRestarted bool   `json:"processRestarted"`
	PID              int    `json:"pid,omitempty"`
}

// AudioHost process management
type AudioHostProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	pid     int
	running bool
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

var (
	serverData       ServerData
	audioHostProcess *AudioHostProcess
	audioHostMutex   sync.RWMutex
)

// Sample rate validation functions
func validateSampleRate(config AudioConfig) error {
	sampleRate := int(config.SampleRate)

	// Check output device sample rate compatibility
	for _, device := range serverData.Devices.AudioOutput {
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
		for _, device := range serverData.Devices.AudioInput {
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
		for _, device := range serverData.Devices.AudioInput {
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
		for _, device := range serverData.Devices.AudioOutput {
			if device.DeviceID == outputDeviceID {
				outputSupportedRates = device.SupportedSampleRates
				break
			}
		}
	} else {
		// Use default output device
		for _, device := range serverData.Devices.AudioOutput {
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
func testDeviceConfiguration(config AudioConfig) (bool, string, string) {
	// Step 1: Validate configuration parameters
	if err := validateSampleRate(config); err != nil {
		return false, 
			fmt.Sprintf("Device configuration invalid: %v", err),
			"Please select compatible audio devices and sample rate"
	}

	// Step 2: Try to actually start audio-host with these parameters
	// This is the real test - can we initialize the audio system?
	tempProcess, err := startAudioHostProcess(config)
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
func switchAudioDevices(config AudioConfig) (bool, string, string, bool, int) {
	// Step 1: Check if audio-host is currently running
	audioHostMutex.RLock()
	wasRunning := audioHostProcess != nil && audioHostProcess.IsRunning()
	currentProcess := audioHostProcess
	audioHostMutex.RUnlock()

	// Step 2: Stop current audio-host if running
	if wasRunning {
		log.Printf("🔄 Stopping current audio-host to switch devices...")
		audioHostMutex.Lock()
		audioHostProcess = nil
		audioHostMutex.Unlock()
		
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
	newProcess, err := startAudioHostProcess(config)
	if err != nil {
		return false,
			fmt.Sprintf("Failed to start audio-host with new devices: %v", err),
			"Check if new devices are available and not in use by other applications",
			wasRunning, 0
	}

	// Step 5: Store the new process
	audioHostMutex.Lock()
	audioHostProcess = newProcess
	audioHostMutex.Unlock()

	// Update reconfiguration system
	audioReconfig.SetCurrentConfig(config)
	audioReconfig.SetRunning(true)

	log.Printf("✅ Audio devices switched successfully - new PID %d", newProcess.pid)
	return true, "", "", wasRunning, newProcess.pid
}

// Audio-host process management functions
func startAudioHostProcess(config AudioConfig) (*AudioHostProcess, error) {
	// Build audio-host command
	args := []string{"--command-mode", "--sample-rate", fmt.Sprintf("%.0f", config.SampleRate)}

	if config.BufferSize > 0 {
		args = append(args, "--buffer-size", strconv.Itoa(config.BufferSize))
	}

	if config.AudioInputDeviceID > 0 {
		args = append(args, "--audio-input-device", strconv.Itoa(config.AudioInputDeviceID))
		args = append(args, "--audio-input-channel", strconv.Itoa(config.AudioInputChannel))
	}

	if !config.EnableTestTone {
		args = append(args, "--no-tone")
	}

	log.Printf("🚀 Starting: ./standalone/audio-host/audio-host %s", strings.Join(args, " "))

	// Create context for process management
	ctx, cancel := context.WithCancel(context.Background())

	// Create command
	cmd := exec.CommandContext(ctx, "./standalone/audio-host/audio-host", args...)

	// Set up pipes for bidirectional communication
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		cancel()
		return nil, fmt.Errorf("failed to start audio-host: %v", err)
	}

	process := &AudioHostProcess{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		pid:     cmd.Process.Pid,
		running: true,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Start goroutine to handle process exit
	go process.handleProcessExit()

	// Wait for "READY" signal from audio-host
	if err := process.waitForReady(); err != nil {
		process.Stop()
		return nil, fmt.Errorf("audio-host failed to start: %v", err)
	}

	// Now start the stderr handler for ongoing logging
	go process.handleStderr()

	log.Printf("✅ Audio-host started successfully with PID %d", process.pid)
	return process, nil
}

func (p *AudioHostProcess) waitForReady() error {
	// Read from stderr until we see "READY"
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	readyChan := make(chan bool, 1)

	// Start a goroutine to scan stderr for the READY signal
	go func() {
		defer close(readyChan)
		scanner := bufio.NewScanner(p.stderr)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("🎧 Audio-host stderr: %s", line)
			if strings.Contains(line, "READY") {
				readyChan <- true
				return
			}
		}
		// If scanner exits without finding READY, send false
		readyChan <- false
	}()

	select {
	case ready := <-readyChan:
		if ready {
			return nil
		}
		return fmt.Errorf("audio-host exited without sending READY signal")
	case <-timeout.C:
		return fmt.Errorf("timeout waiting for READY signal from audio-host")
	}
}

func (p *AudioHostProcess) handleStderr() {
	scanner := bufio.NewScanner(p.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("🎧 Audio-host: %s", line)
	}
}

func (p *AudioHostProcess) handleProcessExit() {
	p.cmd.Wait()
	p.mu.Lock()
	p.running = false
	p.mu.Unlock()
	log.Printf("🔇 Audio-host process (PID %d) has exited", p.pid)
}

func (p *AudioHostProcess) SendCommand(command string) (string, error) {
	p.mu.RLock()
	if !p.running {
		p.mu.RUnlock()
		return "", fmt.Errorf("audio-host process is not running")
	}
	stdin := p.stdin
	stdout := p.stdout
	p.mu.RUnlock()

	// Send command
	_, err := fmt.Fprintf(stdin, "%s\n", command)
	if err != nil {
		return "", fmt.Errorf("failed to send command: %v", err)
	}

	// Read response with timeout
	respChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			respChan <- scanner.Text()
		} else {
			errChan <- fmt.Errorf("failed to read response")
		}
	}()

	select {
	case response := <-respChan:
		return response, nil
	case err := <-errChan:
		return "", err
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("timeout waiting for response")
	}
}

func (p *AudioHostProcess) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	// Send quit command if possible
	if p.stdin != nil {
		fmt.Fprintf(p.stdin, "quit\n")
		p.stdin.Close()
	}

	// Cancel context to kill process if needed
	p.cancel()

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited gracefully
	case <-time.After(3 * time.Second):
		// Force kill if it doesn't exit
		if p.cmd.Process != nil {
			p.cmd.Process.Kill()
		}
	}

	// Close pipes
	if p.stdout != nil {
		p.stdout.Close()
	}
	if p.stderr != nil {
		p.stderr.Close()
	}

	p.running = false
	log.Printf("🔇 Audio-host process stopped")
	return nil
}

func (p *AudioHostProcess) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

func loadDevices() error {
	log.Println("Loading device information...")

	cmd := exec.Command("./standalone/devices/devices")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run devices tool: %v", err)
	}

	err = json.Unmarshal(output, &serverData.Devices)
	if err != nil {
		return fmt.Errorf("failed to parse devices JSON: %v", err)
	}

	log.Printf("✅ Loaded %d audio input devices, %d audio output devices, %d MIDI input devices, %d MIDI output devices",
		serverData.Devices.TotalAudioInputDevices,
		serverData.Devices.TotalAudioOutputDevices,
		serverData.Devices.TotalMIDIInputDevices,
		serverData.Devices.TotalMIDIOutputDevices)

	return nil
}

func loadPlugins() error {
	log.Println("Loading plugin information...")

	cmd := exec.Command("./standalone/inspector/inspector")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run inspector tool: %v", err)
	}

	err = json.Unmarshal(output, &serverData.Plugins)
	if err != nil {
		return fmt.Errorf("failed to parse plugins JSON: %v", err)
	}

	log.Printf("✅ Loaded %d AudioUnit plugins", len(serverData.Plugins))

	return nil
}

// API Handlers
func handleDevices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // For WASM development

	if err := json.NewEncoder(w).Encode(serverData.Devices); err != nil {
		http.Error(w, "Failed to encode devices data", http.StatusInternalServerError)
		return
	}
}

func handlePlugins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // For WASM development

	if err := json.NewEncoder(w).Encode(serverData.Plugins); err != nil {
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

	if pluginID < 0 || pluginID >= len(serverData.Plugins) {
		http.Error(w, "Plugin not found", http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(serverData.Plugins[pluginID]); err != nil {
		http.Error(w, "Failed to encode plugin data", http.StatusInternalServerError)
		return
	}
}

func handleServerData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // For WASM development

	if err := json.NewEncoder(w).Encode(serverData); err != nil {
		http.Error(w, "Failed to encode server data", http.StatusInternalServerError)
		return
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	health := map[string]interface{}{
		"status":    "healthy",
		"devices":   len(serverData.Devices.AudioInput) + len(serverData.Devices.AudioOutput),
		"plugins":   len(serverData.Plugins),
		"timestamp": serverData.Devices.Timestamp,
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
	audioHostMutex.RLock()
	if audioHostProcess != nil && audioHostProcess.IsRunning() {
		audioHostMutex.RUnlock()
		response := StartAudioResponse{
			Success: false,
			Message: fmt.Sprintf("Audio-host is already running (PID %d)", audioHostProcess.pid),
		}
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}
	audioHostMutex.RUnlock()

	var request StartAudioRequest
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
		response := StartAudioResponse{
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
		response := StartAudioResponse{
			Success: false,
			Message: fmt.Sprintf("Sample rate validation failed: %v", err),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Start the audio-host process
	process, err := startAudioHostProcess(config)
	if err != nil {
		log.Printf("❌ Failed to start audio-host: %v", err)
		response := StartAudioResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start audio-host: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Store the process globally
	audioHostMutex.Lock()
	audioHostProcess = process
	audioHostMutex.Unlock()

	// Update the reconfiguration system with the current configuration
	audioReconfig.SetCurrentConfig(config)
	audioReconfig.SetRunning(true)

	response := StartAudioResponse{
		Success: true,
		Message: "Audio-host started successfully with bidirectional communication",
		PID:     process.pid,
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

	audioHostMutex.Lock()
	process := audioHostProcess
	audioHostProcess = nil
	audioHostMutex.Unlock()

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
			"message": fmt.Sprintf("Failed to stop audio-host: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Audio-host stopped successfully",
	}

	// Update the reconfiguration system to reflect stopped state
	audioReconfig.SetRunning(false)

	json.NewEncoder(w).Encode(response)
}

func handleAudioCommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request AudioCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	audioHostMutex.RLock()
	process := audioHostProcess
	audioHostMutex.RUnlock()

	if process == nil || !process.IsRunning() {
		response := AudioCommandResponse{
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
		response := AudioCommandResponse{
			Success: false,
			Error:   fmt.Sprintf("Command failed: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("✅ Command response: %s", output)

	response := AudioCommandResponse{
		Success: true,
		Output:  output,
	}

	json.NewEncoder(w).Encode(response)
}

func handleAudioStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	audioHostMutex.RLock()
	process := audioHostProcess
	audioHostMutex.RUnlock()

	status := map[string]interface{}{
		"running": false,
		"pid":     nil,
	}

	if process != nil && process.IsRunning() {
		status["running"] = true
		status["pid"] = process.pid

		// Try to get detailed status from audio-host
		output, err := process.SendCommand("status")
		if err == nil {
			status["details"] = output
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

	var request DeviceTestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Build AudioConfig from test request
	config := AudioConfig{
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
		for _, device := range serverData.Devices.AudioOutput {
			if device.DeviceID == request.OutputDeviceID {
				found = true
				break
			}
		}
		if !found {
			response := DeviceTestResponse{
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

	response := DeviceTestResponse{
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

	var request DeviceSwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Build AudioConfig from switch request
	config := AudioConfig{
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
		for _, device := range serverData.Devices.AudioOutput {
			if device.DeviceID == request.OutputDeviceID {
				found = true
				break
			}
		}
		if !found {
			response := DeviceSwitchResponse{
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

	response := DeviceSwitchResponse{
		IsAudioReady:     isReady,
		ErrorMessage:     errorMsg,
		RequiredAction:   action,
		NewConfig:        config,
		PreviousRunning:  wasRunning,
		ProcessRestarted: wasRunning && isReady, // Only true if something was running and switch succeeded
		PID:              pid,
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
			audioHostMutex.Lock()
			audioHostProcess = nil
			audioHostMutex.Unlock()
			audioReconfig.SetRunning(false)
		}
	}

	json.NewEncoder(w).Encode(response)
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
	mux.HandleFunc("POST /api/audio/config-change", handleConfigChange)
	mux.HandleFunc("POST /api/audio/test-devices", handleTestDevices)
	mux.HandleFunc("POST /api/audio/switch-devices", handleSwitchDevices)

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

func main() {
	log.Println("🚀 Starting Rackless Audio Server...")

	// Load device information
	if err := loadDevices(); err != nil {
		log.Fatalf("❌ Failed to load devices: %v", err)
	}

	// Load plugin information
	if err := loadPlugins(); err != nil {
		log.Fatalf("❌ Failed to load plugins: %v", err)
	}

	log.Println("🎵 Rackless Audio Server initialized successfully!")
	log.Printf("📊 Server data summary:")
	log.Printf("   • Default audio input: Device %d", serverData.Devices.Defaults.DefaultInput)
	log.Printf("   • Default audio output: Device %d", serverData.Devices.Defaults.DefaultOutput)
	log.Printf("   • Default sample rate: %.0f Hz", serverData.Devices.DefaultSampleRate)
	log.Printf("   • Total plugins available: %d", len(serverData.Plugins))

	// Setup routes
	router := setupRoutes()
	handler := corsMiddleware(router)

	log.Println("🌐 Starting HTTP server on :8080...")
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
	log.Println("   • GET / - Static file serving (web app)")
	log.Println("")
	log.Println("🎯 Smart audio controller ready with bidirectional communication!")
	log.Println("   • Server validates sample rate compatibility before starting audio-host")
	log.Println("   • Audio-host provides clear error messages for any failures")
	log.Println("   • Real-time command communication with running audio-host processes")
	log.Println("   • Automatic process management and cleanup")

	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
