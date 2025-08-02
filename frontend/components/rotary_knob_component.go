//go:build js && wasm
// +build js,wasm

package components

import (
	"fmt"
	"syscall/js"
)

// RotaryKnob represents a rotary control component
type RotaryKnob struct {
	ID           string
	Label        string
	Unit         string
	MinValue     float64
	MaxValue     float64
	Value        float64
	DefaultValue float64
	Size         int
	OnChange     func(float64)
	
	// Internal state
	element      js.Value
	rotatingGroup js.Value // Store direct reference to rotating group
	isDragging   bool
	startY       float64
	startValue   float64
	mouseHandler js.Func
	clickHandler js.Func
}// NewRotaryKnob creates a new rotary knob component
func NewRotaryKnob(id, label, unit string, minVal, maxVal, defaultVal float64, size int) *RotaryKnob {
	return &RotaryKnob{
		ID:           id,
		Label:        label,
		Unit:         unit,
		MinValue:     minVal,
		MaxValue:     maxVal,
		Value:        defaultVal,
		DefaultValue: defaultVal,
		Size:         size,
	}
}

// Render creates and returns the DOM element for the rotary knob
func (rk *RotaryKnob) Render() js.Value {
	doc := js.Global().Get("document")

	// Create container
	container := doc.Call("createElement", "div")
	container.Get("classList").Call("add", "rotary-knob")
	container.Set("id", rk.ID)

	// Create label
	label := doc.Call("createElement", "div")
	label.Get("classList").Call("add", "label")
	label.Set("textContent", rk.Label)

	// Create knob container
	knobContainer := doc.Call("createElement", "div")
	knobContainer.Get("classList").Call("add", "container")

	// Create SVG track and handle
	svg := rk.createSVG()
	knobContainer.Call("appendChild", svg)

	// Create value display
	valueDisplay := doc.Call("createElement", "div")
	valueDisplay.Get("classList").Call("add", "value")
	valueDisplay.Set("id", rk.ID+"-value")
	valueDisplay.Set("textContent", rk.formatValue())

	// Assemble component
	container.Call("appendChild", label)
	container.Call("appendChild", knobContainer)
	container.Call("appendChild", valueDisplay)

	// Store element reference
	rk.element = container

	// Add event handlers
	rk.setupEventHandlers(svg)

	return container
}

// createSVG creates the SVG representation of the knob
func (rk *RotaryKnob) createSVG() js.Value {
	doc := js.Global().Get("document")

	// Create SVG element
	svg := doc.Call("createElementNS", "http://www.w3.org/2000/svg", "svg")
	svg.Get("classList").Call("add", "track")
	svg.Set("width", rk.Size)
	svg.Set("height", rk.Size)
	svg.Set("viewBox", fmt.Sprintf("0 0 %d %d", rk.Size, rk.Size))
	// Make sure SVG is properly positioned
	svg.Get("style").Set("display", "block")
	svg.Get("style").Set("position", "absolute")
	svg.Get("style").Set("top", "0")
	svg.Get("style").Set("left", "0")

	// Calculate dimensions
	center := float64(rk.Size) / 2
	radius := center - 8

	// Create outer ring (shaded contour)
	outerRing := doc.Call("createElementNS", "http://www.w3.org/2000/svg", "circle")
	outerRing.Set("cx", center)
	outerRing.Set("cy", center)
	outerRing.Set("r", radius)
	outerRing.Set("fill", "none")
	outerRing.Set("stroke", "#cbd5e0")
	outerRing.Set("stroke-width", "2")

	// Create knob body (main circle)
	knobBody := doc.Call("createElementNS", "http://www.w3.org/2000/svg", "circle")
	knobBody.Set("cx", center)
	knobBody.Set("cy", center)
	knobBody.Set("r", radius-4)
	knobBody.Set("fill", fmt.Sprintf("url(#%s-knobGradient)", rk.ID)) // Use unique gradient ID
	knobBody.Set("stroke", "#a0aec0")
	knobBody.Set("stroke-width", "1")

	// Create gradient definition
	defs := doc.Call("createElementNS", "http://www.w3.org/2000/svg", "defs")
	gradient := doc.Call("createElementNS", "http://www.w3.org/2000/svg", "radialGradient")
	gradient.Set("id", rk.ID+"-knobGradient") // Make gradient ID unique
	gradient.Set("cx", "30%")
	gradient.Set("cy", "30%")

	stop1 := doc.Call("createElementNS", "http://www.w3.org/2000/svg", "stop")
	stop1.Set("offset", "0%")
	stop1.Set("stop-color", "#f7fafc")

	stop2 := doc.Call("createElementNS", "http://www.w3.org/2000/svg", "stop")
	stop2.Set("offset", "100%")
	stop2.Set("stop-color", "#e2e8f0")

	gradient.Call("appendChild", stop1)
	gradient.Call("appendChild", stop2)
	defs.Call("appendChild", gradient)

	// Create rotating group for the index mark
	rotatingGroup := doc.Call("createElementNS", "http://www.w3.org/2000/svg", "g")
	rotatingGroup.Set("id", rk.ID+"-rotating")
	rotatingGroup.Get("style").Set("transform-origin", fmt.Sprintf("%.1fpx %.1fpx", center, center))
	
	// Store reference to rotating group
	rk.rotatingGroup = rotatingGroup

	// Create slim index mark - make it more visible
	indexMark := doc.Call("createElementNS", "http://www.w3.org/2000/svg", "line")
	indexMark.Set("x1", center)
	indexMark.Set("y1", 6) // Start very close to edge
	indexMark.Set("x2", center)
	indexMark.Set("y2", 30) // Much longer line
	indexMark.Set("stroke", "#ff0000") // Bright red for visibility
	indexMark.Set("stroke-width", "4") // Very thick
	indexMark.Set("stroke-linecap", "round")

	rotatingGroup.Call("appendChild", indexMark)

	// Assemble SVG
	svg.Call("appendChild", defs)
	svg.Call("appendChild", outerRing)
	svg.Call("appendChild", knobBody)
	svg.Call("appendChild", rotatingGroup)

	// Update visual representation
	rk.updateVisuals()

	return svg
}

// setupEventHandlers sets up mouse interaction
func (rk *RotaryKnob) setupEventHandlers(svg js.Value) {
	// Mouse down handler
	rk.mouseHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		eventType := event.Get("type").String()

		switch eventType {
		case "mousedown":
			rk.isDragging = true
			rk.startY = event.Get("clientY").Float()
			rk.startValue = rk.Value
			event.Call("preventDefault")

			// Add global mouse handlers
			doc := js.Global().Get("document")
			doc.Call("addEventListener", "mousemove", rk.mouseHandler)
			doc.Call("addEventListener", "mouseup", rk.mouseHandler)

		case "mousemove":
			if rk.isDragging {
				currentY := event.Get("clientY").Float()
				deltaY := rk.startY - currentY // Invert for intuitive direction

				// Calculate sensitivity (smaller range = more sensitive)
				sensitivity := 100.0
				valueRange := rk.MaxValue - rk.MinValue
				if valueRange < 1 {
					sensitivity = 200.0 // More sensitive for small ranges
				}

				deltaValue := (deltaY / sensitivity) * valueRange
				newValue := rk.startValue + deltaValue

				// Clamp to range
				if newValue < rk.MinValue {
					newValue = rk.MinValue
				} else if newValue > rk.MaxValue {
					newValue = rk.MaxValue
				}

				rk.SetValue(newValue)
				event.Call("preventDefault")
			}

		case "mouseup":
			if rk.isDragging {
				rk.isDragging = false

				// Remove global mouse handlers
				doc := js.Global().Get("document")
				doc.Call("removeEventListener", "mousemove", rk.mouseHandler)
				doc.Call("removeEventListener", "mouseup", rk.mouseHandler)
			}
		}

		return nil
	})

	// Double-click to reset handler
	rk.clickHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		rk.SetValue(rk.DefaultValue)
		return nil
	})

	// Add event listeners
	svg.Call("addEventListener", "mousedown", rk.mouseHandler)
	svg.Call("addEventListener", "dblclick", rk.clickHandler)

	// Prevent context menu
	svg.Call("addEventListener", "contextmenu", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		args[0].Call("preventDefault")
		return nil
	}))
}

// SetValue updates the knob value and visuals
func (rk *RotaryKnob) SetValue(value float64) {
	// Clamp value to range
	if value < rk.MinValue {
		value = rk.MinValue
	} else if value > rk.MaxValue {
		value = rk.MaxValue
	}

	rk.Value = value
	rk.updateVisuals()

	// Call change handler if set
	if rk.OnChange != nil {
		rk.OnChange(value)
	}
}

// updateVisuals updates the SVG representation
func (rk *RotaryKnob) updateVisuals() {
	if rk.element.IsNull() {
		return
	}

	doc := js.Global().Get("document")

	// Update value display
	valueDisplay := doc.Call("getElementById", rk.ID+"-value")
	if !valueDisplay.IsNull() {
		valueDisplay.Set("textContent", rk.formatValue())
	}

	// Update knob rotation
	rk.updateRotation()
}

// updateRotation updates the knob rotation
func (rk *RotaryKnob) updateRotation() {
	if rk.rotatingGroup.IsNull() {
		fmt.Printf("❌ Rotating group is null for %s\n", rk.ID)
		return
	}

	// Calculate rotation angle (270 degrees total range)
	normalizedValue := (rk.Value - rk.MinValue) / (rk.MaxValue - rk.MinValue)
	// Start at -135 degrees (8 o'clock), rotate 270 degrees total to 4 o'clock
	angle := -135.0 + normalizedValue*270.0

	// Apply rotation transform using setAttribute for SVG
	center := float64(rk.Size) / 2
	transform := fmt.Sprintf("rotate(%.1f %.1f %.1f)", angle, center, center)
	
	fmt.Printf("🔄 Rotating %s: value=%.1f, angle=%.1f, transform=%s\n", rk.ID, rk.Value, angle, transform)
	
	// Try both methods to set transform
	rk.rotatingGroup.Call("setAttribute", "transform", transform)
	rk.rotatingGroup.Set("transform", transform)
	
	// Also try setting via style
	rk.rotatingGroup.Get("style").Set("transform", transform)
}

// formatValue formats the value with appropriate precision and unit
func (rk *RotaryKnob) formatValue() string {
	switch rk.Unit {
	case "Hz":
		if rk.Value >= 1000 {
			return fmt.Sprintf("%.1fkHz", rk.Value/1000)
		}
		return fmt.Sprintf("%.0fHz", rk.Value)
	case "dB":
		return fmt.Sprintf("%.1fdB", rk.Value)
	case "%":
		return fmt.Sprintf("%.0f%%", rk.Value)
	case "ms":
		return fmt.Sprintf("%.1fms", rk.Value)
	case "s":
		return fmt.Sprintf("%.2fs", rk.Value)
	default:
		// Determine precision based on value range
		if rk.MaxValue-rk.MinValue < 10 {
			return fmt.Sprintf("%.2f", rk.Value)
		} else if rk.MaxValue-rk.MinValue < 100 {
			return fmt.Sprintf("%.1f", rk.Value)
		} else {
			return fmt.Sprintf("%.0f", rk.Value)
		}
	}
}

// Cleanup removes event handlers (call when component is destroyed)
func (rk *RotaryKnob) Cleanup() {
	if !rk.mouseHandler.IsNull() {
		rk.mouseHandler.Release()
	}
	if !rk.clickHandler.IsNull() {
		rk.clickHandler.Release()
	}
}
