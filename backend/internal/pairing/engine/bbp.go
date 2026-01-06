package engine

import (
	"bufio"
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

// BBPPairings is the bbpPairings engine
type BBPPairings struct {
	execPath string
	logger   zerolog.Logger
	timeout  time.Duration
}

// BBPConfig contains configuration for bbpPairings
type BBPConfig struct {
	ExecPath string        // Path to bbpPairings executable
	Timeout  time.Duration // Execution timeout
}

// NewBBPPairings creates a new bbpPairings engine
func NewBBPPairings(cfg *BBPConfig, logger zerolog.Logger) (*BBPPairings, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}

	// Check if executable exists
	if _, err := os.Stat(cfg.ExecPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("bbpPairings not found at %s", cfg.ExecPath)
	}

	return &BBPPairings{
		execPath: cfg.ExecPath,
		logger:   logger.With().Str("engine", "bbpPairings").Logger(),
		timeout:  cfg.Timeout,
	}, nil
}

// GeneratePairings generates pairings using bbpPairings
func (b *BBPPairings) GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error) {
	startTime := time.Now()

	b.logger.Info().
		Int("round", req.RoundNumber).
		Str("system", req.PairingSystem).
		Msg("Generating pairings with bbpPairings")

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "bbp-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write input TRF file
	// NOTE: The TRF data should already contain XXC record from BBPGenerator.
	// If using GetTRFGenerator(), BBPGenerator automatically adds XXC.
	inputPath := filepath.Join(tmpDir, "input.trf")
	if err := os.WriteFile(inputPath, []byte(req.TRFData), 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	// Prepare output path
	outputPath := filepath.Join(tmpDir, "output.txt")

	// Build command - bbpPairings syntax: bbpPairings --dutch input-file -p output-file
	args := []string{}

	// Add pairing system
	switch req.PairingSystem {
	case "dutch", "":
		args = append(args, "--dutch")
	case "burstein":
		args = append(args, "--burstein")
	default:
		args = append(args, "--dutch")
	}

	args = append(args, inputPath, "-p", outputPath)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	// Execute bbpPairings
	cmd := exec.CommandContext(ctx, b.execPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	b.logger.Debug().
		Strs("args", args).
		Msg("Executing bbpPairings")

	if err := cmd.Run(); err != nil {
		b.logger.Error().
			Err(err).
			Str("stderr", stderr.String()).
			Str("stdout", stdout.String()).
			Msg("bbpPairings execution failed")
		return nil, fmt.Errorf("bbpPairings execution failed: %w - %s", err, stderr.String())
	}

	// Read output file
	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output file: %w", err)
	}

	// Parse output - simple format:
	// First line: number of pairings
	// Following lines: "white_player black_player" (0 = bye)
	pairings, err := b.parseOutput(string(outputData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}

	result := &PairingResult{
		Pairings:  pairings,
		TRFOutput: string(outputData),
		Logs:      stdout.String() + stderr.String(),
		Duration:  time.Since(startTime),
	}

	b.logger.Info().
		Int("pairings", len(pairings)).
		Dur("duration", result.Duration).
		Msg("Pairings generated successfully")

	return result, nil
}

// parseOutput parses bbpPairings output format
func (b *BBPPairings) parseOutput(output string) ([]Pairing, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))

	// First line is count
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty output")
	}
	countStr := strings.TrimSpace(scanner.Text())
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return nil, fmt.Errorf("invalid count: %s", countStr)
	}

	pairings := make([]Pairing, 0, count)
	board := 1

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid pairing line: %s", line)
		}

		white, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid white player: %s", parts[0])
		}

		black, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid black player: %s", parts[1])
		}

		pairing := Pairing{
			Board:        board,
			WhiteStartNo: white,
			BlackStartNo: black,
			IsBye:        black == 0,
		}
		pairings = append(pairings, pairing)
		board++
	}

	return pairings, nil
}

// Name returns the engine name
func (b *BBPPairings) Name() string {
	return "bbpPairings"
}

// IsAvailable checks if bbpPairings is available
func (b *BBPPairings) IsAvailable(ctx context.Context) bool {
	if _, err := os.Stat(b.execPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// GetTRFGenerator returns a TRF generator configured for bbpPairings' requirements.
// bbpPairings uses TRFbx format which:
//   - REQUIRES XXC record for initial color configuration
//   - Validates player scores strictly against tournament results
//   - Only supports Dutch and Burstein pairing systems
func (b *BBPPairings) GetTRFGenerator() trf.TRFGenerator {
	return trf.NewBBPGenerator()
}

// Verify BBPPairings implements TRFAwarePairingEngine
var _ TRFAwarePairingEngine = (*BBPPairings)(nil)
