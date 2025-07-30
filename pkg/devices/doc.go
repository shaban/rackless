// Package devices provides cross-platform audio and MIDI device enumeration
// for the rackless audio plugin host system.
//
// This package follows the same architecture pattern as the introspection package:
// - types.go: Common data structures
// - native.go: CGO implementation for macOS
// - stub.go: Cross-platform fallback
package devices
