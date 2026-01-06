package trf

import "testing"

func TestNewTRFGenerator(t *testing.T) {
	tests := []struct {
		engine     EngineType
		wantEngine string
	}{
		{EngineJaVaFo, "JaVaFo"},
		{EngineBBP, "bbpPairings"},
		{"unknown", "JaVaFo"},   // Default to JaVaFo
		{"", "JaVaFo"},          // Empty defaults to JaVaFo
	}

	for _, tt := range tests {
		t.Run(string(tt.engine), func(t *testing.T) {
			g := NewTRFGenerator(tt.engine)
			if g.EngineName() != tt.wantEngine {
				t.Errorf("NewTRFGenerator(%q).EngineName() = %q, want %q",
					tt.engine, g.EngineName(), tt.wantEngine)
			}
		})
	}
}

func TestNewTRFGeneratorFromString(t *testing.T) {
	tests := []struct {
		engineName string
		wantEngine string
	}{
		{"javafo", "JaVaFo"},
		{"JaVaFo", "JaVaFo"},
		{"swiss", "JaVaFo"},
		{"dutch", "JaVaFo"},
		{"bbp", "bbpPairings"},
		{"bbpPairings", "bbpPairings"},
		{"bbppairings", "bbpPairings"},
		{"unknown", "JaVaFo"},
		{"", "JaVaFo"},
	}

	for _, tt := range tests {
		t.Run(tt.engineName, func(t *testing.T) {
			g := NewTRFGeneratorFromString(tt.engineName)
			if g.EngineName() != tt.wantEngine {
				t.Errorf("NewTRFGeneratorFromString(%q).EngineName() = %q, want %q",
					tt.engineName, g.EngineName(), tt.wantEngine)
			}
		})
	}
}

func TestEngineTypeConstants(t *testing.T) {
	if EngineJaVaFo != "javafo" {
		t.Errorf("EngineJaVaFo = %q, want %q", EngineJaVaFo, "javafo")
	}
	if EngineBBP != "bbp" {
		t.Errorf("EngineBBP = %q, want %q", EngineBBP, "bbp")
	}
}
