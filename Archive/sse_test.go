package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEventBroadcaster(t *testing.T) {
	// Create event broadcaster
	eb := NewEventBroadcaster()
	eb.Start()

	// Create a test client channel
	client := make(chan DeviceEvent, 5)
	eb.addClient <- client

	// Test event
	testEvent := DeviceEvent{
		Type:      "removed",
		DeviceID:  "test-device-123",
		Name:      "Test Audio Interface",
		Category:  "audio_input",
		Severity:  "critical",
		Message:   "Audio interface disconnected during session",
		Timestamp: time.Now(),
	}

	// Broadcast event
	eb.BroadcastEvent(testEvent)

	// Verify client receives event
	select {
	case receivedEvent := <-client:
		if receivedEvent.Type != testEvent.Type {
			t.Errorf("Expected event type %s, got %s", testEvent.Type, receivedEvent.Type)
		}
		if receivedEvent.DeviceID != testEvent.DeviceID {
			t.Errorf("Expected device ID %s, got %s", testEvent.DeviceID, receivedEvent.DeviceID)
		}
		if receivedEvent.Severity != testEvent.Severity {
			t.Errorf("Expected severity %s, got %s", testEvent.Severity, receivedEvent.Severity)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for broadcasted event")
	}

	// Test multiple clients
	client2 := make(chan DeviceEvent, 5)
	eb.addClient <- client2

	secondEvent := DeviceEvent{
		Type:     "added",
		DeviceID: "new-device-456",
		Name:     "New MIDI Controller",
		Category: "midi_input",
		Severity: "info",
		Message:  "New MIDI device detected",
	}

	eb.BroadcastEvent(secondEvent)

	// Both clients should receive the event
	for i, client := range []chan DeviceEvent{client, client2} {
		select {
		case receivedEvent := <-client:
			if receivedEvent.Type != secondEvent.Type {
				t.Errorf("Client %d: Expected event type %s, got %s", i, secondEvent.Type, receivedEvent.Type)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("Client %d: Timeout waiting for broadcasted event", i)
		}
	}
}

func TestSSEEndpoint(t *testing.T) {
	// Create server
	server := &Server{
		eventBroadcaster: NewEventBroadcaster(),
		port:             8080,
	}
	server.eventBroadcaster.Start()

	// Create test request
	req, err := http.NewRequest("GET", "/api/device-events", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Start SSE handler in goroutine (since it blocks)
	go server.handleDeviceEvents(rr, req)

	// Give it a moment to set up
	time.Sleep(100 * time.Millisecond)

	// Trigger a test event
	testEvent := DeviceEvent{
		Type:      "test",
		DeviceID:  "sse-test-device",
		Name:      "SSE Test Device",
		Category:  "audio_output",
		Severity:  "warning",
		Message:   "This is a test event for SSE",
		Timestamp: time.Now(),
	}

	server.eventBroadcaster.BroadcastEvent(testEvent)

	// Give it time to process
	time.Sleep(100 * time.Millisecond)

	// Check response headers
	if contentType := rr.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %s", contentType)
	}

	if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != "no-cache" {
		t.Errorf("Expected Cache-Control no-cache, got %s", cacheControl)
	}

	// Check that we got some SSE data
	body := rr.Body.String()
	if body == "" {
		t.Error("Expected SSE data in response body, got empty string")
	}

	// Should contain at least the initial "connected" event
	if !strings.Contains(body, "data:") {
		t.Error("Expected SSE data format with 'data:' prefix")
	}

	if !strings.Contains(body, "connected") {
		t.Error("Expected initial 'connected' event in SSE stream")
	}
}

func TestDeviceEventTestEndpoint(t *testing.T) {
	// Create server
	server := &Server{
		eventBroadcaster: NewEventBroadcaster(),
		port:             8080,
	}
	server.eventBroadcaster.Start()

	// Create test event
	testEvent := DeviceEvent{
		Type:     "removed",
		DeviceID: "test-interface-789",
		Name:     "Test Interface",
		Category: "audio_input",
		Severity: "critical",
		Message:  "Critical device removal test",
	}

	// Marshal to JSON
	eventJSON, err := json.Marshal(testEvent)
	if err != nil {
		t.Fatal(err)
	}

	// Create POST request
	req, err := http.NewRequest("POST", "/api/test/device-event", bytes.NewBuffer(eventJSON))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	server.handleTestDeviceEvent(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check response
	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("Expected status success, got %s", response["status"])
	}

	if response["message"] != "Test event broadcasted" {
		t.Errorf("Expected specific message, got %s", response["message"])
	}
}

func TestDeviceEventSeverityLevels(t *testing.T) {
	severityLevels := []string{"info", "warning", "critical"}

	for _, severity := range severityLevels {
		t.Run("severity_"+severity, func(t *testing.T) {
			event := DeviceEvent{
				Type:     "test",
				DeviceID: "test-device",
				Name:     "Test Device",
				Category: "audio_input",
				Severity: severity,
				Message:  "Test message for " + severity + " level",
			}

			// Test that event can be marshaled/unmarshaled
			data, err := json.Marshal(event)
			if err != nil {
				t.Errorf("Failed to marshal %s event: %v", severity, err)
			}

			var unmarshaled DeviceEvent
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Errorf("Failed to unmarshal %s event: %v", severity, err)
			}

			if unmarshaled.Severity != severity {
				t.Errorf("Expected severity %s, got %s", severity, unmarshaled.Severity)
			}
		})
	}
}

func TestConcurrentSSEClients(t *testing.T) {
	// Create event broadcaster
	eb := NewEventBroadcaster()
	eb.Start()

	// Create multiple client channels
	numClients := 5
	clients := make([]chan DeviceEvent, numClients)

	for i := 0; i < numClients; i++ {
		clients[i] = make(chan DeviceEvent, 10)
		eb.addClient <- clients[i]
	}

	// Broadcast multiple events
	numEvents := 3
	testEvents := make([]DeviceEvent, numEvents)

	for i := 0; i < numEvents; i++ {
		testEvents[i] = DeviceEvent{
			Type:     "test",
			DeviceID: "concurrent-test-" + string(rune(i+'0')),
			Name:     "Concurrent Test Device",
			Category: "audio_input",
			Severity: "info",
			Message:  "Concurrent test event",
		}
		eb.BroadcastEvent(testEvents[i])
	}

	// Verify all clients receive all events
	for clientIdx, client := range clients {
		for eventIdx := 0; eventIdx < numEvents; eventIdx++ {
			select {
			case receivedEvent := <-client:
				expectedDeviceID := testEvents[eventIdx].DeviceID
				if receivedEvent.DeviceID != expectedDeviceID {
					t.Errorf("Client %d, Event %d: Expected device ID %s, got %s",
						clientIdx, eventIdx, expectedDeviceID, receivedEvent.DeviceID)
				}
			case <-time.After(1 * time.Second):
				t.Errorf("Client %d: Timeout waiting for event %d", clientIdx, eventIdx)
			}
		}
	}
}
