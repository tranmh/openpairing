package trf

import "time"

// TRFGenerator defines the interface for generating TRF files.
// Different pairing engines require different TRF formats:
// - JaVaFo uses TRFx format (fails on certain header codes)
// - bbpPairings uses TRFbx format (requires XXC records, strict score validation)
type TRFGenerator interface {
	// SetTournamentInfo sets basic tournament header information
	SetTournamentInfo(name, city, federation string, rounds int)

	// SetDates sets tournament start and end dates
	SetDates(start, end time.Time)

	// SetArbiters sets chief arbiter and deputy arbiters
	SetArbiters(chief string, deputies []string)

	// SetTimeControl sets the time control string
	SetTimeControl(tc string)

	// SetTournamentType sets the tournament type (Swiss, Round Robin, etc.)
	SetTournamentType(ttype string)

	// SetInitialColors sets initial color preferences (XXC records)
	SetInitialColors(colors ...string)

	// AddPlayer adds a player to the TRF file
	AddPlayer(player TRFPlayer)

	// AddTeam adds a team to the TRF file (for team tournaments)
	AddTeam(team TRFTeam)

	// Generate produces the TRF string output
	Generate() string

	// Validate validates the TRF data for the specific engine
	// Returns a list of validation errors (some may be fatal)
	Validate() []ValidationError

	// EngineName returns the name of the target engine
	EngineName() string
}

// ValidationError represents a TRF validation error
type ValidationError struct {
	Field   string // The field that failed validation
	Message string // Human-readable error message
	Fatal   bool   // If true, generation will fail for this engine
}
