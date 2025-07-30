//go:build darwin && cgo
// +build darwin,cgo

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// DeviceEvent represents a device state change event
type DeviceEvent struct {
	Type      string    `json:"type"`      // "added", "removed", "changed"
	DeviceID  string    `json:"deviceId"`  // Device identifier
	Name      string    `json:"name"`      // Human-readable device name
	Category  string    `json:"category"`  // "audio_input", "audio_output", "midi_input", "midi_output"
	Severity  string    `json:"severity"`  // "info", "warning", "critical"
	Message   string    `json:"message"`   // User-friendly message
	Timestamp time.Time `json:"timestamp"` // When the event occurred
}

// EventBroadcaster manages SSE connections and broadcasts device events
type EventBroadcaster struct {
	clients   map[chan DeviceEvent]bool
	addClient chan chan DeviceEvent
	rmClient  chan chan DeviceEvent
	broadcast chan DeviceEvent
	mutex     sync.RWMutex
}

func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{
		clients:   make(map[chan DeviceEvent]bool),
		addClient: make(chan chan DeviceEvent),
		rmClient:  make(chan chan DeviceEvent),
		broadcast: make(chan DeviceEvent, 10), // Buffer for events
	}
}

func (eb *EventBroadcaster) Start() {
	go func() {
		for {
			select {
			case client := <-eb.addClient:
				eb.mutex.Lock()
				eb.clients[client] = true
				eb.mutex.Unlock()
				log.Printf("📡 SSE client connected (total: %d)", len(eb.clients))

			case client := <-eb.rmClient:
				eb.mutex.Lock()
				if _, ok := eb.clients[client]; ok {
					delete(eb.clients, client)
					close(client)
				}
				eb.mutex.Unlock()
				log.Printf("📡 SSE client disconnected (total: %d)", len(eb.clients))

			case event := <-eb.broadcast:
				eb.mutex.RLock()
				for client := range eb.clients {
					select {
					case client <- event:
					default:
						// Client is slow/blocked, remove it
						delete(eb.clients, client)
						close(client)
					}
				}
				eb.mutex.RUnlock()
				log.Printf("📡 Broadcasted event: %s - %s", event.Type, event.Name)
			}
		}
	}()
}

func (eb *EventBroadcaster) BroadcastEvent(event DeviceEvent) {
	select {
	case eb.broadcast <- event:
	default:
		log.Printf("⚠️  Event broadcast buffer full, dropping event: %s", event.Type)
	}
}

// Server represents the main application server
type Server struct {
	layoutManager    *LayoutManager
	eventBroadcaster *EventBroadcaster
	settingsManager  *SettingsManager
	port             int
}

func main() {
	log.Println("🎵 MC-SoFX AudioUnit Controller")

	// Initialize settings manager first, with device enumeration for defaults
	settingsManager := NewSettingsManager("data/settings.json", DeviceEnum)
	if err := settingsManager.Load(); err != nil {
		log.Fatalf("Failed to load settings: %v", err)
	}

	// Get port from settings, allow command line override
	settings := settingsManager.Get()
	port := settings.Server.Port
	if len(os.Args) > 1 {
		if p, err := strconv.Atoi(os.Args[1]); err == nil {
			port = p
		}
	}

	log.Printf("Starting server on port %d...", port)

	server := NewServer(port, settingsManager)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}

func NewServer(port int, settingsManager *SettingsManager) *Server {
	// Initialize layout manager
	layoutsDir := "data/layouts"
	layoutManager := NewLayoutManager(layoutsDir)

	// Initialize event broadcaster
	eventBroadcaster := NewEventBroadcaster()
	eventBroadcaster.Start()

	// Load all existing layouts first
	if err := layoutManager.LoadAllLayouts(); err != nil {
		log.Printf("Warning: Failed to load layouts: %v", err)
	}

	// Run AudioUnit introspection to get plugin data (required for layout generation)
	log.Println("🔍 Running AudioUnit introspection...")
	start := time.Now()

	if err := ExecuteIntrospection(); err != nil {
		log.Printf("Warning: Introspection failed: %v", err)
	} else {
		duration := time.Since(start)
		pluginCount := len(IntrospectionData)
		log.Printf("✅ Introspection completed in %v (%d plugins found)", duration, pluginCount)
	}

	// Check if we should load the current layout from settings
	currentLayoutName := settingsManager.GetCurrentLayoutName()
	if currentLayoutName == "sample_layout" || currentLayoutName == "Not Selected" {
		// Ensure we have at least one layout (this will work since introspection is done)
		if err := layoutManager.EnsureDefaultLayout(); err != nil {
			log.Printf("Warning: Failed to ensure default layout: %v", err)
		}

		// Update settings with the default layout name
		layoutNames := layoutManager.ListLayouts()
		if len(layoutNames) > 0 {
			defaultLayoutName := layoutNames[0]
			if err := settingsManager.UpdateCurrentLayout(defaultLayoutName, ""); err != nil {
				log.Printf("Warning: Failed to update current layout in settings: %v", err)
			}
		}
	}

	log.Println("Server initialized - ready to serve requests")

	return &Server{
		layoutManager:    layoutManager,
		eventBroadcaster: eventBroadcaster,
		settingsManager:  settingsManager,
		port:             port,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Static file serving
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("frontend/static/"))))
	mux.Handle("/bin/", http.StripPrefix("/bin/", http.FileServer(http.Dir("bin/"))))

	// API routes
	mux.HandleFunc("GET /api/layouts", s.handleListLayouts)
	mux.HandleFunc("GET /api/layouts/{name}", s.handleGetLayout)
	mux.HandleFunc("PUT /api/layouts/{name}", s.handleUpdateLayout)
	mux.HandleFunc("POST /api/layouts/save", s.handleSaveLayout)
	mux.HandleFunc("GET /api/parameters", s.handleGetParameters)
	mux.HandleFunc("GET /api/plugins", s.handleListPlugins)

	// Settings routes
	mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	mux.HandleFunc("PUT /api/settings", s.handleUpdateSettings)
	mux.HandleFunc("PUT /api/settings/audio/input", s.handleUpdateAudioInput)
	mux.HandleFunc("PUT /api/settings/audio/output", s.handleUpdateAudioOutput)
	mux.HandleFunc("PUT /api/settings/layout/current", s.handleUpdateCurrentLayout)
	mux.HandleFunc("PUT /api/settings/midi/input", s.handleUpdateMIDIInput)

	// Device enumeration routes
	mux.HandleFunc("GET /api/devices", s.handleGetAllDevices)
	mux.HandleFunc("GET /api/devices/audio/input", s.handleGetAudioInputDevices)
	mux.HandleFunc("GET /api/devices/audio/output", s.handleGetAudioOutputDevices)
	mux.HandleFunc("GET /api/devices/midi/input", s.handleGetMIDIInputDevices)
	mux.HandleFunc("GET /api/devices/midi/output", s.handleGetMIDIOutputDevices)

	// Server-Sent Events for device monitoring
	mux.HandleFunc("GET /api/device-events", s.handleDeviceEvents)

	// Test endpoint to trigger device events (for testing)
	mux.HandleFunc("POST /api/test/device-event", s.handleTestDeviceEvent)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	// Serve the main SPA page for all other routes
	mux.HandleFunc("/", s.handleSPA)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Starting server on http://localhost%s", addr)
	log.Printf("Available layouts: %v", s.layoutManager.ListLayouts())

	return http.ListenAndServe(addr, mux)
}

// SPA Handler - serves static HTML for all non-API routes
func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	// Only serve HTML for GET requests to avoid issues with API calls
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	// Skip API routes
	if len(r.URL.Path) > 4 && r.URL.Path[:5] == "/api/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	// Read the Vue template from file
	htmlFile := "frontend/app.html"
	if !filepath.IsAbs(htmlFile) {
		wd, err := os.Getwd()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Printf("Failed to get working directory: %v", err)
			return
		}
		htmlFile = filepath.Join(wd, htmlFile)
	}

	htmlContent, err := os.ReadFile(htmlFile)
	if err != nil {
		http.Error(w, "Failed to load page", http.StatusInternalServerError)
		log.Printf("Failed to read HTML file: %v", err)
		return
	}

	w.Write(htmlContent)
}

// API Handlers

func (s *Server) handleListLayouts(w http.ResponseWriter, r *http.Request) {
	layouts := s.layoutManager.GetAllLayouts()

	type LayoutSummary struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		Version      string `json:"version"`
		GroupCount   int    `json:"groupCount"`
		ControlCount int    `json:"controlCount"`
	}

	summaries := make([]LayoutSummary, 0, len(layouts))
	for _, layout := range layouts {
		controlCount := 0
		for _, group := range layout.Groups {
			controlCount += len(group.Controls)
		}

		summaries = append(summaries, LayoutSummary{
			Name:         layout.Name,
			Description:  layout.Description,
			Version:      layout.Version,
			GroupCount:   len(layout.Groups),
			ControlCount: controlCount,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

func (s *Server) handleGetLayout(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Layout name is required", http.StatusBadRequest)
		return
	}

	layout := s.layoutManager.GetLayout(name)
	if layout == nil {
		http.Error(w, "Layout not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(layout)
}

func (s *Server) handleGetParameters(w http.ResponseWriter, r *http.Request) {
	// Return parameters from the selected plugin in introspection data
	if len(IntrospectionData) == 0 {
		http.Error(w, "No introspection data available", http.StatusInternalServerError)
		return
	}

	// Use the best plugin (same logic as layout generation)
	result := IntrospectionResult(IntrospectionData)
	selectedPlugin := result.SelectBestPluginForLayout()
	if selectedPlugin == nil {
		http.Error(w, "No suitable plugin found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(selectedPlugin.Parameters)
}

func (s *Server) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	type PluginIdentifier struct {
		Type         string `json:"type"`
		Subtype      string `json:"subtype"`
		Manufacturer string `json:"manufacturer"`
		Name         string `json:"name"`
	}

	plugins := make([]PluginIdentifier, 0)

	for _, plugin := range IntrospectionData {
		if len(plugin.Parameters) > 0 { // Only plugins with parameters
			plugins = append(plugins, PluginIdentifier{
				Type:         plugin.Type,
				Subtype:      plugin.Subtype,
				Manufacturer: plugin.ManufacturerID,
				Name:         plugin.Name,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plugins)
}

func (s *Server) handleUpdateLayout(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Layout name is required", http.StatusBadRequest)
		return
	}

	// Get the existing layout
	existingLayout := s.layoutManager.GetLayout(name)
	if existingLayout == nil {
		http.Error(w, "Layout not found", http.StatusNotFound)
		return
	}

	// Parse the update request
	var updateReq struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Version     string `json:"version"`
		Grid        struct {
			Rows    int `json:"rows"`
			Columns int `json:"columns"`
			Gutter  int `json:"gutter"`
		} `json:"grid"`
		Groups []struct {
			ID      string `json:"id"`
			Label   string `json:"label"`
			Order   int    `json:"order"`
			ColSpan int    `json:"colspan"`
			RowSpan int    `json:"rowspan"`
			BGType  string `json:"bgType"`
			BGValue string `json:"bgValue"`
		} `json:"groups"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a new layout with updated values but keep existing controls
	updatedLayout := *existingLayout
	updatedLayout.Name = updateReq.Name
	updatedLayout.Description = updateReq.Description
	updatedLayout.Version = updateReq.Version
	updatedLayout.Grid.Rows = updateReq.Grid.Rows
	updatedLayout.Grid.Columns = updateReq.Grid.Columns
	updatedLayout.Grid.Gutter = updateReq.Grid.Gutter

	// Update group properties while preserving controls
	for i, group := range updatedLayout.Groups {
		for _, updateGroup := range updateReq.Groups {
			if group.ID == updateGroup.ID {
				updatedLayout.Groups[i].Label = updateGroup.Label
				updatedLayout.Groups[i].Order = updateGroup.Order
				updatedLayout.Groups[i].ColSpan = updateGroup.ColSpan
				updatedLayout.Groups[i].RowSpan = updateGroup.RowSpan
				updatedLayout.Groups[i].BGType = BackgroundType(updateGroup.BGType)
				updatedLayout.Groups[i].BGValue = updateGroup.BGValue
				break
			}
		}
	}

	// Save the updated layout
	filename := name + ".json"
	if err := s.layoutManager.SaveLayout(&updatedLayout, filename); err != nil {
		http.Error(w, "Failed to save layout: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Reload the layout in the manager
	if err := s.layoutManager.LoadAllLayouts(); err != nil {
		log.Printf("Warning: Failed to reload layouts after update: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&updatedLayout)
}

// Handle saving new or existing layouts
func (s *Server) handleSaveLayout(w http.ResponseWriter, r *http.Request) {
	var saveReq struct {
		Name   string      `json:"name"`
		Layout interface{} `json:"layout"`
	}

	if err := json.NewDecoder(r.Body).Decode(&saveReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if saveReq.Name == "" {
		http.Error(w, "Layout name is required", http.StatusBadRequest)
		return
	}

	// Convert the generic layout interface to JSON and back to Layout struct
	layoutBytes, err := json.Marshal(saveReq.Layout)
	if err != nil {
		http.Error(w, "Failed to process layout data", http.StatusBadRequest)
		return
	}

	var layout Layout
	if err := json.Unmarshal(layoutBytes, &layout); err != nil {
		http.Error(w, "Invalid layout structure", http.StatusBadRequest)
		return
	}

	// Set the name from the request
	layout.Name = saveReq.Name

	// Save the layout
	filename := saveReq.Name + ".json"
	if err := s.layoutManager.SaveLayout(&layout, filename); err != nil {
		http.Error(w, "Failed to save layout: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update settings to set this as the current layout
	if err := s.settingsManager.UpdateCurrentLayout(saveReq.Name, ""); err != nil {
		log.Printf("Warning: Failed to update current layout in settings: %v", err)
	}

	// Reload layouts to include the new one
	if err := s.layoutManager.LoadAllLayouts(); err != nil {
		log.Printf("Warning: Failed to reload layouts after save: %v", err)
	}

	log.Printf("✅ Saved layout: %s", saveReq.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"name":    saveReq.Name,
		"message": "Layout saved successfully",
	})
}

// Device enumeration handlers

func (s *Server) handleGetAllDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := DeviceEnum.GetAllDevices()
	if err != nil {
		log.Printf("Error getting all devices: %v", err)
		http.Error(w, "Failed to enumerate devices", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

func (s *Server) handleGetAudioInputDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := DeviceEnum.GetAudioInputDevices()
	if err != nil {
		log.Printf("Error getting audio input devices: %v", err)
		http.Error(w, "Failed to enumerate audio input devices", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

func (s *Server) handleGetAudioOutputDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := DeviceEnum.GetAudioOutputDevices()
	if err != nil {
		log.Printf("Error getting audio output devices: %v", err)
		http.Error(w, "Failed to enumerate audio output devices", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

func (s *Server) handleGetMIDIInputDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := DeviceEnum.GetMIDIInputDevices()
	if err != nil {
		log.Printf("Error getting MIDI input devices: %v", err)
		http.Error(w, "Failed to enumerate MIDI input devices", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

func (s *Server) handleGetMIDIOutputDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := DeviceEnum.GetMIDIOutputDevices()
	if err != nil {
		log.Printf("Error getting MIDI output devices: %v", err)
		http.Error(w, "Failed to enumerate MIDI output devices", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

// handleDeviceEvents serves Server-Sent Events for device monitoring
func (s *Server) handleDeviceEvents(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	client := make(chan DeviceEvent)
	s.eventBroadcaster.addClient <- client

	// Send initial connection event
	initialEvent := DeviceEvent{
		Type:      "connected",
		DeviceID:  "sse-client",
		Name:      "SSE Connection",
		Category:  "system",
		Severity:  "info",
		Message:   "Device event monitoring connected",
		Timestamp: time.Now(),
	}

	eventData, _ := json.Marshal(initialEvent)
	fmt.Fprintf(w, "data: %s\n\n", eventData)
	w.(http.Flusher).Flush()

	// Listen for events and client disconnect
	for {
		select {
		case event := <-client:
			eventData, err := json.Marshal(event)
			if err != nil {
				log.Printf("Error marshaling device event: %v", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", eventData)
			w.(http.Flusher).Flush()

		case <-r.Context().Done():
			s.eventBroadcaster.rmClient <- client
			return
		}
	}
}

// handleTestDeviceEvent allows triggering test device events for development
func (s *Server) handleTestDeviceEvent(w http.ResponseWriter, r *http.Request) {
	var event DeviceEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid event data", http.StatusBadRequest)
		return
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Broadcast the test event
	s.eventBroadcaster.BroadcastEvent(event)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Test event broadcasted",
	})
}

// Settings API handlers

// handleGetSettings returns the current application settings
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings := s.settingsManager.Get()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(settings); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode settings: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleUpdateSettings updates the entire settings object
func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var newSettings Settings
	if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Update settings using the provided data
	if err := s.settingsManager.Update(func(settings *Settings) {
		settings.Version = newSettings.Version
		settings.Audio = newSettings.Audio
		settings.Layout = newSettings.Layout
		settings.UI = newSettings.UI
		settings.MIDI = newSettings.MIDI
		settings.Server = newSettings.Server
		settings.FirstRun = newSettings.FirstRun
		// LastModified will be set automatically by the Update method
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update settings: %v", err), http.StatusInternalServerError)
		return
	}

	// Return updated settings
	s.handleGetSettings(w, r)
}

// handleUpdateAudioInput updates the audio input device
func (s *Server) handleUpdateAudioInput(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID   *string `json:"deviceId"`
		DeviceName string  `json:"deviceName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.settingsManager.UpdateAudioInput(req.DeviceID, req.DeviceName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update audio input: %v", err), http.StatusInternalServerError)
		return
	}

	// Return updated settings
	s.handleGetSettings(w, r)
}

// handleUpdateAudioOutput updates the audio output device
func (s *Server) handleUpdateAudioOutput(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID   *string `json:"deviceId"`
		DeviceName string  `json:"deviceName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.settingsManager.UpdateAudioOutput(req.DeviceID, req.DeviceName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update audio output: %v", err), http.StatusInternalServerError)
		return
	}

	// Return updated settings
	s.handleGetSettings(w, r)
}

// handleUpdateCurrentLayout updates the current layout
func (s *Server) handleUpdateCurrentLayout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LayoutName string `json:"layoutName"`
		LayoutPath string `json:"layoutPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.LayoutName == "" {
		http.Error(w, "Layout name is required", http.StatusBadRequest)
		return
	}

	// Verify layout exists
	if s.layoutManager.GetLayout(req.LayoutName) == nil {
		http.Error(w, fmt.Sprintf("Layout '%s' not found", req.LayoutName), http.StatusNotFound)
		return
	}

	if err := s.settingsManager.UpdateCurrentLayout(req.LayoutName, req.LayoutPath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update current layout: %v", err), http.StatusInternalServerError)
		return
	}

	// Return updated settings
	s.handleGetSettings(w, r)
}

// handleUpdateMIDIInput updates the MIDI input device
func (s *Server) handleUpdateMIDIInput(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID   *string `json:"deviceId"`
		DeviceName string  `json:"deviceName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.settingsManager.UpdateMIDIInput(req.DeviceID, req.DeviceName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update MIDI input: %v", err), http.StatusInternalServerError)
		return
	}

	// Return updated settings
	s.handleGetSettings(w, r)
}
