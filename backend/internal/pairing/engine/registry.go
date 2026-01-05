package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
)

// Registry manages all available pairing engines
type Registry struct {
	mu      sync.RWMutex
	engines map[string]PairingEngine
	general map[string]GeneralPairingEngine
	logger  zerolog.Logger
}

// RegistryConfig contains configuration for all engines
type RegistryConfig struct {
	JaVaFoPath   string
	BBPPath      string
	ArenaConfig  *ArenaConfig
}

// NewRegistry creates a new engine registry with all available engines
func NewRegistry(cfg *RegistryConfig, logger zerolog.Logger) *Registry {
	r := &Registry{
		engines: make(map[string]PairingEngine),
		general: make(map[string]GeneralPairingEngine),
		logger:  logger.With().Str("component", "engine-registry").Logger(),
	}

	// Register built-in engines (always available)
	r.registerBuiltInEngines(cfg)

	// Register external engines (JaVaFo, BBP) if configured
	r.registerExternalEngines(cfg)

	return r
}

// registerBuiltInEngines registers all built-in pairing engines
func (r *Registry) registerBuiltInEngines(cfg *RegistryConfig) {
	// Round Robin
	rr := NewRoundRobin(r.logger)
	r.general["round_robin"] = rr
	r.engines["round_robin"] = rr

	// Knockout
	ko := NewKnockout(r.logger)
	r.general["knockout"] = ko
	r.engines["knockout"] = ko

	// Scheveningen
	sch := NewScheveningen(r.logger)
	r.general["scheveningen"] = sch
	r.engines["scheveningen"] = sch

	// Team Swiss
	ts := NewTeamSwiss(r.logger)
	r.general["team_swiss"] = ts
	r.engines["team_swiss"] = ts

	// Monrad
	mon := NewMonrad(r.logger)
	r.general["monrad"] = mon
	r.engines["monrad"] = mon

	// Keizer
	kei := NewKeizer(r.logger)
	r.general["keizer"] = kei
	r.engines["keizer"] = kei

	// Arena
	arenaConfig := cfg.ArenaConfig
	if arenaConfig == nil {
		arenaConfig = &ArenaConfig{
			MaxRatingDiff:      300,
			PreventRematches:   3,
		}
	}
	arena := NewArena(r.logger, arenaConfig)
	r.general["arena"] = arena
	r.engines["arena"] = arena

	r.logger.Info().
		Int("builtin_engines", 7).
		Msg("Registered built-in pairing engines")
}

// registerExternalEngines registers external pairing engines (JaVaFo, BBP)
func (r *Registry) registerExternalEngines(cfg *RegistryConfig) {
	// JaVaFo (FIDE-endorsed Swiss pairing)
	if cfg.JaVaFoPath != "" {
		javafo, err := NewJaVaFo(&JaVaFoConfig{JarPath: cfg.JaVaFoPath}, r.logger)
		if err != nil {
			r.logger.Warn().Err(err).Msg("Failed to initialize JaVaFo engine")
		} else {
			r.engines["javafo"] = javafo
			r.engines["swiss"] = javafo // Default Swiss engine
			r.engines["dutch"] = javafo
			r.logger.Info().Str("path", cfg.JaVaFoPath).Msg("JaVaFo engine registered")
		}
	}

	// bbpPairings
	if cfg.BBPPath != "" {
		bbp, err := NewBBPPairings(&BBPConfig{ExecPath: cfg.BBPPath}, r.logger)
		if err != nil {
			r.logger.Warn().Err(err).Msg("Failed to initialize bbpPairings engine")
		} else {
			r.engines["bbp"] = bbp
			// If JaVaFo not available, BBP becomes default Swiss
			if _, hasSwiss := r.engines["swiss"]; !hasSwiss {
				r.engines["swiss"] = bbp
			}
			r.logger.Info().Str("path", cfg.BBPPath).Msg("bbpPairings engine registered")
		}
	}
}

// GetEngine returns a pairing engine by name
func (r *Registry) GetEngine(name string) (PairingEngine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if eng, ok := r.engines[name]; ok {
		return eng, nil
	}
	return nil, fmt.Errorf("pairing engine not found: %s", name)
}

// GetGeneralEngine returns a general pairing engine by name
func (r *Registry) GetGeneralEngine(name string) (GeneralPairingEngine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if eng, ok := r.general[name]; ok {
		return eng, nil
	}
	return nil, fmt.Errorf("general pairing engine not found: %s", name)
}

// GetEngineForTournament returns the appropriate engine for a tournament type
func (r *Registry) GetEngineForTournament(tournamentType TournamentType, engineName string) (PairingEngine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Map tournament type to engine name if not specified
	if engineName == "" {
		engineName = r.defaultEngineForType(tournamentType)
	}

	// Try to find the specific engine
	if eng, ok := r.engines[engineName]; ok {
		return eng, nil
	}

	// Fall back to tournament type as engine name
	if eng, ok := r.engines[string(tournamentType)]; ok {
		return eng, nil
	}

	return nil, fmt.Errorf("no engine available for tournament type %s (requested: %s)", tournamentType, engineName)
}

// GetGeneralEngineForTournament returns a general engine for a tournament type
func (r *Registry) GetGeneralEngineForTournament(tournamentType TournamentType) (GeneralPairingEngine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First try direct match
	if eng, ok := r.general[string(tournamentType)]; ok {
		return eng, nil
	}

	return nil, fmt.Errorf("no general engine available for tournament type %s", tournamentType)
}

// defaultEngineForType returns the default engine name for a tournament type
func (r *Registry) defaultEngineForType(tournamentType TournamentType) string {
	switch tournamentType {
	case TournamentTypeSwiss:
		return "swiss" // Will use JaVaFo or BBP
	case TournamentTypeRoundRobin:
		return "round_robin"
	case TournamentTypeKnockout:
		return "knockout"
	case TournamentTypeTeamSwiss:
		return "team_swiss"
	case TournamentTypeScheveningen:
		return "scheveningen"
	case TournamentTypeArena:
		return "arena"
	default:
		return "swiss"
	}
}

// ListAvailableEngines returns a list of all available engine names
func (r *Registry) ListAvailableEngines() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.engines))
	for name := range r.engines {
		names = append(names, name)
	}
	return names
}

// IsEngineAvailable checks if an engine is available
func (r *Registry) IsEngineAvailable(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if eng, ok := r.engines[name]; ok {
		return eng.IsAvailable(context.Background())
	}
	return false
}

// HasSwissEngine returns true if a Swiss pairing engine is available
func (r *Registry) HasSwissEngine() bool {
	return r.IsEngineAvailable("javafo") || r.IsEngineAvailable("bbp")
}

// GetSwissEngine returns the best available Swiss engine
func (r *Registry) GetSwissEngine() (PairingEngine, error) {
	// Prefer JaVaFo (FIDE-endorsed)
	if eng, err := r.GetEngine("javafo"); err == nil {
		return eng, nil
	}
	// Fall back to BBP
	if eng, err := r.GetEngine("bbp"); err == nil {
		return eng, nil
	}
	// Fall back to Monrad (built-in Swiss variant)
	if eng, err := r.GetEngine("monrad"); err == nil {
		return eng, nil
	}
	return nil, fmt.Errorf("no Swiss pairing engine available")
}

// EngineInfo contains information about an engine
type EngineInfo struct {
	Name        string
	Available   bool
	Types       []TournamentType
	Description string
}

// GetEngineInfo returns information about all registered engines
func (r *Registry) GetEngineInfo() []EngineInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ctx := context.Background()
	seen := make(map[string]bool)
	var infos []EngineInfo

	for name, eng := range r.engines {
		if seen[name] {
			continue
		}
		seen[name] = true

		info := EngineInfo{
			Name:      name,
			Available: eng.IsAvailable(ctx),
		}

		// Add supported types for general engines
		if geng, ok := eng.(GeneralPairingEngine); ok {
			info.Types = geng.SupportedTournamentTypes()
		}

		switch name {
		case "javafo":
			info.Description = "FIDE-endorsed JaVaFo Swiss pairing engine"
		case "bbp":
			info.Description = "bbpPairings Swiss pairing engine"
		case "round_robin":
			info.Description = "Round Robin (Berger tables)"
		case "knockout":
			info.Description = "Single/Double elimination brackets"
		case "scheveningen":
			info.Description = "Team vs Team (Scheveningen system)"
		case "team_swiss":
			info.Description = "Team Swiss (Olympiad style)"
		case "monrad":
			info.Description = "Monrad Swiss variant"
		case "keizer":
			info.Description = "Keizer system (Dutch clubs)"
		case "arena":
			info.Description = "Arena/Continuous pairing"
		}

		infos = append(infos, info)
	}

	return infos
}
