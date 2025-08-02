package audio

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// StartAudioHostProcess creates and starts a new audio-host process with the given configuration
func StartAudioHostProcess(config AudioConfig) (*AudioHostProcess, error) {
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

// waitForReady waits for the READY signal from audio-host
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

// handleStderr continuously reads and logs stderr output
func (p *AudioHostProcess) handleStderr() {
	scanner := bufio.NewScanner(p.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("🎧 Audio-host: %s", line)
	}
}

// handleProcessExit handles process cleanup when it exits
func (p *AudioHostProcess) handleProcessExit() {
	p.cmd.Wait()
	p.mu.Lock()
	p.running = false
	p.mu.Unlock()
	log.Printf("🔇 Audio-host process (PID %d) has exited", p.pid)
}

// SendCommand sends a command to the audio-host process and returns the response
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

// Stop gracefully stops the audio-host process
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

// IsRunning returns whether the process is currently running
func (p *AudioHostProcess) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// GetPID returns the process ID
func (p *AudioHostProcess) GetPID() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pid
}
