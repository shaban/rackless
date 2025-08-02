package audio

import (
	"context"
	"io"
	"os/exec"
	"sync"
)

// Device structures based on standalone/devices output
type AudioDevice struct {
	DeviceID             int    `json:"deviceId"`
	UID                  string `json:"uid"`
	SupportedSampleRates []int  `json:"supportedSampleRates"`
	ChannelCount         int    `json:"channelCount"`
	IsDefault            bool   `json:"isDefault"`
	IsOnline             bool   `json:"isOnline"`
	Name                 string `json:"name"`
	SupportedBitDepths   []int  `json:"supportedBitDepths"`
}

// Implement debug.Device interface for AudioDevice
func (d AudioDevice) GetDeviceID() int               { return d.DeviceID }
func (d AudioDevice) GetName() string                { return d.Name }
func (d AudioDevice) GetSupportedSampleRates() []int { return d.SupportedSampleRates }
func (d AudioDevice) IsDeviceOnline() bool           { return d.IsOnline }
func (d AudioDevice) IsDeviceDefault() bool          { return d.IsDefault }

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

// Audio configuration for starting audio-host
type AudioConfig struct {
	SampleRate         float64 `json:"sampleRate"`
	BufferSize         int     `json:"bufferSize,omitempty"`
	AudioInputDeviceID int     `json:"audioInputDeviceID,omitempty"`
	AudioInputChannel  int     `json:"audioInputChannel,omitempty"`
	EnableTestTone     bool    `json:"enableTestTone,omitempty"`
	PluginPath         string  `json:"pluginPath,omitempty"`
}

// Audio start request
type StartAudioRequest struct {
	Config AudioConfig `json:"config"`
}

// Audio start response
type StartAudioResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	PID     int    `json:"pid,omitempty"`
}

// Audio command request
type AudioCommandRequest struct {
	Command string `json:"command"`
}

// Audio command response
type AudioCommandResponse struct {
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Device test request for simplified boolean approach
type DeviceTestRequest struct {
	InputDeviceID  int     `json:"inputDeviceID"`
	OutputDeviceID int     `json:"outputDeviceID,omitempty"`
	SampleRate     float64 `json:"sampleRate"`
	BufferSize     int     `json:"bufferSize,omitempty"`
}

// Device test response with boolean ready state
type DeviceTestResponse struct {
	IsAudioReady   bool        `json:"isAudioReady"`
	ErrorMessage   string      `json:"errorMessage,omitempty"`
	RequiredAction string      `json:"requiredAction,omitempty"`
	TestedConfig   AudioConfig `json:"testedConfig"`
}

// Device switch request for changing audio devices
type DeviceSwitchRequest struct {
	InputDeviceID  int     `json:"inputDeviceID"`
	OutputDeviceID int     `json:"outputDeviceID,omitempty"`
	SampleRate     float64 `json:"sampleRate"`
	BufferSize     int     `json:"bufferSize,omitempty"`
}

// Device switch response with boolean ready state
type DeviceSwitchResponse struct {
	IsAudioReady           bool        `json:"isAudioReady"`
	ErrorMessage           string      `json:"errorMessage,omitempty"`
	RequiredAction         string      `json:"requiredAction,omitempty"`
	NewConfig              AudioConfig `json:"newConfig"`
	PreviousProcessRunning bool        `json:"previousProcessRunning"`
	ProcessRestarted       bool        `json:"processRestarted"`
	PID                    int         `json:"pid,omitempty"`
}

// AudioHost process management
type AudioHostProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	pid     int
	running bool
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// Configuration management types
type ChangeRequirement int

const (
	NoChangeRequired ChangeRequirement = iota
	ChainRebuildRequired
	ProcessRestartRequired
	DynamicChangeOnly
)

// ConfigChange represents a requested configuration change
type ConfigChange struct {
	NewConfig    AudioConfig
	ChangeReason string
}

// ReconfigurationResult contains the outcome of a reconfiguration attempt
type ReconfigurationResult struct {
	Success          bool
	ChangeType       ChangeRequirement
	Message          string
	PreviousConfig   *AudioConfig
	NewConfig        *AudioConfig
	RequiredRestart  bool
	ProcessIDChanged bool
	OldPID           int
	NewPID           int
}

// AudioEngineReconfiguration handles changes that require rebuilding the audio chain
type AudioEngineReconfiguration struct {
	currentConfig *AudioConfig
	isRunning     bool
}
