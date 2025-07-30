// Package main defines the control layout format for the MC-SoFX Controller
package main

// Grid defines the overall layout grid for control groups
type Grid struct {
	Rows    int `json:"rows" validate:"min=1,max=5"`    // 1-5 rows
	Columns int `json:"columns" validate:"min=1,max=5"` // 1-5 columns
	Gutter  int `json:"gutter"`                         // Gutter in pixels
}

// BackgroundType defines the type of background for a group
type BackgroundType string

const (
	BackgroundColor BackgroundType = "color"
	BackgroundImage BackgroundType = "image"
)

// BackgroundSize defines how background images are sized
type BackgroundSize string

const (
	BackgroundContain    BackgroundSize = "contain"
	BackgroundCover      BackgroundSize = "cover"
	BackgroundPercentage BackgroundSize = "percentage" // For tile-based percentages
)

// Group represents a collection of controls with shared styling and layout
type Group struct {
	Label    string         `json:"label"`
	ID       string         `json:"id"` // UUID
	BGType   BackgroundType `json:"bgType"`
	BGSize   BackgroundSize `json:"bgSize,omitempty"`
	BGValue  string         `json:"bgValue,omitempty"` // Color hex or image URL/path
	Order    int            `json:"order"`             // Display/rendering order
	ColSpan  int            `json:"colspan,omitempty"` // Takes more than 1 column
	RowSpan  int            `json:"rowspan,omitempty"` // Expands vertically
	Controls []Control      `json:"controls"`          // Controls within this group
	X        int            `json:"x"`                 // Grid position X
	Y        int            `json:"y"`                 // Grid position Y
}

// ControlType defines the type of control
type ControlType string

const (
	ControlSwitch ControlType = "switch"
	ControlRadio  ControlType = "radio"
	ControlRange  ControlType = "range"
)

// SwitchImplementation defines how a switch control is rendered
type SwitchImplementation string

const (
	SwitchToggleButton SwitchImplementation = "toggle_button"
	SwitchCheckbox     SwitchImplementation = "checkbox"
)

// RadioImplementation defines how a radio control is rendered
type RadioImplementation string

const (
	RadioGroup         RadioImplementation = "radio_group"
	RadioToggleButtons RadioImplementation = "toggle_button_group"
	RadioSteppedSlider RadioImplementation = "stepped_slider"
)

// RangeImplementation defines how a range control is rendered
type RangeImplementation string

const (
	RangeRotaryKnob   RangeImplementation = "rotary_knob"
	RangeRangedSlider RangeImplementation = "ranged_slider"
)

// Control represents an individual control element
type Control struct {
	Label          string      `json:"label"`
	ID             string      `json:"id"` // UUID
	Type           ControlType `json:"type"`
	Implementation string      `json:"implementation"` // Specific implementation based on type
	X              int         `json:"x"`              // Coordinate within group
	Y              int         `json:"y"`              // Coordinate within group
	Targets        []Target    `json:"targets"`        // Array of target assignments

	// Optional visual and behavioral properties
	Width    int     `json:"width,omitempty"`        // Control width in pixels
	Height   int     `json:"height,omitempty"`       // Control height in pixels
	MinValue float64 `json:"minValue,omitempty"`     // Minimum value for ranges
	MaxValue float64 `json:"maxValue,omitempty"`     // Maximum value for ranges
	StepSize float64 `json:"stepSize,omitempty"`     // Step size for discrete controls
	Default  float64 `json:"defaultValue,omitempty"` // Default value
}

// Target represents a parameter or MIDI target for a control
type Target struct {
	// Neural DSP Parameter targeting
	ParameterAddress int    `json:"parameterAddress,omitempty"` // From introspection data
	ParameterName    string `json:"parameterName,omitempty"`    // From introspection data

	// MIDI targeting
	CCMidi  int  `json:"ccMidi,omitempty"`  // MIDI CC number (0-127)
	Channel int  `json:"channel,omitempty"` // MIDI channel (1-16) for IAC driver
	Invert  bool `json:"invert"`            // Invert control position
	Stepped bool `json:"stepped"`           // For non-boolean indexed values

	// Display override
	Label string `json:"label,omitempty"` // Override plain names from introspection

	// Value mapping
	MinValue float64 `json:"minValue,omitempty"` // Target minimum value
	MaxValue float64 `json:"maxValue,omitempty"` // Target maximum value
}

// Layout represents the complete control layout configuration
type Layout struct {
	Name        string  `json:"name"`        // Layout name
	Description string  `json:"description"` // Layout description
	Version     string  `json:"version"`     // Layout format version
	Grid        Grid    `json:"grid"`        // Grid configuration
	Groups      []Group `json:"groups"`      // Control groups
}

// Validation and helper methods

// IsValid checks if the grid configuration is valid
func (g *Grid) IsValid() bool {
	return g.Rows >= 1 && g.Rows <= 5 && g.Columns >= 1 && g.Columns <= 5 && g.Gutter >= 0
}

// GetGridPosition calculates the actual pixel position based on grid coordinates
func (g *Grid) GetGridPosition(x, y, cellWidth, cellHeight int) (int, int) {
	pixelX := x * (cellWidth + g.Gutter)
	pixelY := y * (cellHeight + g.Gutter)
	return pixelX, pixelY
}

// GetControlsByType returns all controls of a specific type
func (group *Group) GetControlsByType(controlType ControlType) []Control {
	var controls []Control
	for _, control := range group.Controls {
		if control.Type == controlType {
			controls = append(controls, control)
		}
	}
	return controls
}

// GetTargetByParameterAddress finds a target by parameter address
func (control *Control) GetTargetByParameterAddress(address int) *Target {
	for i := range control.Targets {
		if control.Targets[i].ParameterAddress == address {
			return &control.Targets[i]
		}
	}
	return nil
}

// GetTargetByMIDI finds a target by MIDI CC and channel
func (control *Control) GetTargetByMIDI(cc, channel int) *Target {
	for i := range control.Targets {
		if control.Targets[i].CCMidi == cc && control.Targets[i].Channel == channel {
			return &control.Targets[i]
		}
	}
	return nil
}

// HasParameterTarget checks if control targets a specific parameter
func (control *Control) HasParameterTarget(address int) bool {
	return control.GetTargetByParameterAddress(address) != nil
}

// HasMIDITarget checks if control has MIDI targeting
func (control *Control) HasMIDITarget() bool {
	for _, target := range control.Targets {
		if target.CCMidi > 0 {
			return true
		}
	}
	return false
}

// GetGroupByID finds a group by its ID
func (layout *Layout) GetGroupByID(id string) *Group {
	for i := range layout.Groups {
		if layout.Groups[i].ID == id {
			return &layout.Groups[i]
		}
	}
	return nil
}

// GetControlByID finds a control by its ID across all groups
func (layout *Layout) GetControlByID(id string) (*Control, *Group) {
	for i := range layout.Groups {
		for j := range layout.Groups[i].Controls {
			if layout.Groups[i].Controls[j].ID == id {
				return &layout.Groups[i].Controls[j], &layout.Groups[i]
			}
		}
	}
	return nil, nil
}

// GetControlsByParameterAddress finds all controls targeting a specific parameter
func (layout *Layout) GetControlsByParameterAddress(address int) []Control {
	var controls []Control
	for _, group := range layout.Groups {
		for _, control := range group.Controls {
			if control.HasParameterTarget(address) {
				controls = append(controls, control)
			}
		}
	}
	return controls
}
