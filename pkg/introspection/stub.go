//go:build !darwin || !cgo
// +build !darwin !cgo

package introspection

// GetAudioUnits returns mock data on non-macOS platforms
func GetAudioUnits() (IntrospectionResult, error) {
	return IntrospectionResult{
		{
			Name:           "Mock AudioUnit",
			ManufacturerID: "MOCK",
			Type:           "aufx",
			Subtype:        "mock",
			Parameters: []Parameter{
				{
					Unit:         "Percent",
					DisplayName:  "Mock Parameter",
					Address:      1,
					MaxValue:     100.0,
					Identifier:   "mock_param",
					MinValue:     0.0,
					CanRamp:      true,
					IsWritable:   true,
					RawFlags:     0,
					DefaultValue: 50.0,
					CurrentValue: 50.0,
				},
			},
		},
	}, nil
}

// GetAudioUnitsJSON returns mock JSON on non-macOS platforms
func GetAudioUnitsJSON() (string, error) {
	return `[{"name":"Mock AudioUnit","manufacturerID":"MOCK","type":"aufx","subtype":"mock","parameters":[{"unit":"Percent","displayName":"Mock Parameter","address":1,"maxValue":100,"identifier":"mock_param","minValue":0,"canRamp":true,"isWritable":true,"rawFlags":0,"defaultValue":50,"currentValue":50}]}]`, nil
}
