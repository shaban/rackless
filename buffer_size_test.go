package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test buffer size validation in server
func TestBufferSizeValidation(t *testing.T) {
	tests := []struct {
		name           string
		bufferSize     int
		expectedStatus int
		shouldPass     bool
		description    string
	}{
		{
			name:           "Valid_32_samples",
			bufferSize:     32,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
			description:    "Minimum professional audio buffer size",
		},
		{
			name:           "Valid_64_samples",
			bufferSize:     64,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
			description:    "Common low-latency buffer size",
		},
		{
			name:           "Valid_128_samples",
			bufferSize:     128,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
			description:    "Standard low-latency buffer size",
		},
		{
			name:           "Valid_256_samples_default",
			bufferSize:     256,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
			description:    "Default buffer size (good balance)",
		},
		{
			name:           "Valid_512_samples",
			bufferSize:     512,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
			description:    "Standard stable buffer size",
		},
		{
			name:           "Valid_1024_samples",
			bufferSize:     1024,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
			description:    "Maximum professional audio buffer size",
		},
		{
			name:           "Zero_uses_default",
			bufferSize:     0,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
			description:    "Zero buffer size should use default (256)",
		},
		{
			name:           "Invalid_too_small_16",
			bufferSize:     16,
			expectedStatus: http.StatusBadRequest,
			shouldPass:     false,
			description:    "Too small - below professional minimum",
		},
		{
			name:           "Invalid_too_small_31",
			bufferSize:     31,
			expectedStatus: http.StatusBadRequest,
			shouldPass:     false,
			description:    "Just below minimum threshold",
		},
		{
			name:           "Invalid_too_large_1025",
			bufferSize:     1025,
			expectedStatus: http.StatusBadRequest,
			shouldPass:     false,
			description:    "Just above maximum threshold",
		},
		{
			name:           "Invalid_too_large_2048",
			bufferSize:     2048,
			expectedStatus: http.StatusBadRequest,
			shouldPass:     false,
			description:    "Way too large for professional audio",
		},
		{
			name:           "Invalid_negative",
			bufferSize:     -1,
			expectedStatus: http.StatusBadRequest,
			shouldPass:     false,
			description:    "Negative buffer size is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure audio-host is stopped before each test
			stopAudioHost()
			defer stopAudioHost() // Clean up after test

			// Create a StartAudioRequest with the test buffer size
			request := StartAudioRequest{
				Config: AudioConfig{
					SampleRate:         44100,
					AudioInputDeviceID: 0,
					BufferSize:         tt.bufferSize,
				},
			}

			// Convert to JSON
			jsonData, err := json.Marshal(request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Create HTTP request
			req := httptest.NewRequest("POST", "/start-audio", bytes.NewReader(jsonData))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call the handler
			handleStartAudio(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Parse response
			var response StartAudioResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Check success flag
			if response.Success != tt.shouldPass {
				t.Errorf("Expected success=%v, got success=%v", tt.shouldPass, response.Success)
			}

			// For failed cases, check error message mentions buffer size
			if !tt.shouldPass && response.Message != "" {
				if !contains(response.Message, "buffer size") && !contains(response.Message, "Buffer size") {
					t.Errorf("Error message should mention buffer size, got: %s", response.Message)
				}
			}

			t.Logf("✅ %s: %s", tt.name, tt.description)
			if !tt.shouldPass {
				t.Logf("   Expected error: %s", response.Message)
			}
		})
	}
}

// Test edge cases and boundary conditions
func TestBufferSizeBoundaryConditions(t *testing.T) {
	boundaryTests := []struct {
		name       string
		bufferSize int
		shouldPass bool
	}{
		{"Boundary_31_invalid", 31, false},
		{"Boundary_32_valid", 32, true},
		{"Boundary_1024_valid", 1024, true},
		{"Boundary_1025_invalid", 1025, false},
	}

	for _, tt := range boundaryTests {
		t.Run(tt.name, func(t *testing.T) {
			stopAudioHost()
			defer stopAudioHost()

			request := StartAudioRequest{
				Config: AudioConfig{
					SampleRate:         44100,
					AudioInputDeviceID: 0,
					BufferSize:         tt.bufferSize,
				},
			}

			jsonData, _ := json.Marshal(request)
			req := httptest.NewRequest("POST", "/start-audio", bytes.NewReader(jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handleStartAudio(w, req)

			var response StartAudioResponse
			json.Unmarshal(w.Body.Bytes(), &response)

			if response.Success != tt.shouldPass {
				t.Errorf("Buffer size %d: expected success=%v, got success=%v",
					tt.bufferSize, tt.shouldPass, response.Success)
			}
		})
	}
}

// Test default buffer size application
func TestDefaultBufferSize(t *testing.T) {
	request := StartAudioRequest{
		Config: AudioConfig{
			SampleRate:         44100,
			AudioInputDeviceID: 0,
			BufferSize:         0, // Should trigger default
		},
	}

	jsonData, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/start-audio", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handleStartAudio(w, req)

	var response StartAudioResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Success {
		t.Errorf("Default buffer size should be valid, got error: %s", response.Message)
	}

	t.Logf("✅ Default buffer size handling works correctly")
}

// Benchmark buffer size validation performance
func BenchmarkBufferSizeValidation(b *testing.B) {
	request := StartAudioRequest{
		Config: AudioConfig{
			SampleRate:         44100,
			AudioInputDeviceID: 0,
			BufferSize:         256,
		},
	}

	jsonData, _ := json.Marshal(request)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/start-audio", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleStartAudio(w, req)
	}
}

// Test common professional audio buffer sizes
func TestProfessionalAudioBufferSizes(t *testing.T) {
	professionalSizes := []int{32, 64, 128, 256, 512, 1024}

	for _, size := range professionalSizes {
		t.Run(fmt.Sprintf("Professional_%d", size), func(t *testing.T) {
			request := StartAudioRequest{
				Config: AudioConfig{
					SampleRate:         44100,
					AudioInputDeviceID: 0,
					BufferSize:         size,
				},
			}

			jsonData, _ := json.Marshal(request)
			req := httptest.NewRequest("POST", "/start-audio", bytes.NewReader(jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handleStartAudio(w, req)

			var response StartAudioResponse
			json.Unmarshal(w.Body.Bytes(), &response)

			if !response.Success {
				t.Errorf("Professional buffer size %d should be valid, got error: %s",
					size, response.Message)
			}

			t.Logf("✅ Professional buffer size %d samples validated", size)
		})
	}
}

// Helper function to check if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

// Helper function to stop the audio-host between tests
func stopAudioHost() {
	req := httptest.NewRequest("POST", "/api/audio/stop", nil)
	w := httptest.NewRecorder()
	handleStopAudio(w, req)
	// Ignore errors - it's OK if nothing was running
}

// Test concurrent buffer size validation (stress test)
func TestConcurrentBufferSizeValidation(t *testing.T) {
	const numConcurrent = 10
	const numRequests = 5

	done := make(chan bool, numConcurrent)
	errors := make(chan error, numConcurrent*numRequests)

	for i := 0; i < numConcurrent; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for j := 0; j < numRequests; j++ {
				// Test different buffer sizes concurrently
				bufferSize := []int{32, 64, 128, 256, 512, 1024}[j%len([]int{32, 64, 128, 256, 512, 1024})]

				request := StartAudioRequest{
					Config: AudioConfig{
						SampleRate:         44100,
						AudioInputDeviceID: 0,
						BufferSize:         bufferSize,
					},
				}

				jsonData, _ := json.Marshal(request)
				req := httptest.NewRequest("POST", "/start-audio", bytes.NewReader(jsonData))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				handleStartAudio(w, req)

				var response StartAudioResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					errors <- fmt.Errorf("goroutine %d, request %d: unmarshal error: %v", goroutineID, j, err)
					continue
				}

				if !response.Success {
					errors <- fmt.Errorf("goroutine %d, request %d: unexpected failure for buffer size %d: %s",
						goroutineID, j, bufferSize, response.Message)
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete with timeout
	timeout := time.After(10 * time.Second)
	completed := 0

	for completed < numConcurrent {
		select {
		case <-done:
			completed++
		case err := <-errors:
			t.Error(err)
		case <-timeout:
			t.Fatal("Test timed out - possible deadlock")
		}
	}

	// Check for any remaining errors
	close(errors)
	for err := range errors {
		t.Error(err)
	}

	t.Logf("✅ Concurrent validation test completed: %d goroutines × %d requests", numConcurrent, numRequests)
}
