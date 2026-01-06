//go:build integration

package trf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestJaVaFoIntegration tests that generated TRF is accepted by JaVaFo.
// Requires JAVAFO_JAR_PATH environment variable to be set.
func TestJaVaFoIntegration(t *testing.T) {
	jarPath := os.Getenv("JAVAFO_JAR_PATH")
	if jarPath == "" {
		t.Skip("JAVAFO_JAR_PATH not set")
	}

	// Check java is available
	if _, err := exec.LookPath("java"); err != nil {
		t.Skip("Java not found in PATH")
	}

	g := NewJaVaFoGenerator()
	g.SetTournamentInfo("Integration Test Tournament", "Berlin", "GER", 5)

	// Add 8 players for round 1 pairing
	for i := 1; i <= 8; i++ {
		g.AddPlayer(TRFPlayer{
			StartNo:    i,
			Name:       fmt.Sprintf("Player %d", i),
			Rating:     2000 - i*50,
			Federation: "GER",
			Points:     0,
		})
	}

	trfContent := g.Generate()

	// Write to temp file
	tmpDir, err := os.MkdirTemp("", "javafo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.trf")
	outputPath := filepath.Join(tmpDir, "output.trf")

	if err := os.WriteFile(inputPath, []byte(trfContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Run JaVaFo
	cmd := exec.Command("java", "-jar", jarPath, inputPath, "-p", outputPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("JaVaFo failed: %v\nOutput: %s\nTRF:\n%s", err, output, trfContent)
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("JaVaFo did not produce output file")
	}
}

// TestBBPIntegration tests that generated TRF is accepted by bbpPairings.
// Requires BBP_PATH environment variable to be set.
func TestBBPIntegration(t *testing.T) {
	bbpPath := os.Getenv("BBP_PATH")
	if bbpPath == "" {
		t.Skip("BBP_PATH not set")
	}

	g := NewBBPGenerator()
	g.SetTournamentInfo("Integration Test Tournament", "Berlin", "GER", 5)
	g.SetInitialColorConfig(BBPColorWhite1)

	// Add 8 players for round 1 pairing
	for i := 1; i <= 8; i++ {
		g.AddPlayer(TRFPlayer{
			StartNo:    i,
			Name:       fmt.Sprintf("Player %d", i),
			Rating:     2000 - i*50,
			Federation: "GER",
			Points:     0,
		})
	}

	trfContent := g.Generate()

	// Write to temp file
	tmpDir, err := os.MkdirTemp("", "bbp-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.trf")
	outputPath := filepath.Join(tmpDir, "output.txt")

	if err := os.WriteFile(inputPath, []byte(trfContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Run bbpPairings
	cmd := exec.Command(bbpPath, "--dutch", inputPath, "-p", outputPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("bbpPairings failed: %v\nOutput: %s\nTRF:\n%s", err, output, trfContent)
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("bbpPairings did not produce output file")
	}
}

// TestJaVaFoRound2Integration tests pairing generation after round 1 results.
func TestJaVaFoRound2Integration(t *testing.T) {
	jarPath := os.Getenv("JAVAFO_JAR_PATH")
	if jarPath == "" {
		t.Skip("JAVAFO_JAR_PATH not set")
	}

	if _, err := exec.LookPath("java"); err != nil {
		t.Skip("Java not found in PATH")
	}

	g := NewJaVaFoGenerator()
	g.SetTournamentInfo("Round 2 Test", "Berlin", "GER", 5)

	// Players with round 1 results
	players := []TRFPlayer{
		{StartNo: 1, Name: "Player 1", Rating: 2000, Points: 1.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 5, Color: "w", Result: "1"}}},
		{StartNo: 2, Name: "Player 2", Rating: 1950, Points: 1.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 6, Color: "w", Result: "1"}}},
		{StartNo: 3, Name: "Player 3", Rating: 1900, Points: 1.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 7, Color: "w", Result: "1"}}},
		{StartNo: 4, Name: "Player 4", Rating: 1850, Points: 1.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 8, Color: "w", Result: "1"}}},
		{StartNo: 5, Name: "Player 5", Rating: 1800, Points: 0.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 1, Color: "b", Result: "0"}}},
		{StartNo: 6, Name: "Player 6", Rating: 1750, Points: 0.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 2, Color: "b", Result: "0"}}},
		{StartNo: 7, Name: "Player 7", Rating: 1700, Points: 0.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 3, Color: "b", Result: "0"}}},
		{StartNo: 8, Name: "Player 8", Rating: 1650, Points: 0.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 4, Color: "b", Result: "0"}}},
	}

	for _, p := range players {
		g.AddPlayer(p)
	}

	trfContent := g.Generate()

	tmpDir, err := os.MkdirTemp("", "javafo-r2-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.trf")
	outputPath := filepath.Join(tmpDir, "output.trf")

	if err := os.WriteFile(inputPath, []byte(trfContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("java", "-jar", jarPath, inputPath, "-p", outputPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("JaVaFo round 2 failed: %v\nOutput: %s\nTRF:\n%s", err, output, trfContent)
	}
}

// TestBBPRound2Integration tests bbpPairings after round 1 results.
func TestBBPRound2Integration(t *testing.T) {
	bbpPath := os.Getenv("BBP_PATH")
	if bbpPath == "" {
		t.Skip("BBP_PATH not set")
	}

	g := NewBBPGenerator()
	g.SetTournamentInfo("Round 2 Test", "Berlin", "GER", 5)

	// Players with round 1 results - note: BBP validates scores strictly
	players := []TRFPlayer{
		{StartNo: 1, Name: "Player 1", Rating: 2000, Points: 1.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 5, Color: "w", Result: "1"}}},
		{StartNo: 2, Name: "Player 2", Rating: 1950, Points: 1.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 6, Color: "w", Result: "1"}}},
		{StartNo: 3, Name: "Player 3", Rating: 1900, Points: 1.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 7, Color: "w", Result: "1"}}},
		{StartNo: 4, Name: "Player 4", Rating: 1850, Points: 1.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 8, Color: "w", Result: "1"}}},
		{StartNo: 5, Name: "Player 5", Rating: 1800, Points: 0.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 1, Color: "b", Result: "0"}}},
		{StartNo: 6, Name: "Player 6", Rating: 1750, Points: 0.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 2, Color: "b", Result: "0"}}},
		{StartNo: 7, Name: "Player 7", Rating: 1700, Points: 0.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 3, Color: "b", Result: "0"}}},
		{StartNo: 8, Name: "Player 8", Rating: 1650, Points: 0.0,
			RoundResults: []TRFRoundResult{{Round: 1, OpponentNo: 4, Color: "b", Result: "0"}}},
	}

	for _, p := range players {
		g.AddPlayer(p)
	}

	// Validate before generating
	errors := g.Validate()
	for _, e := range errors {
		if e.Fatal {
			t.Fatalf("Validation failed: %s", e.Message)
		}
	}

	trfContent := g.Generate()

	tmpDir, err := os.MkdirTemp("", "bbp-r2-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.trf")
	outputPath := filepath.Join(tmpDir, "output.txt")

	if err := os.WriteFile(inputPath, []byte(trfContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bbpPath, "--dutch", inputPath, "-p", outputPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("bbpPairings round 2 failed: %v\nOutput: %s\nTRF:\n%s", err, output, trfContent)
	}
}
