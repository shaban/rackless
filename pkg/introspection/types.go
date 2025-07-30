package introspection

// Plugin represents an AudioUnit plugin with its parameters
type Plugin struct {
	Name           string      `json:"name"`
	ManufacturerID string      `json:"manufacturerID"`
	Type           string      `json:"type"`
	Subtype        string      `json:"subtype"`
	Parameters     []Parameter `json:"parameters"`
}

// Parameter represents a plugin parameter with full introspection data
type Parameter struct {
	Unit          string   `json:"unit"`
	DisplayName   string   `json:"displayName"`
	Address       uint64   `json:"address"`
	MaxValue      float32  `json:"maxValue"`
	Identifier    string   `json:"identifier"`
	MinValue      float32  `json:"minValue"`
	CanRamp       bool     `json:"canRamp"`
	IsWritable    bool     `json:"isWritable"`
	RawFlags      uint32   `json:"rawFlags"`
	DefaultValue  float32  `json:"defaultValue"`
	CurrentValue  float32  `json:"currentValue"`
	IndexedValues []string `json:"indexedValues,omitempty"`
}

// IntrospectionResult provides query methods for plugin data
type IntrospectionResult []Plugin

// SelectBestPluginForLayout finds the best plugin for demonstration/layout
func (result IntrospectionResult) SelectBestPluginForLayout() *Plugin {
	// Prioritize Neural DSP plugins (known for comprehensive parameter sets)
	for i := range result {
		if result[i].ManufacturerID == "NDSP" && len(result[i].Parameters) > 0 {
			return &result[i]
		}
	}

	// Fall back to any plugin with a good number of parameters
	var bestPlugin *Plugin
	maxParams := 0
	
	for i := range result {
		if len(result[i].Parameters) > maxParams {
			maxParams = len(result[i].Parameters)
			bestPlugin = &result[i]
		}
	}
	
	return bestPlugin
}

// FindPluginByName searches for a plugin by name
func (result IntrospectionResult) FindPluginByName(name string) *Plugin {
	for i := range result {
		if result[i].Name == name {
			return &result[i]
		}
	}
	return nil
}

// GetParameterCount returns total parameters across all plugins
func (result IntrospectionResult) GetParameterCount() int {
	total := 0
	for _, plugin := range result {
		total += len(plugin.Parameters)
	}
	return total
}
