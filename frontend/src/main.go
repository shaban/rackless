//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"syscall/js"
)

// Device and Plugin types matching the server API
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

type DevicesData struct {
	TotalMIDIInputDevices   int           `json:"totalMIDIInputDevices"`
	MIDIInput               []MIDIDevice  `json:"midiInput"`
	TotalAudioInputDevices  int           `json:"totalAudioInputDevices"`
	AudioInput              []AudioDevice `json:"audioInput"`
	AudioOutput             []AudioDevice `json:"audioOutput"`
	TotalMIDIOutputDevices  int           `json:"totalMIDIOutputDevices"`
	Timestamp               string        `json:"timestamp"`
	MIDIOutput              []MIDIDevice  `json:"midiOutput"`
	TotalAudioOutputDevices int           `json:"totalAudioOutputDevices"`
	DefaultSampleRate       float64       `json:"defaultSampleRate"`
}

// Global data
var devices DevicesData
var plugins []Plugin

// Fetch data from the server
func fetchData() {
	// Fetch devices
	resp, err := http.Get("http://localhost:8080/api/devices")
	if err != nil {
		fmt.Printf("Error fetching devices: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		fmt.Printf("Error decoding devices: %v\n", err)
		return
	}

	// Fetch plugins
	resp, err = http.Get("http://localhost:8080/api/plugins")
	if err != nil {
		fmt.Printf("Error fetching plugins: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&plugins); err != nil {
		fmt.Printf("Error decoding plugins: %v\n", err)
		return
	}

	fmt.Printf("✅ Loaded %d audio devices and %d plugins\n", 
		devices.TotalAudioInputDevices+devices.TotalAudioOutputDevices, len(plugins))
}

// Update the UI with loaded data
func updateUI() {
	doc := js.Global().Get("document")
	
	// Update devices section
	devicesDiv := doc.Call("getElementById", "devices")
	if !devicesDiv.IsNull() {
		html := "<h3>Audio Devices</h3><ul>"
		for _, device := range devices.AudioInput {
			html += fmt.Sprintf("<li>%s (Input, %d channels)</li>", device.Name, device.ChannelCount)
		}
		for _, device := range devices.AudioOutput {
			html += fmt.Sprintf("<li>%s (Output, %d channels)</li>", device.Name, device.ChannelCount)
		}
		html += "</ul>"
		devicesDiv.Set("innerHTML", html)
	}

	// Update plugins section
	pluginsDiv := doc.Call("getElementById", "plugins")
	if !pluginsDiv.IsNull() {
		html := fmt.Sprintf("<h3>Plugins (%d total)</h3><ul>", len(plugins))
		for i, plugin := range plugins {
			if i < 10 { // Show first 10 plugins
				html += fmt.Sprintf("<li>%s (%d parameters)</li>", plugin.Name, len(plugin.Parameters))
			}
		}
		if len(plugins) > 10 {
			html += fmt.Sprintf("<li>... and %d more plugins</li>", len(plugins)-10)
		}
		html += "</ul>"
		pluginsDiv.Set("innerHTML", html)
	}
}

// JavaScript function exports
func loadData(this js.Value, args []js.Value) interface{} {
	go func() {
		fetchData()
		updateUI()
	}()
	return nil
}

func main() {
	fmt.Println("🎵 Rackless WASM Frontend Starting...")

	// Export functions to JavaScript
	js.Global().Set("loadData", js.FuncOf(loadData))

	// Initial data load
	go func() {
		fetchData()
		updateUI()
	}()

	fmt.Println("✅ Rackless WASM Frontend Ready")

	// Keep the program running
	select {}
}
