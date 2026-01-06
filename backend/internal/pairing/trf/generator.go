package trf

import (
	"fmt"
	"time"
)

// Generator generates TRF files using JaVaFo-compatible format by default.
// For engine-specific TRF generation, use NewTRFGenerator(engine) instead.
type Generator struct {
	impl TRFGenerator
}

// NewGenerator creates a new TRF generator using JaVaFo-compatible format.
// For engine-specific TRF generation, use NewTRFGenerator(engine) instead.
func NewGenerator() *Generator {
	return &Generator{
		impl: NewJaVaFoGenerator(),
	}
}

// SetTournamentInfo sets tournament header information
func (g *Generator) SetTournamentInfo(name, city, federation string, rounds int) {
	g.impl.SetTournamentInfo(name, city, federation, rounds)
}

// SetDates sets tournament dates
func (g *Generator) SetDates(start, end time.Time) {
	g.impl.SetDates(start, end)
}

// SetArbiters sets the arbiters
func (g *Generator) SetArbiters(chief string, deputies []string) {
	g.impl.SetArbiters(chief, deputies)
}

// SetTimeControl sets the time control string
func (g *Generator) SetTimeControl(tc string) {
	g.impl.SetTimeControl(tc)
}

// SetTournamentType sets the tournament type
func (g *Generator) SetTournamentType(ttype string) {
	g.impl.SetTournamentType(ttype)
}

// SetInitialColors sets the initial color configuration for bbpPairings
func (g *Generator) SetInitialColors(colors ...string) {
	g.impl.SetInitialColors(colors...)
}

// SetRounds sets the number of rounds (XXR record)
func (g *Generator) SetRounds(rounds int) {
	// Access the underlying JaVaFoGenerator to set rounds directly
	if jg, ok := g.impl.(*JaVaFoGenerator); ok {
		jg.trf.Rounds = rounds
	}
}

// AddPlayer adds a player to the TRF file
func (g *Generator) AddPlayer(player TRFPlayer) {
	g.impl.AddPlayer(player)
}

// AddTeam adds a team to the TRF file
func (g *Generator) AddTeam(team TRFTeam) {
	g.impl.AddTeam(team)
}

// Generate creates the TRF string
func (g *Generator) Generate() string {
	return g.impl.Generate()
}

// ValidateForFIDE validates the TRF file for FIDE submission
func (g *Generator) ValidateForFIDE() []string {
	errors := g.impl.Validate()
	var result []string
	for _, err := range errors {
		if err.Fatal {
			result = append(result, err.Message)
		}
	}
	return result
}

// Validate implements TRFGenerator interface
func (g *Generator) Validate() []ValidationError {
	return g.impl.Validate()
}

// EngineName implements TRFGenerator interface
func (g *Generator) EngineName() string {
	return g.impl.EngineName()
}

// Ensure Generator implements TRFGenerator
var _ TRFGenerator = (*Generator)(nil)

// TournamentData represents tournament data for TRF generation
type TournamentData struct {
	Name           string
	City           string
	Federation     string
	StartDate      time.Time
	EndDate        time.Time
	Rounds         int
	TournamentType string
	TimeControl    string
	ChiefArbiter   string
	Players        []PlayerData
}

// PlayerData represents player data for TRF generation
type PlayerData struct {
	StartNo      int
	Sex          string
	Title        string
	FirstName    string
	LastName     string
	Rating       int
	Federation   string
	FideID       string
	BirthDate    string
	Points       float64
	Rank         int
	RoundResults []RoundResultData
}

// RoundResultData represents a round result
type RoundResultData struct {
	Round      int
	OpponentNo int
	Color      string
	Result     string
}

// CreateFromTournament creates a TRF generator from tournament data
func CreateFromTournament(data *TournamentData) *Generator {
	g := NewGenerator()

	g.SetTournamentInfo(data.Name, data.City, data.Federation, data.Rounds)
	g.SetDates(data.StartDate, data.EndDate)
	g.SetTournamentType(data.TournamentType)
	g.SetTimeControl(data.TimeControl)
	g.SetArbiters(data.ChiefArbiter, nil)

	for _, p := range data.Players {
		player := TRFPlayer{
			StartNo:    p.StartNo,
			Sex:        p.Sex,
			Title:      p.Title,
			Name:       fmt.Sprintf("%s, %s", p.LastName, p.FirstName),
			Rating:     p.Rating,
			Federation: p.Federation,
			FideID:     p.FideID,
			BirthDate:  p.BirthDate,
			Points:     p.Points,
			Rank:       p.Rank,
		}

		for _, r := range p.RoundResults {
			player.RoundResults = append(player.RoundResults, TRFRoundResult{
				Round:      r.Round,
				OpponentNo: r.OpponentNo,
				Color:      r.Color,
				Result:     r.Result,
			})
		}

		g.AddPlayer(player)
	}

	return g
}
