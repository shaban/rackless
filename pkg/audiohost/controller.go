// Package audiohost provides a Go interface to control the standalone audio host process
package audiohost

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AudioHostConfig defines the configuration for the audio host
type AudioHostConfig struct {
	SampleRate     float64 `json:"sampleRate"`
	BitDepth       int     `json:"bitDepth"`
	BufferSize     int     `json:"bufferSize"`
	EnableTestTone bool    `json:"enableTestTone"`
}

// DefaultConfig returns a reasonable default configuration
func DefaultConfig() AudioHostConfig {
	return AudioHostConfig{
		SampleRate:     44100.0,
		BitDepth:       32,
		BufferSize:     256,
		EnableTestTone: true,
	}
}

// AudioHostStatus represents the current status of the audio host
type AudioHostStatus struct {
	Running     bool    `json:"running"`
	SampleRate  float64 `json:"sampleRate"`
	BufferSize  int     `json:"bufferSize"`
	TestTone    bool    `json:"testTone"`
	ToneFreq    float64 `json:"toneFreq,omitempty"`
	ProcessID   int     `json:"processId,omitempty"`
	Uptime      string  `json:"uptime,omitempty"`
	LastCommand string  `json:"lastCommand,omitempty"`
	LastError   string  `json:"lastError,omitempty"`
}

// AudioHostController manages the standalone audio host process
type AudioHostController struct {
	config     AudioHostConfig
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     *bufio.Scanner
	stderr     *bufio.Scanner
	status     AudioHostStatus
	statusMu   sync.RWMutex
	running    bool
	runningMu  sync.RWMutex
	startTime  time.Time
	
	// Communication channels
	responseChan chan string
	errorChan    chan error
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewAudioHostController creates a new audio host controller
func NewAudioHostController(config AudioHostConfig) (*AudioHostController, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	controller := &AudioHostController{
		config:       config,
		responseChan: make(chan string, 10),
		errorChan:    make(chan error, 10),
		ctx:          ctx,
		cancel:       cancel,
	}
	
	return controller, nil
}

// Start launches the audio host process and begins communication
func (c *AudioHostController) Start() error {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	
	if c.running {
		return fmt.Errorf("audio host already running")
	}
	
	// Find the audio host executable
	execPath, err := c.findAudioHostExecutable()
	if err != nil {
		return fmt.Errorf("failed to find audio host executable: %v", err)
	}
	
	// Build command arguments
	args := []string{"--command-mode"}
	if c.config.SampleRate != 44100.0 {
		args = append(args, "--sample-rate", fmt.Sprintf("%.0f", c.config.SampleRate))
	}
	if c.config.BufferSize != 256 {
		args = append(args, "--buffer-size", strconv.Itoa(c.config.BufferSize))
	}
	if !c.config.EnableTestTone {
		args = append(args, "--no-tone")
	}
	
	// Create command
	c.cmd = exec.CommandContext(c.ctx, execPath, args...)
	
	// Set up pipes
	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}
	c.stdin = stdin
	
	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	c.stdout = bufio.NewScanner(stdout)
	
	stderr, err := c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}
	c.stderr = bufio.NewScanner(stderr)
	
	// Start the process
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start audio host process: %v", err)
	}
	
	c.running = true
	c.startTime = time.Now()
	
	// Start communication goroutines
	go c.readStdout()
	go c.readStderr()
	go c.watchProcess()
	
	// Initialize audio host
	if err := c.sendCommand("start"); err != nil {
		c.Stop()
		return fmt.Errorf("failed to start audio host: %v", err)
	}
	
	// Update initial status
	if err := c.updateStatus(); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to get initial status: %v\n", err)
	}
	
	return nil
}

// Stop gracefully shuts down the audio host process
func (c *AudioHostController) Stop() error {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	
	if !c.running {
		return nil
	}
	
	// Send quit command
	if c.stdin != nil {
		c.sendCommand("quit")
		time.Sleep(100 * time.Millisecond) // Give it time to process
	}
	
	// Cancel context to signal goroutines to stop
	c.cancel()
	
	// Close pipes
	if c.stdin != nil {
		c.stdin.Close()
	}
	
	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		if c.cmd != nil && c.cmd.Process != nil {
			done <- c.cmd.Wait()
		} else {
			done <- nil
		}
	}()
	
	select {
	case <-done:
		// Process exited cleanly
	case <-time.After(2 * time.Second):
		// Force kill if it doesn't exit
		if c.cmd != nil && c.cmd.Process != nil {
			c.cmd.Process.Kill()
		}
	}
	
	c.running = false
	return nil
}

// IsRunning returns whether the audio host process is currently running
func (c *AudioHostController) IsRunning() bool {
	c.runningMu.RLock()
	defer c.runningMu.RUnlock()
	return c.running
}

// GetStatus returns the current status of the audio host
func (c *AudioHostController) GetStatus() AudioHostStatus {
	c.statusMu.RLock()
	defer c.statusMu.RUnlock()
	
	status := c.status
	if c.running && !c.startTime.IsZero() {
		status.Uptime = time.Since(c.startTime).Round(time.Second).String()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		status.ProcessID = c.cmd.Process.Pid
	}
	
	return status
}

// SetTestToneFrequency changes the test tone frequency
func (c *AudioHostController) SetTestToneFrequency(freq float64) error {
	cmd := fmt.Sprintf("tone freq %.1f", freq)
	if err := c.sendCommand(cmd); err != nil {
		return err
	}
	
	c.statusMu.Lock()
	c.status.ToneFreq = freq
	c.statusMu.Unlock()
	
	return nil
}

// EnableTestTone enables or disables the test tone
func (c *AudioHostController) EnableTestTone(enable bool) error {
	var cmd string
	if enable {
		cmd = "tone on"
	} else {
		cmd = "tone off"
	}
	
	if err := c.sendCommand(cmd); err != nil {
		return err
	}
	
	c.statusMu.Lock()
	c.status.TestTone = enable
	c.statusMu.Unlock()
	
	return nil
}

// sendCommand sends a command to the audio host process
func (c *AudioHostController) sendCommand(command string) error {
	c.runningMu.RLock()
	defer c.runningMu.RUnlock()
	
	if !c.running || c.stdin == nil {
		return fmt.Errorf("audio host not running")
	}
	
	_, err := fmt.Fprintf(c.stdin, "%s\n", command)
	if err != nil {
		return fmt.Errorf("failed to send command '%s': %v", command, err)
	}
	
	c.statusMu.Lock()
	c.status.LastCommand = command
	c.statusMu.Unlock()
	
	return nil
}

// updateStatus requests and updates the current status
func (c *AudioHostController) updateStatus() error {
	return c.sendCommand("status")
}

// findAudioHostExecutable locates the audio host executable
func (c *AudioHostController) findAudioHostExecutable() (string, error) {
	// Try current directory first (development)
	if _, err := os.Stat("./audio-host"); err == nil {
		return "./audio-host", nil
	}
	
	// Try standalone-audio-host directory
	standalonePath := filepath.Join("standalone-audio-host", "audio-host")
	if _, err := os.Stat(standalonePath); err == nil {
		return standalonePath, nil
	}
	
	// Try relative to current working directory
	cwd, err := os.Getwd()
	if err == nil {
		relPath := filepath.Join(cwd, "standalone-audio-host", "audio-host")
		if _, err := os.Stat(relPath); err == nil {
			return relPath, nil
		}
	}
	
	return "", fmt.Errorf("audio-host executable not found")
}

// readStdout reads responses from the audio host process
func (c *AudioHostController) readStdout() {
	for c.stdout.Scan() {
		line := strings.TrimSpace(c.stdout.Text())
		if line == "" {
			continue
		}
		
		// Parse status responses
		if strings.HasPrefix(line, "STATUS:") {
			c.parseStatusResponse(line)
		}
		
		// Send to response channel for other handlers
		select {
		case c.responseChan <- line:
		case <-c.ctx.Done():
			return
		}
	}
}

// readStderr reads error output from the audio host process
func (c *AudioHostController) readStderr() {
	for c.stderr.Scan() {
		line := strings.TrimSpace(c.stderr.Text())
		if line == "" {
			continue
		}
		
		// Log error
		fmt.Printf("Audio Host Error: %s\n", line)
		
		c.statusMu.Lock()
		c.status.LastError = line
		c.statusMu.Unlock()
		
		// Send to error channel
		select {
		case c.errorChan <- fmt.Errorf("audio host error: %s", line):
		case <-c.ctx.Done():
			return
		}
	}
}

// watchProcess monitors the audio host process for unexpected exits
func (c *AudioHostController) watchProcess() {
	if c.cmd == nil {
		return
	}
	
	err := c.cmd.Wait()
	
	c.runningMu.Lock()
	c.running = false
	c.runningMu.Unlock()
	
	if err != nil && c.ctx.Err() == nil {
		// Process exited unexpectedly
		fmt.Printf("Audio host process exited unexpectedly: %v\n", err)
		
		select {
		case c.errorChan <- fmt.Errorf("audio host process exited: %v", err):
		case <-c.ctx.Done():
		}
	}
}

// parseStatusResponse parses a status response from the audio host
func (c *AudioHostController) parseStatusResponse(line string) {
	// Expected format: "STATUS: running=true sampleRate=44100 ..."
	parts := strings.Split(line, " ")
	
	c.statusMu.Lock()
	defer c.statusMu.Unlock()
	
	for _, part := range parts[1:] { // Skip "STATUS:"
		if kv := strings.Split(part, "="); len(kv) == 2 {
			key, value := kv[0], kv[1]
			
			switch key {
			case "running":
				c.status.Running = value == "true"
			case "sampleRate":
				if rate, err := strconv.ParseFloat(value, 64); err == nil {
					c.status.SampleRate = rate
				}
			case "bufferSize":
				if size, err := strconv.Atoi(value); err == nil {
					c.status.BufferSize = size
				}
			case "testTone":
				c.status.TestTone = value == "true"
			case "toneFreq":
				if freq, err := strconv.ParseFloat(value, 64); err == nil {
					c.status.ToneFreq = freq
				}
			}
		}
	}
}
