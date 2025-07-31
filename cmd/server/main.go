package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
)

type AudioDevice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type MIDIDevice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Plugin struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Vendor     string            `json:"vendor"`
	Category   string            `json:"category"`
	Parameters []PluginParameter `json:"parameters"`
}

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

func loadDevices() ([]AudioDevice, []MIDIDevice, error) {
	// Path to devices executable from cmd/server directory
	execPath := "../../standalone/devices/devices"

	cmd := exec.Command(execPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to run devices scanner at %s: %v", execPath, err)
	}

	log.Printf("Devices output: %s", string(output))

	// For now, return empty slices - we'll parse the JSON output later
	var audioDevices []AudioDevice
	var midiDevices []MIDIDevice

	return audioDevices, midiDevices, nil
}

func loadPlugins() ([]Plugin, error) {
	// Path to inspector executable from cmd/server directory
	execPath := "../../standalone/inspector/inspector"

	cmd := exec.Command(execPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run plugin scanner at %s: %v", execPath, err)
	}

	log.Printf("Plugins output: %s", string(output))

	// For now, return empty slice - we'll parse the JSON output later
	var plugins []Plugin

	return plugins, nil
}

func devicesHandler(w http.ResponseWriter, r *http.Request) {
	audioDevices, midiDevices, err := loadDevices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"audio": audioDevices,
		"midi":  midiDevices,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func pluginsHandler(w http.ResponseWriter, r *http.Request) {
	plugins, err := loadPlugins()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plugins)
}

func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/devices", devicesHandler)
	mux.HandleFunc("/api/plugins", pluginsHandler)

	// Static file serving
	staticDir := "../../web/static/"
	if _, err := os.Stat(staticDir); err == nil {
		mux.Handle("/static/", http.StripPrefix("/static/",
			http.FileServer(http.Dir(staticDir))))
	}

	// Default route
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Rackless Audio Plugin Server")
	})

	return mux
}

func main() {
	// Check if standalone executables exist
	devicesPath := "../../standalone/devices/devices"
	inspectorPath := "../../standalone/inspector/inspector"

	if _, err := os.Stat(devicesPath); os.IsNotExist(err) {
		log.Printf("Warning: devices scanner not found at %s", devicesPath)
	}

	if _, err := os.Stat(inspectorPath); os.IsNotExist(err) {
		log.Printf("Warning: plugin inspector not found at %s", inspectorPath)
	}

	router := setupRoutes()

	port := ":8080"
	log.Printf("Starting server on port %s", port)
	log.Printf("Working directory: %s", func() string {
		wd, _ := os.Getwd()
		return wd
	}())

	if err := http.ListenAndServe(port, router); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
