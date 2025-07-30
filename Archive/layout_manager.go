// Package layout provides functionality for managing control layouts
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LayoutManager handles loading, saving, and managing control layouts
type LayoutManager struct {
	layoutsDir  string
	layouts     map[string]*Layout // Keyed by layout name
	uuidCounter int                // Simple counter for UUID generation
}

// NewLayoutManager creates a new layout manager
func NewLayoutManager(layoutsDir string) *LayoutManager {
	return &LayoutManager{
		layoutsDir:  layoutsDir,
		layouts:     make(map[string]*Layout),
		uuidCounter: 1000,
	}
}

// LoadLayout loads a layout from a JSON file
func (lm *LayoutManager) LoadLayout(filename string) (*Layout, error) {
	filePath := filepath.Join(lm.layoutsDir, filename)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open layout file %s: %w", filename, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read layout file %s: %w", filename, err)
	}

	var layout Layout
	if err := json.Unmarshal(data, &layout); err != nil {
		return nil, fmt.Errorf("failed to parse layout file %s: %w", filename, err)
	}

	// Validate the layout
	if err := lm.ValidateLayout(&layout); err != nil {
		return nil, fmt.Errorf("invalid layout in file %s: %w", filename, err)
	}

	// Cache the layout
	lm.layouts[layout.Name] = &layout

	return &layout, nil
}

// SaveLayout saves a layout to a JSON file
func (lm *LayoutManager) SaveLayout(layout *Layout, filename string) error {
	// Validate the layout before saving
	if err := lm.ValidateLayout(layout); err != nil {
		return fmt.Errorf("cannot save invalid layout: %w", err)
	}

	filePath := filepath.Join(lm.layoutsDir, filename)

	// Ensure directory exists
	if err := os.MkdirAll(lm.layoutsDir, 0755); err != nil {
		return fmt.Errorf("failed to create layouts directory: %w", err)
	}

	data, err := json.MarshalIndent(layout, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal layout: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write layout file: %w", err)
	}

	// Cache the layout
	lm.layouts[layout.Name] = layout

	return nil
}

// LoadAllLayouts loads all layout files from the layouts directory
func (lm *LayoutManager) LoadAllLayouts() error {
	files, err := os.ReadDir(lm.layoutsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist yet, that's OK
		}
		return fmt.Errorf("failed to read layouts directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		if _, err := lm.LoadLayout(file.Name()); err != nil {
			return fmt.Errorf("failed to load layout %s: %w", file.Name(), err)
		}
	}

	return nil
}

// GetLayout returns a cached layout by name
func (lm *LayoutManager) GetLayout(name string) *Layout {
	return lm.layouts[name]
}

// GetAllLayouts returns all cached layouts
func (lm *LayoutManager) GetAllLayouts() map[string]*Layout {
	// Return a copy to prevent external modification
	layouts := make(map[string]*Layout)
	for name, layout := range lm.layouts {
		layouts[name] = layout
	}
	return layouts
}

// ListLayouts returns a list of all layout names
func (lm *LayoutManager) ListLayouts() []string {
	names := make([]string, 0, len(lm.layouts))
	for name := range lm.layouts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// EnsureDefaultLayout ensures at least one layout exists, creating a default one if necessary
func (lm *LayoutManager) EnsureDefaultLayout() error {
	// If we already have layouts, we're good
	if len(lm.layouts) > 0 {
		return nil
	}

	// Generate a comprehensive demo/test layout (kitchen sink approach)
	if len(IntrospectionData) == 0 {
		return fmt.Errorf("no introspection data available to generate default layout")
	}

	// Find the best plugin to use for layout generation
	introspectionResult := IntrospectionResult(IntrospectionData)
	selectedPlugin := introspectionResult.SelectBestPluginForLayout()
	if selectedPlugin == nil {
		return fmt.Errorf("no suitable plugin found for layout generation")
	}

	// Generate comprehensive kitchen sink layout (in-memory only, no disk persistence)
	layoutName := fmt.Sprintf("%s - Demo Layout", selectedPlugin.Name)
	defaultLayout := lm.GenerateLayoutFromIntrospection(selectedPlugin, layoutName)
	defaultLayout.Description = fmt.Sprintf("Comprehensive demo/test layout for %s - exercises all parameter types and UI components", selectedPlugin.Name)

	// Store in memory only (no SaveLayout call - this is just a demo/test layout)
	lm.layouts[defaultLayout.Name] = defaultLayout

	log.Printf("Generated demo layout '%s' with %d groups and %d total controls", defaultLayout.Name, len(defaultLayout.Groups), lm.countTotalControls(defaultLayout))
	return nil
}

// countTotalControls counts the total number of controls in a layout
func (lm *LayoutManager) countTotalControls(layout *Layout) int {
	total := 0
	for _, group := range layout.Groups {
		total += len(group.Controls)
	}
	return total
}

// ValidateLayout validates a layout structure
func (lm *LayoutManager) ValidateLayout(layout *Layout) error {
	if layout.Name == "" {
		return fmt.Errorf("layout name cannot be empty")
	}

	if layout.Version == "" {
		return fmt.Errorf("layout version cannot be empty")
	}

	if !layout.Grid.IsValid() {
		return fmt.Errorf("invalid grid configuration")
	}

	// Validate groups
	groupIDs := make(map[string]bool)
	for i, group := range layout.Groups {
		if group.ID == "" {
			return fmt.Errorf("group %d: ID cannot be empty", i)
		}

		if groupIDs[group.ID] {
			return fmt.Errorf("group %d: duplicate ID %s", i, group.ID)
		}
		groupIDs[group.ID] = true

		// Validate group position
		if group.X < 0 || group.X >= layout.Grid.Columns {
			return fmt.Errorf("group %s: invalid X position %d", group.ID, group.X)
		}
		if group.Y < 0 || group.Y >= layout.Grid.Rows {
			return fmt.Errorf("group %s: invalid Y position %d", group.ID, group.Y)
		}

		// Validate controls within group
		controlIDs := make(map[string]bool)
		for j, control := range group.Controls {
			if control.ID == "" {
				return fmt.Errorf("group %s, control %d: ID cannot be empty", group.ID, j)
			}

			if controlIDs[control.ID] {
				return fmt.Errorf("group %s, control %d: duplicate ID %s", group.ID, j, control.ID)
			}
			controlIDs[control.ID] = true

			// Validate control type and implementation
			if err := lm.validateControlType(control); err != nil {
				return fmt.Errorf("group %s, control %s: %w", group.ID, control.ID, err)
			}

			// Validate targets
			if len(control.Targets) == 0 {
				return fmt.Errorf("group %s, control %s: must have at least one target", group.ID, control.ID)
			}

			for k, target := range control.Targets {
				if err := lm.validateTarget(target); err != nil {
					return fmt.Errorf("group %s, control %s, target %d: %w", group.ID, control.ID, k, err)
				}
			}
		}
	}

	return nil
}

// validateControlType validates control type and implementation combinations
func (lm *LayoutManager) validateControlType(control Control) error {
	switch control.Type {
	case ControlSwitch:
		switch control.Implementation {
		case string(SwitchToggleButton), string(SwitchCheckbox):
			return nil
		default:
			return fmt.Errorf("invalid switch implementation: %s", control.Implementation)
		}
	case ControlRadio:
		switch control.Implementation {
		case string(RadioGroup), string(RadioToggleButtons), string(RadioSteppedSlider):
			return nil
		default:
			return fmt.Errorf("invalid radio implementation: %s", control.Implementation)
		}
	case ControlRange:
		switch control.Implementation {
		case string(RangeRotaryKnob), string(RangeRangedSlider):
			return nil
		default:
			return fmt.Errorf("invalid range implementation: %s", control.Implementation)
		}
	default:
		return fmt.Errorf("invalid control type: %s", control.Type)
	}
}

// validateTarget validates a target configuration
func (lm *LayoutManager) validateTarget(target Target) error {
	hasParameter := target.ParameterAddress >= 0 || target.ParameterName != ""
	hasMIDI := target.CCMidi > 0

	if !hasParameter && !hasMIDI {
		return fmt.Errorf("target must have either parameter or MIDI assignment")
	}

	if hasMIDI {
		if target.CCMidi < 1 || target.CCMidi > 127 {
			return fmt.Errorf("MIDI CC must be between 1 and 127, got %d", target.CCMidi)
		}
		if target.Channel < 1 || target.Channel > 16 {
			return fmt.Errorf("MIDI channel must be between 1 and 16, got %d", target.Channel)
		}
	}

	return nil
}

// GenerateLayoutFromIntrospection creates a basic layout from introspection data
func (lm *LayoutManager) GenerateLayoutFromIntrospection(plugin *Plugin, name string) *Layout {
	layout := &Layout{
		Name:        name,
		Description: fmt.Sprintf("Auto-generated layout for %s", plugin.Name),
		Version:     "1.0.0",
		Grid: Grid{
			Rows:    3,
			Columns: 4,
			Gutter:  10,
		},
		Groups: []Group{},
	}

	// Group parameters by category based on naming patterns
	groups := lm.categorizeParameters(plugin.Parameters)

	x, y := 0, 0
	order := 1

	for groupName, params := range groups {
		group := Group{
			Label:    groupName,
			ID:       lm.generateUUID(),
			BGType:   BackgroundColor,
			BGValue:  "#2a2a2a",
			Order:    order,
			ColSpan:  1,
			RowSpan:  1,
			X:        x,
			Y:        y,
			Controls: []Control{},
		}

		// Create controls for parameters in this group
		controlX, controlY := 10, 10
		for _, param := range params {
			control := lm.createControlFromParameter(param)
			control.X = controlX
			control.Y = controlY
			group.Controls = append(group.Controls, control)

			// Position next control
			controlX += 70
			if controlX > 200 {
				controlX = 10
				controlY += 70
			}
		}

		layout.Groups = append(layout.Groups, group)

		// Position next group
		x++
		if x >= layout.Grid.Columns {
			x = 0
			y++
		}
		order++
	}

	return layout
}

// categorizeParameters groups parameters by their names/functionality
func (lm *LayoutManager) categorizeParameters(params []Parameter) map[string][]Parameter {
	groups := make(map[string][]Parameter)

	for _, param := range params {
		category := lm.categorizeParameter(param)
		groups[category] = append(groups[category], param)
	}

	return groups
}

// categorizeParameter determines which category a parameter belongs to
// This creates a comprehensive "kitchen sink" categorization for demo/testing purposes
func (lm *LayoutManager) categorizeParameter(param Parameter) string {
	name := strings.ToLower(param.DisplayName)

	switch {
	// Input/Output section
	case strings.Contains(name, "input gain") || strings.Contains(name, "output gain"):
		return "Input/Output"
	case strings.Contains(name, "gate"):
		return "Gate"
	case strings.Contains(name, "transpose"):
		return "Utility"

	// Compressor section
	case strings.Contains(name, "compressor") || strings.Contains(name, "comp"):
		return "Compressor"

	// Pre-effects (overdrive/distortion)
	case strings.Contains(name, "overdrive1"):
		return "Overdrive 1"
	case strings.Contains(name, "overdrive2"):
		return "Overdrive 2"

	// Amplifier sections (separate group for each amp)
	case strings.Contains(name, "ac20") && !strings.Contains(name, "eq"):
		return "AC20 Amp"
	case strings.Contains(name, "pr12") && !strings.Contains(name, "eq"):
		return "PR12 Amp"
	case strings.Contains(name, "sw50r") && !strings.Contains(name, "eq"):
		return "SW50R Amp"
	case strings.Contains(name, "amp type") || strings.Contains(name, "active amp"):
		return "Amp Selection"

	// EQ sections (separate group for each amp's EQ)
	case strings.Contains(name, "ac20 eq") || (strings.Contains(name, "ac20") && strings.Contains(name, "eq")):
		return "AC20 EQ"
	case strings.Contains(name, "pr12 eq") || (strings.Contains(name, "pr12") && strings.Contains(name, "eq")):
		return "PR12 EQ"
	case strings.Contains(name, "sw50r eq") || (strings.Contains(name, "sw50r") && strings.Contains(name, "eq")):
		return "SW50R EQ"
	case strings.Contains(name, "eq") && strings.Contains(name, "active"):
		return "EQ Control"

	// Cabinet section
	case strings.Contains(name, "cab") || strings.Contains(name, "mic"):
		return "Cabinet"
	case strings.Contains(name, "active cab"):
		return "Cabinet Control"

	// Post-effects
	case strings.Contains(name, "delay"):
		return "Delay"
	case strings.Contains(name, "reverb"):
		return "Reverb"
	case strings.Contains(name, "tremolo"):
		return "Tremolo"
	case strings.Contains(name, "doubler"):
		return "Doubler"

	// Room/ambience
	case strings.Contains(name, "room"):
		return "Room"

	// Presets and controls
	case strings.Contains(name, "preset"):
		return "Presets"
	case strings.Contains(name, "active") && (strings.Contains(name, "pre fx") || strings.Contains(name, "post fx")):
		return "FX Control"

	// Catch-all for anything we missed
	default:
		return "Other Controls"
	}
}

// createControlFromParameter creates a control from an introspection parameter
func (lm *LayoutManager) createControlFromParameter(param Parameter) Control {
	control := Control{
		Label: param.DisplayName,
		ID:    lm.generateUUID(),
		Targets: []Target{
			{
				ParameterAddress: int(param.Address),
				ParameterName:    param.DisplayName,
			},
		},
		Width:  50,
		Height: 50,
	}

	// Determine control type based on parameter unit
	switch param.Unit {
	case "Boolean":
		control.Type = ControlSwitch
		control.Implementation = string(SwitchToggleButton)
		control.Height = 30
	case "Indexed":
		control.Type = ControlRadio
		control.Implementation = string(RadioToggleButtons)
		control.Width = 80
		control.Height = 30
		control.Targets[0].Stepped = true
	default: // Generic
		control.Type = ControlRange
		control.Implementation = string(RangeRotaryKnob)
		control.MinValue = float64(param.MinValue)
		control.MaxValue = float64(param.MaxValue)
	}

	return control
}

// generateUUID generates a simple UUID (this is a placeholder - use a proper UUID library in production)
func (lm *LayoutManager) generateUUID() string {
	// This is a simple placeholder - in production, use github.com/google/uuid or similar
	lm.uuidCounter++
	return fmt.Sprintf("550e8400-e29b-41d4-a716-%012d", lm.uuidCounter)
}
