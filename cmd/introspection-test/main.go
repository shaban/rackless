package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/shaban/rackless/pkg/introspection"
)

func main() {
	log.Println("Rackless AudioUnit Introspection Test")
	log.Println("=====================================")

	// Get AudioUnit plugins
	plugins, err := introspection.GetAudioUnits()
	if err != nil {
		log.Fatalf("Failed to get AudioUnits: %v", err)
	}

	log.Printf("Found %d AudioUnit plugins", len(plugins))
	log.Printf("Total parameters across all plugins: %d", plugins.GetParameterCount())

	// Show summary
	for i, plugin := range plugins {
		if i >= 5 { // Limit output for readability
			log.Printf("... and %d more plugins", len(plugins)-5)
			break
		}
		log.Printf("Plugin %d: %s (%s) - %d parameters",
			i+1, plugin.Name, plugin.ManufacturerID, len(plugin.Parameters))
	}

	// Find the best plugin for demonstration
	bestPlugin := plugins.SelectBestPluginForLayout()
	if bestPlugin == nil {
		log.Println("No suitable plugin found for demonstration")
		return
	}

	log.Printf("\nBest plugin for demonstration: %s", bestPlugin.Name)
	log.Printf("Manufacturer: %s", bestPlugin.ManufacturerID)
	log.Printf("Parameters: %d", len(bestPlugin.Parameters))

	// Show first few parameters
	log.Println("\nSample parameters:")
	for i, param := range bestPlugin.Parameters {
		if i >= 3 { // Show first 3 parameters
			log.Printf("... and %d more parameters", len(bestPlugin.Parameters)-3)
			break
		}
		log.Printf("  %s: %s (%.2f - %.2f, default: %.2f)",
			param.Identifier, param.DisplayName,
			param.MinValue, param.MaxValue, param.DefaultValue)

		if len(param.IndexedValues) > 0 {
			log.Printf("    Indexed values: %v", param.IndexedValues)
		}
	}

	// Optional: Save full introspection data to file
	if len(os.Args) > 1 && os.Args[1] == "--save" {
		filename := "introspection_data.json"
		data, err := json.MarshalIndent(plugins, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
			return
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			log.Printf("Failed to write file: %v", err)
			return
		}

		log.Printf("Full introspection data saved to %s", filename)
	}

	log.Println("\nIntrospection test completed successfully! 🎯")
}
