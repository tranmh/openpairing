package trf

// EngineType represents a pairing engine type
type EngineType string

const (
	// EngineJaVaFo represents the FIDE-endorsed JaVaFo pairing engine
	EngineJaVaFo EngineType = "javafo"
	// EngineBBP represents the bbpPairings engine
	EngineBBP EngineType = "bbp"
)

// NewTRFGenerator creates a TRF generator for the specified engine.
// Each engine has specific TRF format requirements:
//   - JaVaFo (TRFx): Fails on header codes 062, 072, 082, 092, 102, 112
//   - bbpPairings (TRFbx): Requires XXC records, validates scores strictly
//
// If an unknown engine type is provided, defaults to JaVaFo generator.
func NewTRFGenerator(engine EngineType) TRFGenerator {
	switch engine {
	case EngineBBP:
		return NewBBPGenerator()
	case EngineJaVaFo:
		return NewJaVaFoGenerator()
	default:
		// Default to JaVaFo for unknown engines (safest option)
		return NewJaVaFoGenerator()
	}
}

// NewTRFGeneratorFromString creates a TRF generator from a string engine name.
// Useful when engine name comes from configuration or user input.
func NewTRFGeneratorFromString(engineName string) TRFGenerator {
	switch engineName {
	case "bbp", "bbpPairings", "bbppairings":
		return NewBBPGenerator()
	case "javafo", "JaVaFo", "swiss", "dutch":
		return NewJaVaFoGenerator()
	default:
		return NewJaVaFoGenerator()
	}
}
