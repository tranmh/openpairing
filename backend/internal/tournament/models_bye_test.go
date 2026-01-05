package tournament

import (
	"encoding/json"
	"testing"
)

func TestDefaultByeSettings(t *testing.T) {
	settings := DefaultByeSettings()

	if settings == nil {
		t.Fatal("DefaultByeSettings returned nil")
	}

	// Test Late Entry defaults (FIDE compliant)
	t.Run("LateEntry defaults", func(t *testing.T) {
		if !settings.LateEntry.Allowed {
			t.Error("LateEntry.Allowed should be true")
		}
		if settings.LateEntry.MaxRound != 2 {
			t.Errorf("LateEntry.MaxRound = %d, want 2", settings.LateEntry.MaxRound)
		}
		if settings.LateEntry.MissedRoundPoints != 0 {
			t.Errorf("LateEntry.MissedRoundPoints = %f, want 0", settings.LateEntry.MissedRoundPoints)
		}
		if !settings.LateEntry.RequiresApproval {
			t.Error("LateEntry.RequiresApproval should be true")
		}
		if settings.LateEntry.PairingPriority != "bottom" {
			t.Errorf("LateEntry.PairingPriority = %s, want bottom", settings.LateEntry.PairingPriority)
		}
	})

	// Test Half Point Bye defaults (FIDE compliant)
	t.Run("HalfPointBye defaults", func(t *testing.T) {
		if !settings.HalfPointBye.Allowed {
			t.Error("HalfPointBye.Allowed should be true")
		}
		if settings.HalfPointBye.MaxPerPlayer != 1 {
			t.Errorf("HalfPointBye.MaxPerPlayer = %d, want 1", settings.HalfPointBye.MaxPerPlayer)
		}
		if settings.HalfPointBye.Points != 0.5 {
			t.Errorf("HalfPointBye.Points = %f, want 0.5", settings.HalfPointBye.Points)
		}
		if settings.HalfPointBye.AllowedRounds != "first_half" {
			t.Errorf("HalfPointBye.AllowedRounds = %s, want first_half", settings.HalfPointBye.AllowedRounds)
		}
	})

	// Test System Bye defaults (FIDE compliant)
	t.Run("SystemBye defaults", func(t *testing.T) {
		if settings.SystemBye.Points != 1.0 {
			t.Errorf("SystemBye.Points = %f, want 1.0", settings.SystemBye.Points)
		}
		if settings.SystemBye.Selection != SystemByeLowestScoreThenRating {
			t.Errorf("SystemBye.Selection = %s, want %s", settings.SystemBye.Selection, SystemByeLowestScoreThenRating)
		}
		if settings.SystemBye.MaxPerPlayer != 1 {
			t.Errorf("SystemBye.MaxPerPlayer = %d, want 1", settings.SystemBye.MaxPerPlayer)
		}
		if !settings.SystemBye.CountsForBuchholz {
			t.Error("SystemBye.CountsForBuchholz should be true")
		}
	})

	// Test Forfeit defaults (FIDE compliant)
	t.Run("Forfeit defaults", func(t *testing.T) {
		if settings.Forfeit.WinnerPoints != 1.0 {
			t.Errorf("Forfeit.WinnerPoints = %f, want 1.0", settings.Forfeit.WinnerPoints)
		}
		if settings.Forfeit.LoserPoints != 0 {
			t.Errorf("Forfeit.LoserPoints = %f, want 0", settings.Forfeit.LoserPoints)
		}
		if settings.Forfeit.AutoWithdrawAfter != 2 {
			t.Errorf("Forfeit.AutoWithdrawAfter = %d, want 2", settings.Forfeit.AutoWithdrawAfter)
		}
		if !settings.Forfeit.ConsecutiveOnly {
			t.Error("Forfeit.ConsecutiveOnly should be true")
		}
	})

	// Test Withdrawal defaults (FIDE compliant)
	t.Run("Withdrawal defaults", func(t *testing.T) {
		if !settings.Withdrawal.Allowed {
			t.Error("Withdrawal.Allowed should be true")
		}
		if settings.Withdrawal.BuchholzMethod != BuchholzVirtual50Percent {
			t.Errorf("Withdrawal.BuchholzMethod = %s, want %s", settings.Withdrawal.BuchholzMethod, BuchholzVirtual50Percent)
		}
		if settings.Withdrawal.VirtualScorePercent != 50 {
			t.Errorf("Withdrawal.VirtualScorePercent = %f, want 50", settings.Withdrawal.VirtualScorePercent)
		}
	})
}

func TestDefaultByeSettings_JSONRoundTrip(t *testing.T) {
	settings := DefaultByeSettings()

	// Serialize to JSON
	jsonBytes, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal ByeSettings: %v", err)
	}

	// Deserialize back
	var decoded ByeSettings
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ByeSettings: %v", err)
	}

	// Verify key fields survived round-trip
	if decoded.HalfPointBye.MaxPerPlayer != settings.HalfPointBye.MaxPerPlayer {
		t.Errorf("HalfPointBye.MaxPerPlayer changed after JSON round-trip: got %d, want %d",
			decoded.HalfPointBye.MaxPerPlayer, settings.HalfPointBye.MaxPerPlayer)
	}
	if decoded.SystemBye.Points != settings.SystemBye.Points {
		t.Errorf("SystemBye.Points changed after JSON round-trip: got %f, want %f",
			decoded.SystemBye.Points, settings.SystemBye.Points)
	}
}

func TestValidateFIDECompliance_NoViolations(t *testing.T) {
	settings := DefaultByeSettings()
	violations := settings.ValidateFIDECompliance(9) // 9 round tournament

	if len(violations) != 0 {
		t.Errorf("Default settings should have no FIDE violations, got %d: %+v", len(violations), violations)
	}
}

func TestValidateFIDECompliance_LateEntryViolations(t *testing.T) {
	settings := DefaultByeSettings()

	t.Run("MaxRound exceeds half", func(t *testing.T) {
		settings.LateEntry.MaxRound = 6 // More than half of 9 rounds
		violations := settings.ValidateFIDECompliance(9)

		found := false
		for _, v := range violations {
			if v.Setting == "lateEntry.maxRound" {
				found = true
				if v.Severity != "warning" {
					t.Errorf("Expected severity 'warning', got %s", v.Severity)
				}
				break
			}
		}
		if !found {
			t.Error("Expected violation for lateEntry.maxRound not found")
		}
	})

	t.Run("MissedRoundPoints non-zero", func(t *testing.T) {
		settings.LateEntry.MaxRound = 2 // Reset
		settings.LateEntry.MissedRoundPoints = 0.5
		violations := settings.ValidateFIDECompliance(9)

		found := false
		for _, v := range violations {
			if v.Setting == "lateEntry.missedRoundPoints" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected violation for lateEntry.missedRoundPoints not found")
		}
	})
}

func TestValidateFIDECompliance_HalfPointByeViolations(t *testing.T) {
	settings := DefaultByeSettings()

	t.Run("MaxPerPlayer greater than 1", func(t *testing.T) {
		settings.HalfPointBye.MaxPerPlayer = 3
		violations := settings.ValidateFIDECompliance(9)

		found := false
		for _, v := range violations {
			if v.Setting == "halfPointBye.maxPerPlayer" {
				found = true
				if v.FIDEValue != 1 {
					t.Errorf("Expected FIDE value 1, got %v", v.FIDEValue)
				}
				break
			}
		}
		if !found {
			t.Error("Expected violation for halfPointBye.maxPerPlayer not found")
		}
	})

	t.Run("Points not 0.5", func(t *testing.T) {
		settings.HalfPointBye.MaxPerPlayer = 1 // Reset
		settings.HalfPointBye.Points = 1.0      // Full point bye
		violations := settings.ValidateFIDECompliance(9)

		found := false
		for _, v := range violations {
			if v.Setting == "halfPointBye.points" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected violation for halfPointBye.points not found")
		}
	})

	t.Run("AllowedRounds is 'all'", func(t *testing.T) {
		settings.HalfPointBye.Points = 0.5 // Reset
		settings.HalfPointBye.AllowedRounds = "all"
		violations := settings.ValidateFIDECompliance(9)

		found := false
		for _, v := range violations {
			if v.Setting == "halfPointBye.allowedRounds" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected violation for halfPointBye.allowedRounds not found")
		}
	})
}

func TestValidateFIDECompliance_SystemByeViolations(t *testing.T) {
	settings := DefaultByeSettings()

	t.Run("Points not 1.0", func(t *testing.T) {
		settings.SystemBye.Points = 0.5
		violations := settings.ValidateFIDECompliance(9)

		found := false
		for _, v := range violations {
			if v.Setting == "systemBye.points" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected violation for systemBye.points not found")
		}
	})

	t.Run("MaxPerPlayer greater than 1", func(t *testing.T) {
		settings.SystemBye.Points = 1.0 // Reset
		settings.SystemBye.MaxPerPlayer = 2
		violations := settings.ValidateFIDECompliance(9)

		found := false
		for _, v := range violations {
			if v.Setting == "systemBye.maxPerPlayer" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected violation for systemBye.maxPerPlayer not found")
		}
	})
}

func TestValidateFIDECompliance_ForfeitViolations(t *testing.T) {
	settings := DefaultByeSettings()

	t.Run("AutoWithdrawAfter disabled", func(t *testing.T) {
		settings.Forfeit.AutoWithdrawAfter = 0
		violations := settings.ValidateFIDECompliance(9)

		found := false
		for _, v := range violations {
			if v.Setting == "forfeit.autoWithdrawAfter" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected violation for forfeit.autoWithdrawAfter not found")
		}
	})

	t.Run("ConsecutiveOnly false", func(t *testing.T) {
		settings.Forfeit.AutoWithdrawAfter = 2 // Reset
		settings.Forfeit.ConsecutiveOnly = false
		violations := settings.ValidateFIDECompliance(9)

		found := false
		for _, v := range violations {
			if v.Setting == "forfeit.consecutiveOnly" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected violation for forfeit.consecutiveOnly not found")
		}
	})
}

func TestValidateFIDECompliance_MultipleViolations(t *testing.T) {
	settings := &ByeSettings{
		LateEntry: LateEntrySettings{
			Allowed:           true,
			MaxRound:          8, // Violation: more than half of 9
			MissedRoundPoints: 1, // Violation: should be 0
		},
		HalfPointBye: HalfPointByeSettings{
			Allowed:       true,
			MaxPerPlayer:  5,     // Violation: should be 1
			Points:        1.0,   // Violation: should be 0.5
			AllowedRounds: "all", // Violation: should be first_half
		},
		SystemBye: SystemByeSettings{
			Points:       0.5, // Violation: should be 1.0
			MaxPerPlayer: 3,   // Violation: should be 1
		},
		Forfeit: ForfeitSettings{
			AutoWithdrawAfter: 0,     // Violation: should be > 0
			ConsecutiveOnly:   false, // Violation: should be true
		},
		Withdrawal: WithdrawalSettings{
			Allowed: true,
		},
	}

	violations := settings.ValidateFIDECompliance(9)

	// Should have at least 8 violations
	if len(violations) < 8 {
		t.Errorf("Expected at least 8 violations, got %d", len(violations))
		for _, v := range violations {
			t.Logf("Violation: %s = %v (FIDE: %v)", v.Setting, v.Value, v.FIDEValue)
		}
	}
}

func TestValidateFIDECompliance_EdgeCases(t *testing.T) {
	t.Run("Zero rounds tournament", func(t *testing.T) {
		settings := DefaultByeSettings()
		violations := settings.ValidateFIDECompliance(0)
		// Should not panic and handle gracefully
		_ = violations
	})

	t.Run("Single round tournament", func(t *testing.T) {
		settings := DefaultByeSettings()
		violations := settings.ValidateFIDECompliance(1)
		// MaxRound of 2 is more than half of 1, should trigger violation
		found := false
		for _, v := range violations {
			if v.Setting == "lateEntry.maxRound" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected violation for lateEntry.maxRound in single round tournament")
		}
	})

	t.Run("Large tournament", func(t *testing.T) {
		settings := DefaultByeSettings()
		settings.LateEntry.MaxRound = 5
		violations := settings.ValidateFIDECompliance(20)
		// 5 is <= half of 20, so no violation
		found := false
		for _, v := range violations {
			if v.Setting == "lateEntry.maxRound" {
				found = true
				break
			}
		}
		if found {
			t.Error("Should not have violation for lateEntry.maxRound when <= half rounds")
		}
	})
}

func TestParticipant_EffectiveRating(t *testing.T) {
	t.Run("No override uses StartingRating", func(t *testing.T) {
		p := &Participant{
			StartingRating:       1500,
			ManualRatingOverride: nil,
		}
		if p.EffectiveRating() != 1500 {
			t.Errorf("EffectiveRating() = %d, want 1500", p.EffectiveRating())
		}
	})

	t.Run("Override takes precedence", func(t *testing.T) {
		override := 1800
		p := &Participant{
			StartingRating:       1500,
			ManualRatingOverride: &override,
		}
		if p.EffectiveRating() != 1800 {
			t.Errorf("EffectiveRating() = %d, want 1800", p.EffectiveRating())
		}
	})

	t.Run("Zero override is valid", func(t *testing.T) {
		override := 0
		p := &Participant{
			StartingRating:       1500,
			ManualRatingOverride: &override,
		}
		if p.EffectiveRating() != 0 {
			t.Errorf("EffectiveRating() = %d, want 0", p.EffectiveRating())
		}
	})
}

func TestByeSettings_NilHandling(t *testing.T) {
	t.Run("Nil ByeSettings pointer", func(t *testing.T) {
		var settings *ByeSettings
		// This should panic or be handled - let's see behavior
		defer func() {
			if r := recover(); r != nil {
				// Expected panic for nil pointer
				t.Log("Expected panic for nil pointer dereference")
			}
		}()

		// This will panic if not handled
		_ = settings.ValidateFIDECompliance(9)
		t.Error("Expected panic for nil ByeSettings")
	})
}

func TestByeTrackingFields(t *testing.T) {
	p := &Participant{
		HalfPointByesUsed:   1,
		ZeroPointByesUsed:   2,
		ForfeitCount:        1,
		ConsecutiveForfeits: 1,
	}

	t.Run("Initial values set correctly", func(t *testing.T) {
		if p.HalfPointByesUsed != 1 {
			t.Errorf("HalfPointByesUsed = %d, want 1", p.HalfPointByesUsed)
		}
		if p.ZeroPointByesUsed != 2 {
			t.Errorf("ZeroPointByesUsed = %d, want 2", p.ZeroPointByesUsed)
		}
		if p.ForfeitCount != 1 {
			t.Errorf("ForfeitCount = %d, want 1", p.ForfeitCount)
		}
		if p.ConsecutiveForfeits != 1 {
			t.Errorf("ConsecutiveForfeits = %d, want 1", p.ConsecutiveForfeits)
		}
	})

	t.Run("Optional fields are nil by default", func(t *testing.T) {
		p2 := &Participant{}
		if p2.SystemByeRound != nil {
			t.Error("SystemByeRound should be nil by default")
		}
		if p2.LateEntryRound != nil {
			t.Error("LateEntryRound should be nil by default")
		}
		if p2.WithdrawnRound != nil {
			t.Error("WithdrawnRound should be nil by default")
		}
		if p2.WithdrawalReason != nil {
			t.Error("WithdrawalReason should be nil by default")
		}
	})
}

func TestByeRequests_SliceOperations(t *testing.T) {
	p := &Participant{}

	t.Run("Initial nil slice", func(t *testing.T) {
		if p.ByeRequests != nil {
			t.Error("ByeRequests should be nil initially")
		}
	})

	t.Run("Append to nil slice", func(t *testing.T) {
		p.ByeRequests = append(p.ByeRequests, 3)
		if len(p.ByeRequests) != 1 {
			t.Errorf("len(ByeRequests) = %d, want 1", len(p.ByeRequests))
		}
		if p.ByeRequests[0] != 3 {
			t.Errorf("ByeRequests[0] = %d, want 3", p.ByeRequests[0])
		}
	})

	t.Run("Multiple bye requests", func(t *testing.T) {
		p.ByeRequests = append(p.ByeRequests, 5, 7)
		if len(p.ByeRequests) != 3 {
			t.Errorf("len(ByeRequests) = %d, want 3", len(p.ByeRequests))
		}
	})

	t.Run("Remove from middle", func(t *testing.T) {
		// Remove round 5 (index 1)
		p.ByeRequests = append(p.ByeRequests[:1], p.ByeRequests[2:]...)
		if len(p.ByeRequests) != 2 {
			t.Errorf("len(ByeRequests) = %d, want 2", len(p.ByeRequests))
		}
		if p.ByeRequests[0] != 3 || p.ByeRequests[1] != 7 {
			t.Errorf("ByeRequests = %v, want [3, 7]", p.ByeRequests)
		}
	})
}
