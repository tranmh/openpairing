package engine

import (
	"context"

	"github.com/openpairing/openpairing/internal/pairing/trf"
)

// PairingEngine is the interface for pairing engines
type PairingEngine interface {
	GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error)
	Name() string
	IsAvailable(ctx context.Context) bool
}

// TRFAwarePairingEngine extends PairingEngine with TRF generation capability.
// Engines that use TRF format (JaVaFo, bbpPairings) implement this interface
// to provide engine-specific TRF generators.
type TRFAwarePairingEngine interface {
	PairingEngine
	// GetTRFGenerator returns a TRF generator configured for this engine's requirements
	GetTRFGenerator() trf.TRFGenerator
}
