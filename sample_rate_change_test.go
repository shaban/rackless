package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test sample rate change behavior - does audio-host need restart?
func TestSampleRateChangeRequiresRestart(t *testing.T) {
	// Ensure clean state
	stopAudioHost()
	defer stopAudioHost()

	// Start audio-host with 44.1kHz
	t.Log("🎯 Starting audio-host with 44.1kHz")
	request1 := StartAudioRequest{
		Config: AudioConfig{
			SampleRate:         44100,
			AudioInputDeviceID: 0,
			BufferSize:         256,
		},
	}

	jsonData1, _ := json.Marshal(request1)
	req1 := httptest.NewRequest("POST", "/start-audio", bytes.NewReader(jsonData1))
	req1.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	handleStartAudio(w1, req1)

	var response1 StartAudioResponse
	json.Unmarshal(w1.Body.Bytes(), &response1)

	if !response1.Success {
		t.Fatalf("Failed to start audio with 44.1kHz: %s", response1.Message)
	}

	originalPID := response1.PID
	t.Logf("✅ Audio-host started successfully with PID %d at 44.1kHz", originalPID)

	// Try to start with different sample rate (48kHz) while already running
	t.Log("🔄 Attempting to change sample rate to 48kHz while running...")
	request2 := StartAudioRequest{
		Config: AudioConfig{
			SampleRate:         48000,
			AudioInputDeviceID: 0,
			BufferSize:         256,
		},
	}

	jsonData2, _ := json.Marshal(request2)
	req2 := httptest.NewRequest("POST", "/start-audio", bytes.NewReader(jsonData2))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	handleStartAudio(w2, req2)

	var response2 StartAudioResponse
	json.Unmarshal(w2.Body.Bytes(), &response2)

	// This should fail because audio-host is already running
	if response2.Success {
		t.Errorf("Expected failure when trying to change sample rate while running, but got success")
	}

	// Check that we get the "already running" error
	if w2.Code != http.StatusConflict {
		t.Errorf("Expected HTTP 409 Conflict, got %d", w2.Code)
	}

	expectedError := "Audio-host is already running"
	if !contains(response2.Message, expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedError, response2.Message)
	}

	t.Logf("✅ Correctly rejected sample rate change while running: %s", response2.Message)

	// Now stop the audio-host
	t.Log("⏹️ Stopping audio-host...")
	stopReq := httptest.NewRequest("POST", "/api/audio/stop", nil)
	stopW := httptest.NewRecorder()
	handleStopAudio(stopW, stopReq)

	var stopResponse map[string]interface{}
	json.Unmarshal(stopW.Body.Bytes(), &stopResponse)

	if success, ok := stopResponse["success"].(bool); !ok || !success {
		t.Errorf("Failed to stop audio-host: %v", stopResponse)
	}

	t.Log("✅ Audio-host stopped successfully")

	// Now try to start with the new sample rate
	t.Log("🆕 Starting audio-host with 48kHz after stop...")
	req3 := httptest.NewRequest("POST", "/start-audio", bytes.NewReader(jsonData2))
	req3.Header.Set("Content-Type", "application/json")

	w3 := httptest.NewRecorder()
	handleStartAudio(w3, req3)

	var response3 StartAudioResponse
	json.Unmarshal(w3.Body.Bytes(), &response3)

	if !response3.Success {
		t.Errorf("Failed to start audio with 48kHz after stop: %s", response3.Message)
	}

	newPID := response3.PID
	t.Logf("✅ Audio-host started successfully with new PID %d at 48kHz", newPID)

	// Verify it's a different process (PID should be different)
	if newPID == originalPID {
		t.Errorf("Expected different PID after restart, but got same PID %d", newPID)
	}

	t.Log("🎉 Test complete: Sample rate changes require audio-host restart")
}

// Test buffer size change behavior
func TestBufferSizeChangeRequiresRestart(t *testing.T) {
	// Ensure clean state
	stopAudioHost()
	defer stopAudioHost()

	// Start audio-host with 256 buffer size
	t.Log("🎯 Starting audio-host with 256 buffer size")
	request1 := StartAudioRequest{
		Config: AudioConfig{
			SampleRate:         44100,
			AudioInputDeviceID: 0,
			BufferSize:         256,
		},
	}

	jsonData1, _ := json.Marshal(request1)
	req1 := httptest.NewRequest("POST", "/start-audio", bytes.NewReader(jsonData1))
	req1.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	handleStartAudio(w1, req1)

	var response1 StartAudioResponse
	json.Unmarshal(w1.Body.Bytes(), &response1)

	if !response1.Success {
		t.Fatalf("Failed to start audio with 256 buffer: %s", response1.Message)
	}

	originalPID := response1.PID
	t.Logf("✅ Audio-host started successfully with PID %d at 256 buffer size", originalPID)

	// Try to start with different buffer size (512) while already running
	t.Log("🔄 Attempting to change buffer size to 512 while running...")
	request2 := StartAudioRequest{
		Config: AudioConfig{
			SampleRate:         44100,
			AudioInputDeviceID: 0,
			BufferSize:         512,
		},
	}

	jsonData2, _ := json.Marshal(request2)
	req2 := httptest.NewRequest("POST", "/start-audio", bytes.NewReader(jsonData2))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	handleStartAudio(w2, req2)

	var response2 StartAudioResponse
	json.Unmarshal(w2.Body.Bytes(), &response2)

	// This should fail because audio-host is already running
	if response2.Success {
		t.Errorf("Expected failure when trying to change buffer size while running, but got success")
	}

	// Check that we get the "already running" error
	if w2.Code != http.StatusConflict {
		t.Errorf("Expected HTTP 409 Conflict, got %d", w2.Code)
	}

	t.Logf("✅ Correctly rejected buffer size change while running: %s", response2.Message)

	t.Log("🎉 Test complete: Buffer size changes also require audio-host restart")
}

// Test what audio parameters can change without restart
func TestDynamicParameterChanges(t *testing.T) {
	// This test documents which parameters (if any) can be changed dynamically
	// Based on the audio-host command interface

	// Ensure clean state
	stopAudioHost()
	defer stopAudioHost()

	// Start audio-host
	t.Log("🎯 Starting audio-host for dynamic parameter testing")
	request := StartAudioRequest{
		Config: AudioConfig{
			SampleRate:         44100,
			AudioInputDeviceID: 0,
			BufferSize:         256,
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
		t.Fatalf("Failed to start audio: %s", response.Message)
	}

	t.Logf("✅ Audio-host started successfully with PID %d", response.PID)

	// From the command interface analysis, these are the parameters that CAN be changed:
	// - Test tone on/off (tone on/off command)
	// - Test tone frequency (tone freq <hz> command)
	// - Plugin loading/unloading (load-plugin/unload-plugin commands)

	t.Log("📋 Parameters that CAN be changed dynamically (via commands):")
	t.Log("   • Test tone enable/disable")
	t.Log("   • Test tone frequency")
	t.Log("   • Plugin loading/unloading")
	t.Log("")
	t.Log("📋 Parameters that CANNOT be changed without restart:")
	t.Log("   • Sample rate (requires new AudioUnit configuration)")
	t.Log("   • Buffer size (requires new AudioUnit configuration)")
	t.Log("   • Audio input device (requires new AudioUnit configuration)")
	t.Log("   • Audio output device (requires new AudioUnit configuration)")

	t.Log("🎉 Test complete: Core audio parameters require restart for changes")
}
