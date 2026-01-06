package engine

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tranmh/openpairing/internal/pairing/trf"
	"github.com/rs/zerolog"
)

// JaVaFo is the FIDE-endorsed pairing engine
type JaVaFo struct {
	jarPath string
	logger  zerolog.Logger
	timeout time.Duration
}

// JaVaFoConfig contains configuration for JaVaFo
type JaVaFoConfig struct {
	JarPath string        // Path to javafo.jar
	Timeout time.Duration // Execution timeout
}

// NewJaVaFo creates a new JaVaFo pairing engine
func NewJaVaFo(cfg *JaVaFoConfig, logger zerolog.Logger) (*JaVaFo, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}

	// Check if JAR exists
	if _, err := os.Stat(cfg.JarPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("javafo.jar not found at %s", cfg.JarPath)
	}

	return &JaVaFo{
		jarPath: cfg.JarPath,
		logger:  logger.With().Str("engine", "javafo").Logger(),
		timeout: cfg.Timeout,
	}, nil
}

// PairingRequest contains the data needed to generate pairings
type PairingRequest struct {
	TRFData      string // TRF file content
	RoundNumber  int    // Round to generate pairings for
	PairingSystem string // dutch, burstein, dubov
}

// PairingResult contains the generated pairings
type PairingResult struct {
	Pairings   []Pairing
	TRFOutput  string
	Logs       string
	Duration   time.Duration
}

// Pairing represents a single pairing
type Pairing struct {
	Board        int
	WhiteStartNo int
	BlackStartNo int
	IsBye        bool
}

// GeneratePairings generates pairings using JaVaFo
func (j *JaVaFo) GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error) {
	startTime := time.Now()

	j.logger.Info().
		Int("round", req.RoundNumber).
		Str("system", req.PairingSystem).
		Msg("Generating pairings with JaVaFo")

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "javafo-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write input TRF file
	inputPath := filepath.Join(tmpDir, "input.trf")
	if err := os.WriteFile(inputPath, []byte(req.TRFData), 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	// Debug: Log TRF content
	j.logger.Debug().Str("trf_content", req.TRFData).Msg("Generated TRF for JaVaFo")

	// Prepare output path
	outputPath := filepath.Join(tmpDir, "output.trf")

	// Build command - JaVaFo syntax: java -jar javafo.jar input-file -p output-file
	args := []string{
		"-jar", j.jarPath,
		inputPath,          // Input TRF file (positional)
		"-p", outputPath,   // -p specifies output file for pairings
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, j.timeout)
	defer cancel()

	// Execute JaVaFo
	cmd := exec.CommandContext(ctx, "java", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	j.logger.Debug().
		Strs("args", args).
		Msg("Executing JaVaFo")

	if err := cmd.Run(); err != nil {
		j.logger.Error().
			Err(err).
			Str("stderr", stderr.String()).
			Msg("JaVaFo execution failed")
		return nil, fmt.Errorf("JaVaFo execution failed: %w - %s", err, stderr.String())
	}

	// Read output file - JaVaFo outputs compact pairing format with -p flag
	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output file: %w", err)
	}

	// Parse JaVaFo compact pairing format:
	// Line 1: number of pairings
	// Following lines: white_startno black_startno (or just startno for bye)
	lines := strings.Split(strings.TrimSpace(string(outputData)), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty output from JaVaFo")
	}

	var pairings []Pairing
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// First line is the number of pairings, skip it
		if i == 0 {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		board := i // Board number is 1-indexed from line position

		if len(parts) == 1 {
			// Bye: single player number
			startNo, err := strconv.Atoi(parts[0])
			if err != nil {
				j.logger.Warn().Str("line", line).Msg("Invalid bye pairing line")
				continue
			}
			pairings = append(pairings, Pairing{
				Board:        board,
				WhiteStartNo: startNo,
				BlackStartNo: 0,
				IsBye:        true,
			})
		} else if len(parts) >= 2 {
			// Regular pairing: white black
			white, err := strconv.Atoi(parts[0])
			if err != nil {
				j.logger.Warn().Str("line", line).Msg("Invalid pairing line (white)")
				continue
			}
			black, err := strconv.Atoi(parts[1])
			if err != nil {
				j.logger.Warn().Str("line", line).Msg("Invalid pairing line (black)")
				continue
			}
			pairings = append(pairings, Pairing{
				Board:        board,
				WhiteStartNo: white,
				BlackStartNo: black,
				IsBye:        false,
			})
		}
	}

	result := &PairingResult{
		Pairings:  pairings,
		TRFOutput: string(outputData),
		Logs:      stdout.String() + stderr.String(),
		Duration:  time.Since(startTime),
	}

	j.logger.Info().
		Int("pairings", len(pairings)).
		Dur("duration", result.Duration).
		Msg("Pairings generated successfully")

	return result, nil
}

// ValidatePairings validates that pairings follow FIDE rules
func (j *JaVaFo) ValidatePairings(ctx context.Context, req *PairingRequest, pairings []Pairing) ([]ValidationError, error) {
	var errors []ValidationError

	// Parse input TRF to get player data
	parser := trf.NewParser()
	inputTRF, err := parser.Parse(strings.NewReader(req.TRFData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse TRF: %w", err)
	}

	// Build player map
	players := make(map[int]*trf.TRFPlayer)
	for i := range inputTRF.Players {
		players[inputTRF.Players[i].StartNo] = &inputTRF.Players[i]
	}

	// Check for rematches
	for _, pairing := range pairings {
		if pairing.IsBye {
			continue
		}

		white := players[pairing.WhiteStartNo]
		black := players[pairing.BlackStartNo]

		if white == nil || black == nil {
			errors = append(errors, ValidationError{
				Type:    "invalid_player",
				Message: fmt.Sprintf("Invalid player in pairing: board %d", pairing.Board),
			})
			continue
		}

		// Check for rematch
		for _, result := range white.RoundResults {
			if result.OpponentNo == pairing.BlackStartNo {
				errors = append(errors, ValidationError{
					Type:    "rematch",
					Message: fmt.Sprintf("Rematch detected on board %d: %s vs %s (already played in round %d)",
						pairing.Board, white.Name, black.Name, result.Round),
					Board: pairing.Board,
				})
			}
		}
	}

	return errors, nil
}

// ValidationError represents a pairing validation error
type ValidationError struct {
	Type    string
	Message string
	Board   int
}

// GetVersion returns the JaVaFo version
func (j *JaVaFo) GetVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "java", "-jar", j.jarPath, "-v")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// IsAvailable checks if JaVaFo is available and working
func (j *JaVaFo) IsAvailable(ctx context.Context) bool {
	// Check if java is available
	if _, err := exec.LookPath("java"); err != nil {
		j.logger.Warn().Msg("Java not found in PATH")
		return false
	}

	// Check if JAR exists
	if _, err := os.Stat(j.jarPath); os.IsNotExist(err) {
		j.logger.Warn().Str("path", j.jarPath).Msg("JaVaFo JAR not found")
		return false
	}

	// Try to get version
	_, err := j.GetVersion(ctx)
	return err == nil
}

// Name returns the engine name
func (j *JaVaFo) Name() string {
	return "JaVaFo"
}

// SupportsAcceleration returns whether the engine supports accelerated pairing
func (j *JaVaFo) SupportsAcceleration() bool {
	return true
}

// SupportedSystems returns the supported pairing systems
func (j *JaVaFo) SupportedSystems() []string {
	return []string{"dutch", "burstein", "dubov"}
}

// GetTRFGenerator returns a TRF generator configured for JaVaFo's requirements.
// JaVaFo uses TRFx format which:
//   - Fails on unrecognized header codes (062, 072, 082, 092, 102, 112)
//   - Requires XXR record for round count
//   - XXC records are optional
func (j *JaVaFo) GetTRFGenerator() trf.TRFGenerator {
	return trf.NewJaVaFoGenerator()
}

// Verify JaVaFo implements TRFAwarePairingEngine
var _ TRFAwarePairingEngine = (*JaVaFo)(nil)
