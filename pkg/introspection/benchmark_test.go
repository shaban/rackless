//go:build darwin && cgo
// +build darwin,cgo

package introspection

import (
	"testing"
)

func BenchmarkGetAudioUnits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetAudioUnits()
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func BenchmarkGetAudioUnitsJSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetAudioUnitsJSON()
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func BenchmarkIntrospectionResultMethods(b *testing.B) {
	// Get plugins once for the benchmark
	plugins, err := GetAudioUnits()
	if err != nil {
		b.Fatalf("Failed to get plugins for benchmark: %v", err)
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Benchmark query methods
		_ = plugins.GetParameterCount()
		_ = plugins.SelectBestPluginForLayout()
		
		if len(plugins) > 0 {
			_ = plugins.FindPluginByName(plugins[0].Name)
		}
	}
}
