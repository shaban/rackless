package audio

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

// LoadDevices loads audio device information using the standalone devices tool
func LoadDevices() error {
	log.Println("Loading device information...")

	cmd := exec.Command("./standalone/devices/devices")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run devices tool: %v", err)
	}

	err = json.Unmarshal(output, &Data.Devices)
	if err != nil {
		return fmt.Errorf("failed to parse devices JSON: %v", err)
	}

	log.Printf("✅ Loaded %d audio input devices, %d audio output devices, %d MIDI input devices, %d MIDI output devices",
		Data.Devices.TotalAudioInputDevices,
		Data.Devices.TotalAudioOutputDevices,
		Data.Devices.TotalMIDIInputDevices,
		Data.Devices.TotalMIDIOutputDevices)

	return nil
}

// LoadPlugins loads plugin information using the standalone inspector tool
func LoadPlugins() error {
	log.Println("Loading plugin information...")

	cmd := exec.Command("./standalone/inspector/inspector")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run inspector tool: %v", err)
	}

	err = json.Unmarshal(output, &Data.Plugins)
	if err != nil {
		return fmt.Errorf("failed to parse plugins JSON: %v", err)
	}

	log.Printf("✅ Loaded %d AudioUnit plugins", len(Data.Plugins))

	return nil
}
