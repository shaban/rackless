package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

// Device structures based on standalone/devices output
type AudioDevice struct {
	DeviceID             int    `json:"deviceId"`
	UID                  string `json:"uid"`
	SupportedSampleRates []int  `json:"supportedSampleRates"`
	ChannelCount         int    `json:"channelCount"`
	IsDefault            bool   `json:"isDefault"`
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

var serverData ServerData

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

	// Static file serving (for WASM app)
	fs := http.FileServer(http.Dir("./web/static/"))
	mux.Handle("GET /", fs)

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
	log.Println("   • GET / - Static file serving (web app)")

	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
