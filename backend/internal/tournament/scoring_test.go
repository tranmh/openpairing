package tournament

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// ScoringSettings Tests
// =============================================================================

func TestDefaultScoringSettings(t *testing.T) {
	settings := DefaultScoringSettings()

	if settings == nil {
		t.Fatal("DefaultScoringSettings returned nil")
	}

	t.Run("Standard preset values", func(t *testing.T) {
		if settings.Preset != ScoringPresetStandard {
			t.Errorf("Preset = %s, want %s", settings.Preset, ScoringPresetStandard)
		}
		if settings.WinPoints != 1.0 {
			t.Errorf("WinPoints = %f, want 1.0", settings.WinPoints)
		}
		if settings.DrawPoints != 0.5 {
			t.Errorf("DrawPoints = %f, want 0.5", settings.DrawPoints)
		}
		if settings.LossPoints != 0.0 {
			t.Errorf("LossPoints = %f, want 0.0", settings.LossPoints)
		}
		if settings.SonnebornBergerMode != SBModeProportional {
			t.Errorf("SonnebornBergerMode = %s, want %s", settings.SonnebornBergerMode, SBModeProportional)
		}
	})

	t.Run("Should be valid", func(t *testing.T) {
		if !settings.IsValid() {
			t.Error("Default settings should be valid")
		}
	})

	t.Run("Should be FIDE compliant", func(t *testing.T) {
		violations := settings.ValidateFIDECompliance()
		if len(violations) != 0 {
			t.Errorf("Default settings should have no FIDE violations, got %d: %+v", len(violations), violations)
		}
	})
}

func TestFootballScoringSettings(t *testing.T) {
	settings := FootballScoringSettings()

	if settings == nil {
		t.Fatal("FootballScoringSettings returned nil")
	}

	t.Run("Football preset values", func(t *testing.T) {
		if settings.Preset != ScoringPresetFootball {
			t.Errorf("Preset = %s, want %s", settings.Preset, ScoringPresetFootball)
		}
		if settings.WinPoints != 3.0 {
			t.Errorf("WinPoints = %f, want 3.0", settings.WinPoints)
		}
		if settings.DrawPoints != 1.0 {
			t.Errorf("DrawPoints = %f, want 1.0", settings.DrawPoints)
		}
		if settings.LossPoints != 0.0 {
			t.Errorf("LossPoints = %f, want 0.0", settings.LossPoints)
		}
	})

	t.Run("Should be valid", func(t *testing.T) {
		if !settings.IsValid() {
			t.Error("Football settings should be valid")
		}
	})

	t.Run("Should have FIDE violations", func(t *testing.T) {
		violations := settings.ValidateFIDECompliance()
		if len(violations) != 2 { // WinPoints and DrawPoints differ from FIDE
			t.Errorf("Football settings should have 2 FIDE violations, got %d", len(violations))
		}
	})
}

func TestGetScoringPreset(t *testing.T) {
	t.Run("Standard preset", func(t *testing.T) {
		settings := GetScoringPreset(ScoringPresetStandard)
		if settings.WinPoints != 1.0 {
			t.Errorf("Standard preset WinPoints = %f, want 1.0", settings.WinPoints)
		}
	})

	t.Run("Football preset", func(t *testing.T) {
		settings := GetScoringPreset(ScoringPresetFootball)
		if settings.WinPoints != 3.0 {
			t.Errorf("Football preset WinPoints = %f, want 3.0", settings.WinPoints)
		}
	})

	t.Run("Custom preset falls back to standard", func(t *testing.T) {
		settings := GetScoringPreset(ScoringPresetCustom)
		if settings.WinPoints != 1.0 {
			t.Errorf("Custom preset should fallback to standard, WinPoints = %f, want 1.0", settings.WinPoints)
		}
	})

	t.Run("Unknown preset falls back to standard", func(t *testing.T) {
		settings := GetScoringPreset("unknown")
		if settings.WinPoints != 1.0 {
			t.Errorf("Unknown preset should fallback to standard, WinPoints = %f, want 1.0", settings.WinPoints)
		}
	})
}

func TestScoringSettings_Validate(t *testing.T) {
	t.Run("Valid standard settings", func(t *testing.T) {
		settings := DefaultScoringSettings()
		errors := settings.Validate()
		if len(errors) != 0 {
			t.Errorf("Standard settings should have no errors, got %d: %+v", len(errors), errors)
		}
	})

	t.Run("Valid football settings", func(t *testing.T) {
		settings := FootballScoringSettings()
		errors := settings.Validate()
		if len(errors) != 0 {
			t.Errorf("Football settings should have no errors, got %d: %+v", len(errors), errors)
		}
	})

	t.Run("Invalid - zero win points", func(t *testing.T) {
		settings := &ScoringSettings{
			Preset:              ScoringPresetCustom,
			WinPoints:           0,
			DrawPoints:          0.5,
			LossPoints:          0,
			SonnebornBergerMode: SBModeProportional,
		}
		errors := settings.Validate()
		found := false
		for _, e := range errors {
			if e.Field == "winPoints" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected validation error for zero win points")
		}
	})

	t.Run("Invalid - negative draw points", func(t *testing.T) {
		settings := &ScoringSettings{
			Preset:              ScoringPresetCustom,
			WinPoints:           1.0,
			DrawPoints:          -0.5,
			LossPoints:          0,
			SonnebornBergerMode: SBModeProportional,
		}
		errors := settings.Validate()
		found := false
		for _, e := range errors {
			if e.Field == "drawPoints" && e.Message == "draw points cannot be negative" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected validation error for negative draw points")
		}
	})

	t.Run("Invalid - win equal to draw", func(t *testing.T) {
		settings := &ScoringSettings{
			Preset:              ScoringPresetCustom,
			WinPoints:           1.0,
			DrawPoints:          1.0,
			LossPoints:          0,
			SonnebornBergerMode: SBModeProportional,
		}
		errors := settings.Validate()
		found := false
		for _, e := range errors {
			if e.Field == "winPoints" && e.Message == "win points must be greater than draw points" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected validation error for win equal to draw")
		}
	})

	t.Run("Invalid - draw less than or equal to loss", func(t *testing.T) {
		settings := &ScoringSettings{
			Preset:              ScoringPresetCustom,
			WinPoints:           3.0,
			DrawPoints:          0,
			LossPoints:          0, // Equal to draw - should be invalid
			SonnebornBergerMode: SBModeProportional,
		}
		errors := settings.Validate()
		found := false
		for _, e := range errors {
			if e.Field == "drawPoints" && e.Message == "draw points must be greater than loss points" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected validation error for draw equal to loss")
		}
	})

	t.Run("Invalid - bad SB mode", func(t *testing.T) {
		settings := &ScoringSettings{
			Preset:              ScoringPresetCustom,
			WinPoints:           1.0,
			DrawPoints:          0.5,
			LossPoints:          0,
			SonnebornBergerMode: "invalid",
		}
		errors := settings.Validate()
		found := false
		for _, e := range errors {
			if e.Field == "sonnebornBergerMode" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected validation error for invalid SB mode")
		}
	})

	t.Run("Invalid - standard preset with wrong values", func(t *testing.T) {
		settings := &ScoringSettings{
			Preset:              ScoringPresetStandard,
			WinPoints:           3.0, // Wrong for standard
			DrawPoints:          1.0, // Wrong for standard
			LossPoints:          0,
			SonnebornBergerMode: SBModeProportional,
		}
		errors := settings.Validate()
		found := false
		for _, e := range errors {
			if e.Field == "preset" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected validation error for preset mismatch")
		}
	})

	t.Run("Valid - custom preset with valid values", func(t *testing.T) {
		settings := &ScoringSettings{
			Preset:              ScoringPresetCustom,
			WinPoints:           2.0,
			DrawPoints:          1.0,
			LossPoints:          0,
			SonnebornBergerMode: SBModeTraditional,
		}
		errors := settings.Validate()
		if len(errors) != 0 {
			t.Errorf("Custom preset with valid values should have no errors, got %d: %+v", len(errors), errors)
		}
	})
}

func TestScoringSettings_ValidateFIDECompliance(t *testing.T) {
	t.Run("Standard is FIDE compliant", func(t *testing.T) {
		settings := DefaultScoringSettings()
		violations := settings.ValidateFIDECompliance()
		if len(violations) != 0 {
			t.Errorf("Standard settings should be FIDE compliant, got %d violations", len(violations))
		}
	})

	t.Run("Football has FIDE violations", func(t *testing.T) {
		settings := FootballScoringSettings()
		violations := settings.ValidateFIDECompliance()

		// Should have violations for WinPoints and DrawPoints
		hasWinViolation := false
		hasDrawViolation := false
		for _, v := range violations {
			if v.Setting == "winPoints" {
				hasWinViolation = true
				if v.FIDEValue != 1.0 {
					t.Errorf("FIDE value for winPoints should be 1.0, got %v", v.FIDEValue)
				}
			}
			if v.Setting == "drawPoints" {
				hasDrawViolation = true
				if v.FIDEValue != 0.5 {
					t.Errorf("FIDE value for drawPoints should be 0.5, got %v", v.FIDEValue)
				}
			}
		}
		if !hasWinViolation {
			t.Error("Football settings should have winPoints FIDE violation")
		}
		if !hasDrawViolation {
			t.Error("Football settings should have drawPoints FIDE violation")
		}
	})
}

func TestScoringSettings_IsStandardScoring(t *testing.T) {
	t.Run("Standard settings", func(t *testing.T) {
		settings := DefaultScoringSettings()
		if !settings.IsStandardScoring() {
			t.Error("Default settings should be standard scoring")
		}
	})

	t.Run("Football settings", func(t *testing.T) {
		settings := FootballScoringSettings()
		if settings.IsStandardScoring() {
			t.Error("Football settings should not be standard scoring")
		}
	})

	t.Run("Custom with standard values", func(t *testing.T) {
		settings := &ScoringSettings{
			Preset:     ScoringPresetCustom,
			WinPoints:  1.0,
			DrawPoints: 0.5,
			LossPoints: 0.0,
		}
		if !settings.IsStandardScoring() {
			t.Error("Custom settings with 1-0.5-0 should be standard scoring")
		}
	})
}

func TestScoringSettings_JSONRoundTrip(t *testing.T) {
	settings := FootballScoringSettings()

	// Serialize to JSON
	jsonBytes, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal ScoringSettings: %v", err)
	}

	// Deserialize back
	var decoded ScoringSettings
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ScoringSettings: %v", err)
	}

	// Verify fields survived round-trip
	if decoded.Preset != settings.Preset {
		t.Errorf("Preset changed: got %s, want %s", decoded.Preset, settings.Preset)
	}
	if decoded.WinPoints != settings.WinPoints {
		t.Errorf("WinPoints changed: got %f, want %f", decoded.WinPoints, settings.WinPoints)
	}
	if decoded.DrawPoints != settings.DrawPoints {
		t.Errorf("DrawPoints changed: got %f, want %f", decoded.DrawPoints, settings.DrawPoints)
	}
}

// =============================================================================
// Scoring Functions Tests
// =============================================================================

func TestGetPointsForResult(t *testing.T) {
	t.Run("Standard scoring", func(t *testing.T) {
		scoring := DefaultScoringSettings()

		if GetPointsForResult(ResultTypeWin, scoring) != 1.0 {
			t.Error("Win should give 1.0 points in standard scoring")
		}
		if GetPointsForResult(ResultTypeDraw, scoring) != 0.5 {
			t.Error("Draw should give 0.5 points in standard scoring")
		}
		if GetPointsForResult(ResultTypeLoss, scoring) != 0.0 {
			t.Error("Loss should give 0.0 points in standard scoring")
		}
	})

	t.Run("Football scoring", func(t *testing.T) {
		scoring := FootballScoringSettings()

		if GetPointsForResult(ResultTypeWin, scoring) != 3.0 {
			t.Error("Win should give 3.0 points in football scoring")
		}
		if GetPointsForResult(ResultTypeDraw, scoring) != 1.0 {
			t.Error("Draw should give 1.0 points in football scoring")
		}
		if GetPointsForResult(ResultTypeLoss, scoring) != 0.0 {
			t.Error("Loss should give 0.0 points in football scoring")
		}
	})

	t.Run("Nil scoring uses defaults", func(t *testing.T) {
		if GetPointsForResult(ResultTypeWin, nil) != 1.0 {
			t.Error("Nil scoring should use default win points (1.0)")
		}
	})
}

func TestGetResultType(t *testing.T) {
	tests := []struct {
		result   string
		forWhite bool
		expected ResultType
	}{
		{ResultWhiteWins, true, ResultTypeWin},
		{ResultWhiteWins, false, ResultTypeLoss},
		{ResultBlackWins, true, ResultTypeLoss},
		{ResultBlackWins, false, ResultTypeWin},
		{ResultDraw, true, ResultTypeDraw},
		{ResultDraw, false, ResultTypeDraw},
		{ResultDoubleForfeit, true, ResultTypeLoss},
		{ResultDoubleForfeit, false, ResultTypeLoss},
	}

	for _, tt := range tests {
		result := GetResultType(tt.result, tt.forWhite)
		if result != tt.expected {
			t.Errorf("GetResultType(%s, %v) = %v, want %v", tt.result, tt.forWhite, result, tt.expected)
		}
	}
}

func TestParseResultWithScoring(t *testing.T) {
	t.Run("Standard scoring", func(t *testing.T) {
		scoring := DefaultScoringSettings()

		white, black := ParseResultWithScoring(ResultWhiteWins, scoring)
		if white != 1.0 || black != 0.0 {
			t.Errorf("White wins: got (%f, %f), want (1.0, 0.0)", white, black)
		}

		white, black = ParseResultWithScoring(ResultBlackWins, scoring)
		if white != 0.0 || black != 1.0 {
			t.Errorf("Black wins: got (%f, %f), want (0.0, 1.0)", white, black)
		}

		white, black = ParseResultWithScoring(ResultDraw, scoring)
		if white != 0.5 || black != 0.5 {
			t.Errorf("Draw: got (%f, %f), want (0.5, 0.5)", white, black)
		}
	})

	t.Run("Football scoring", func(t *testing.T) {
		scoring := FootballScoringSettings()

		white, black := ParseResultWithScoring(ResultWhiteWins, scoring)
		if white != 3.0 || black != 0.0 {
			t.Errorf("White wins: got (%f, %f), want (3.0, 0.0)", white, black)
		}

		white, black = ParseResultWithScoring(ResultDraw, scoring)
		if white != 1.0 || black != 1.0 {
			t.Errorf("Draw: got (%f, %f), want (1.0, 1.0)", white, black)
		}
	})
}

func TestIsWinDrawLoss(t *testing.T) {
	scoring := FootballScoringSettings()

	t.Run("IsWin", func(t *testing.T) {
		if !IsWin(3.0, scoring) {
			t.Error("3.0 should be a win in football scoring")
		}
		if IsWin(1.0, scoring) {
			t.Error("1.0 should not be a win in football scoring")
		}
	})

	t.Run("IsDraw", func(t *testing.T) {
		if !IsDraw(1.0, scoring) {
			t.Error("1.0 should be a draw in football scoring")
		}
		if IsDraw(3.0, scoring) {
			t.Error("3.0 should not be a draw in football scoring")
		}
	})

	t.Run("IsLoss", func(t *testing.T) {
		if !IsLoss(0.0, scoring) {
			t.Error("0.0 should be a loss in football scoring")
		}
		if IsLoss(1.0, scoring) {
			t.Error("1.0 should not be a loss in football scoring")
		}
	})
}

func TestGetResultTypeFromScore(t *testing.T) {
	scoring := FootballScoringSettings()

	if GetResultTypeFromScore(3.0, scoring) != ResultTypeWin {
		t.Error("3.0 should be ResultTypeWin in football scoring")
	}
	if GetResultTypeFromScore(1.0, scoring) != ResultTypeDraw {
		t.Error("1.0 should be ResultTypeDraw in football scoring")
	}
	if GetResultTypeFromScore(0.0, scoring) != ResultTypeLoss {
		t.Error("0.0 should be ResultTypeLoss in football scoring")
	}
}

func TestCalculateSonnebornBerger(t *testing.T) {
	t.Run("Traditional mode", func(t *testing.T) {
		scoring := &ScoringSettings{
			Preset:              ScoringPresetCustom,
			WinPoints:           1.0,
			DrawPoints:          0.5,
			LossPoints:          0.0,
			SonnebornBergerMode: SBModeTraditional,
		}

		// Win: full opponent points
		if CalculateSonnebornBerger(1.0, 5.0, scoring) != 5.0 {
			t.Error("Win should give full opponent points")
		}

		// Draw: half opponent points (traditional)
		if CalculateSonnebornBerger(0.5, 5.0, scoring) != 2.5 {
			t.Error("Draw should give half opponent points in traditional mode")
		}

		// Loss: 0 points
		if CalculateSonnebornBerger(0.0, 5.0, scoring) != 0.0 {
			t.Error("Loss should give 0 points")
		}
	})

	t.Run("Proportional mode with football scoring", func(t *testing.T) {
		scoring := FootballScoringSettings()
		scoring.SonnebornBergerMode = SBModeProportional

		// Win: full opponent points
		if CalculateSonnebornBerger(3.0, 9.0, scoring) != 9.0 {
			t.Error("Win should give full opponent points")
		}

		// Draw: proportional (1/3) opponent points
		expected := 9.0 * (1.0 / 3.0) // 3.0
		if CalculateSonnebornBerger(1.0, 9.0, scoring) != expected {
			t.Errorf("Draw should give proportional opponent points, got %f, want %f",
				CalculateSonnebornBerger(1.0, 9.0, scoring), expected)
		}

		// Loss: 0 points
		if CalculateSonnebornBerger(0.0, 9.0, scoring) != 0.0 {
			t.Error("Loss should give 0 points")
		}
	})
}

func TestFormatPoints(t *testing.T) {
	t.Run("Standard scoring format", func(t *testing.T) {
		scoring := DefaultScoringSettings()

		tests := []struct {
			points   float64
			expected string
		}{
			{0.0, "0.0"},
			{0.5, "0.5"},
			{1.0, "1.0"},
			{3.5, "3.5"},
		}

		for _, tt := range tests {
			result := FormatPoints(tt.points, scoring)
			if result != tt.expected {
				t.Errorf("FormatPoints(%f) = %s, want %s", tt.points, result, tt.expected)
			}
		}
	})

	t.Run("Football scoring format", func(t *testing.T) {
		scoring := FootballScoringSettings()

		tests := []struct {
			points   float64
			expected string
		}{
			{0.0, "0"},
			{1.0, "1"},
			{3.0, "3"},
			{15.0, "15"},
			{1.5, "1.5"}, // Non-integer should show decimal
		}

		for _, tt := range tests {
			result := FormatPoints(tt.points, scoring)
			if result != tt.expected {
				t.Errorf("FormatPoints(%f) = %s, want %s", tt.points, result, tt.expected)
			}
		}
	})
}

func TestGetByePoints(t *testing.T) {
	byeSettings := DefaultByeSettings()
	scoring := DefaultScoringSettings()

	if GetByePoints("system", byeSettings, scoring) != 1.0 {
		t.Error("System bye should give 1.0 points by default")
	}

	if GetByePoints("half_point", byeSettings, scoring) != 0.5 {
		t.Error("Half-point bye should give 0.5 points by default")
	}

	if GetByePoints("zero_point", byeSettings, scoring) != 0.0 {
		t.Error("Zero-point bye should give 0.0 points")
	}
}
