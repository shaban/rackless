package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/shaban/rackless/pkg/devices"
)

func main() {
	log.Println("🎛️  Device Enumeration Test - Rackless Audio System")
	log.Println("====================================================")

	// Create device enumerator
	enumerator := devices.NewDeviceEnumerator()

	// Test comprehensive device enumeration
	start := time.Now()
	result, err := enumerator.GetAllDevices()
	if err != nil {
		log.Fatalf("❌ Device enumeration failed: %v", err)
	}

	log.Printf("✅ Device enumeration completed in %v", time.Since(start))
	log.Printf("📊 Results: %d audio inputs, %d audio outputs, %d MIDI inputs, %d MIDI outputs",
		len(result.AudioInputs), len(result.AudioOutputs), len(result.MIDIInputs), len(result.MIDIOutputs))

	// Display summary
	fmt.Println("\n🎤 Audio Input Devices:")
	for i, device := range result.AudioInputs {
		fmt.Printf("  %d. %s (%d channels, ID: %d)\n", i+1, device.Name, device.ChannelCount, device.DeviceID)
		if len(device.SupportedSampleRates) > 0 {
			fmt.Printf("     Sample rates: %v Hz\n", device.SupportedSampleRates)
		}
		if len(device.SupportedBitDepths) > 0 {
			fmt.Printf("     Bit depths: %v bit\n", device.SupportedBitDepths)
		}
	}

	fmt.Println("\n🔊 Audio Output Devices:")
	for i, device := range result.AudioOutputs {
		fmt.Printf("  %d. %s (%d channels, ID: %d)\n", i+1, device.Name, device.ChannelCount, device.DeviceID)
		if len(device.SupportedSampleRates) > 0 {
			fmt.Printf("     Sample rates: %v Hz\n", device.SupportedSampleRates)
		}
		if len(device.SupportedBitDepths) > 0 {
			fmt.Printf("     Bit depths: %v bit\n", device.SupportedBitDepths)
		}
	}

	fmt.Println("\n🎹 MIDI Input Devices:")
	for i, device := range result.MIDIInputs {
		fmt.Printf("  %d. %s (ID: %d, Online: %t)\n", i+1, device.Name, device.EndpointID, device.IsOnline)
	}

	fmt.Println("\n🎹 MIDI Output Devices:")
	for i, device := range result.MIDIOutputs {
		fmt.Printf("  %d. %s (ID: %d, Online: %t)\n", i+1, device.Name, device.EndpointID, device.IsOnline)
	}

	fmt.Printf("\n⚙️  Default Devices: Input ID %d, Output ID %d\n", 
		result.DefaultDevices.DefaultInput, result.DefaultDevices.DefaultOutput)

	// Save full JSON output if requested
	if len(os.Args) > 1 && os.Args[1] == "-json" {
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Printf("❌ Failed to marshal JSON: %v", err)
			return
		}

		filename := fmt.Sprintf("device-enumeration-%d.json", time.Now().Unix())
		if err := os.WriteFile(filename, jsonData, 0644); err != nil {
			log.Printf("❌ Failed to write JSON file: %v", err)
			return
		}

		log.Printf("💾 Full device enumeration saved to %s", filename)
		fmt.Printf("\n📄 JSON size: %.1f KB\n", float64(len(jsonData))/1024)
	}

	log.Println("✅ Device enumeration test completed successfully")
}
