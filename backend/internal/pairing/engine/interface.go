package engine

import "context"

// PairingEngine is the interface for pairing engines
type PairingEngine interface {
	GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error)
	Name() string
	IsAvailable(ctx context.Context) bool
}
