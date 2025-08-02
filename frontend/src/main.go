//go:build js && wasm
// +build js,wasm

package main

import (
	"syscall/js"
)

func main() {
	// Set up the application
	document := js.Global().Get("document")
	app := document.Call("getElementById", "app")

	// Create initial content
	app.Set("innerHTML", `
		<div style="padding: 2rem; text-align: center;">
			<h1>🎵 Rackless Audio Control</h1>
			<p>Ready to build something amazing!</p>
		</div>
	`)

	// Keep the Go program running
	select {}
}
