package debug

import (
	"fmt"
	"strings"
)

// Device represents the minimal interface needed for dashboard rendering
type Device interface {
	GetDeviceID() int
	GetName() string
	GetSupportedSampleRates() []int
	IsDeviceOnline() bool
	IsDeviceDefault() bool
}

// DashboardData holds all the data needed for the debug dashboard
type DashboardData struct {
	ProcessRunning bool
	PID            int
	EngineRunning  bool
	StatusDetails  string
	InputDevices   []Device
	OutputDevices  []Device
	PluginCount    int
	DefaultInput   int
	DefaultOutput  int
	DefaultRate    float64
	Timestamp      string
}

// RenderHTML generates the complete HTML for the debug dashboard
func RenderHTML(data DashboardData) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Rackless Debug Dashboard</title>
    <style>%s</style>
</head>
<body>
    <h1>🎛️ Rackless Debug Dashboard</h1>
    
    <div class="section">
        <h2>Audio System Status</h2>
        %s
        %s
    </div>
    
    <div class="section">
        <h2>Quick Actions</h2>
        %s
    </div>
    
    <div class="section">
        <h2>Available Audio Devices</h2>
        <h3>Input Devices:</h3>
        %s
        <h3>Output Devices:</h3>
        %s
    </div>
    
    <div class="section">
        <h2>Server Info</h2>
        %s
    </div>
    
    <script>%s</script>
</body>
</html>`,
		getCSS(),
		renderAudioStatus(data),
		renderStatusDetails(data),
		renderQuickActions(),
		renderDeviceList(data.InputDevices),
		renderDeviceList(data.OutputDevices),
		renderServerInfo(data),
		getJavaScript(),
	)
}

// getCSS returns the CSS styles for the debug dashboard
func getCSS() string {
	return `
        body { font-family: Arial, sans-serif; margin: 20px; background: #1a1a1a; color: #e0e0e0; }
        .status { padding: 10px; margin: 10px 0; border-radius: 5px; }
        .running { background: #2d5a27; border: 1px solid #4a8f42; }
        .stopped { background: #5a2727; border: 1px solid #8f4242; }
        .info { background: #2d4a5a; border: 1px solid #4a7a8f; }
        .section { margin: 20px 0; padding: 15px; background: #2a2a2a; border-radius: 5px; }
        button { padding: 8px 15px; margin: 5px; background: #3a3a3a; color: #e0e0e0; border: 1px solid #555; border-radius: 3px; cursor: pointer; }
        button:hover { background: #4a4a4a; }
        pre { background: #1a1a1a; padding: 10px; border-radius: 3px; overflow-x: auto; }
        .device { margin: 5px 0; padding: 8px; background: #333; border-radius: 3px; }
        .device.online { border-left: 3px solid #4a8f42; }
        .device.offline { border-left: 3px solid #8f4242; }
    `
}

// renderAudioStatus renders the audio system status section
func renderAudioStatus(data DashboardData) string {
	statusClass := "stopped"
	processStatus := "STOPPED"
	pidInfo := ""
	engineStatus := "NOT RUNNING"
	additionalInfo := ""

	if data.ProcessRunning {
		statusClass = "running"
		processStatus = "RUNNING"
		pidInfo = fmt.Sprintf("(PID %d)", data.PID)
		
		if data.EngineRunning {
			engineStatus = "RUNNING"
		} else {
			engineStatus = "STOPPED"
		}
		
		if data.StatusDetails != "" {
			additionalInfo = fmt.Sprintf("<br><strong>Details:</strong> %s", data.StatusDetails)
		}
	}

	return fmt.Sprintf(`<div class="status %s">
            <strong>Process:</strong> %s %s<br>
            <strong>Engine:</strong> %s%s
        </div>`, statusClass, processStatus, pidInfo, engineStatus, additionalInfo)
}

// renderStatusDetails renders the detailed status information
func renderStatusDetails(data DashboardData) string {
	if data.ProcessRunning && data.StatusDetails != "" {
		return fmt.Sprintf("<pre>%s</pre>", data.StatusDetails)
	}
	return ""
}

// renderQuickActions renders the quick action buttons
func renderQuickActions() string {
	return `
        <button onclick="sendCommand('status')">Get Status</button>
        <button onclick="sendCommand('ping')">Ping Audio Host</button>
        <button onclick="stopAudio()">Stop Audio</button>
        <button onclick="refreshPage()">Refresh Page</button>
    `
}

// renderDeviceList renders a list of audio devices
func renderDeviceList(devices []Device) string {
	var html strings.Builder
	for _, device := range devices {
		status := "offline"
		if device.IsDeviceOnline() {
			status = "online"
		}
		
		defaultLabel := ""
		if device.IsDeviceDefault() {
			defaultLabel = "(DEFAULT)"
		}
		
		html.WriteString(fmt.Sprintf(
			`<div class="device %s"><strong>%d:</strong> %s %s<br><small>Rates: %v</small></div>`,
			status, device.GetDeviceID(), device.GetName(), defaultLabel, device.GetSupportedSampleRates(),
		))
	}
	return html.String()
}

// renderServerInfo renders the server information section
func renderServerInfo(data DashboardData) string {
	return fmt.Sprintf(`<div class="info">
            <strong>Plugins loaded:</strong> %d<br>
            <strong>Default input:</strong> %d<br>
            <strong>Default output:</strong> %d<br>
            <strong>Default sample rate:</strong> %.0f Hz<br>
            <strong>Timestamp:</strong> %s
        </div>`, data.PluginCount, data.DefaultInput, data.DefaultOutput, data.DefaultRate, data.Timestamp)
}

// getJavaScript returns the JavaScript for the debug dashboard
func getJavaScript() string {
	return `
        function sendCommand(cmd) {
            fetch('/api/audio/command', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ command: cmd })
            })
            .then(r => r.json())
            .then(data => {
                alert('Response: ' + (data.output || data.error || 'No response'));
            })
            .catch(err => alert('Error: ' + err));
        }
        
        function stopAudio() {
            if (confirm('Stop audio host?')) {
                fetch('/api/audio/stop', { method: 'POST' })
                .then(r => r.json())
                .then(data => {
                    alert(data.message);
                    setTimeout(() => location.reload(), 1000);
                });
            }
        }
        
        function refreshPage() {
            location.reload();
        }
    `
}
