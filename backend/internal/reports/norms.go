package reports

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// NormType represents the type of FIDE title norm
type NormType string

const (
	NormGM  NormType = "GM"
	NormIM  NormType = "IM"
	NormWGM NormType = "WGM"
	NormWIM NormType = "WIM"
	NormFM  NormType = "FM"
	NormWFM NormType = "WFM"
	NormCM  NormType = "CM"
	NormWCM NormType = "WCM"
)

// NormRequirements contains FIDE norm requirements
type NormRequirements struct {
	NormType       NormType
	MinRating      int     // Minimum rating to achieve
	MinGames       int     // Minimum games required (usually 9)
	MinOpponents   int     // Minimum different opponents
	MinFederations int     // Minimum different federations (including own)
	MinTitledOpps  int     // Minimum titled opponents
	MinForeignOpps int     // Minimum opponents from other federations
	Performance    int     // Required performance rating
	TPR            float64 // Tournament Performance Rating required
}

// Standard FIDE norm requirements
var NormRequirementsTable = map[NormType]NormRequirements{
	NormGM: {
		NormType:       NormGM,
		MinRating:      2200,
		MinGames:       9,
		MinOpponents:   9,
		MinFederations: 3,
		MinTitledOpps:  3,
		MinForeignOpps: 3,
		Performance:    2600,
	},
	NormIM: {
		NormType:       NormIM,
		MinRating:      2000,
		MinGames:       9,
		MinOpponents:   9,
		MinFederations: 3,
		MinTitledOpps:  3,
		MinForeignOpps: 3,
		Performance:    2450,
	},
	NormWGM: {
		NormType:       NormWGM,
		MinRating:      2000,
		MinGames:       9,
		MinOpponents:   9,
		MinFederations: 3,
		MinTitledOpps:  3,
		MinForeignOpps: 3,
		Performance:    2400,
	},
	NormWIM: {
		NormType:       NormWIM,
		MinRating:      1800,
		MinGames:       9,
		MinOpponents:   9,
		MinFederations: 3,
		MinTitledOpps:  3,
		MinForeignOpps: 3,
		Performance:    2250,
	},
	NormFM: {
		NormType:       NormFM,
		MinRating:      1800,
		MinGames:       9,
		MinOpponents:   9,
		MinFederations: 2,
		MinTitledOpps:  0,
		MinForeignOpps: 0,
		Performance:    2300,
	},
	NormCM: {
		NormType:       NormCM,
		MinRating:      1600,
		MinGames:       9,
		MinOpponents:   9,
		MinFederations: 2,
		MinTitledOpps:  0,
		MinForeignOpps: 0,
		Performance:    2200,
	},
}

// NormResult contains the result of a norm calculation
type NormResult struct {
	PlayerID        uuid.UUID
	PlayerName      string
	PlayerFideID    string
	PlayerRating    int
	PlayerFederation string
	NormType        NormType
	Achieved        bool
	Performance     int
	GamesPlayed     int
	Score           float64
	OpponentCount   int
	FederationCount int
	TitledOppCount  int
	ForeignOppCount int
	AverageOppRating int
	Details         []NormGameDetail
	MissingReasons  []string
}

// NormGameDetail contains details of a single game for norm calculation
type NormGameDetail struct {
	Round           int
	OpponentName    string
	OpponentFideID  string
	OpponentRating  int
	OpponentTitle   string
	OpponentFed     string
	Color           string
	Result          float64
	RatingDiff      int
	ExpectedScore   float64
}

// NormCalculator calculates FIDE title norms
type NormCalculator struct {
	// Probability table for expected score calculation
	probabilityTable map[int]float64
}

// NewNormCalculator creates a new norm calculator
func NewNormCalculator() *NormCalculator {
	return &NormCalculator{
		probabilityTable: buildProbabilityTable(),
	}
}

// buildProbabilityTable builds the FIDE expected score table
func buildProbabilityTable() map[int]float64 {
	// FIDE probability table based on rating difference
	table := make(map[int]float64)

	// Rating difference -> expected score for higher rated player
	// Simplified FIDE table
	diffs := []struct {
		diff int
		prob float64
	}{
		{0, 0.50}, {10, 0.51}, {20, 0.53}, {30, 0.54}, {40, 0.55},
		{50, 0.57}, {60, 0.58}, {70, 0.59}, {80, 0.60}, {90, 0.61},
		{100, 0.64}, {110, 0.65}, {120, 0.66}, {130, 0.67}, {140, 0.68},
		{150, 0.69}, {160, 0.70}, {170, 0.71}, {180, 0.72}, {190, 0.73},
		{200, 0.76}, {210, 0.77}, {220, 0.78}, {230, 0.79}, {240, 0.80},
		{250, 0.81}, {260, 0.82}, {270, 0.83}, {280, 0.84}, {290, 0.85},
		{300, 0.85}, {310, 0.86}, {320, 0.87}, {330, 0.87}, {340, 0.88},
		{350, 0.89}, {360, 0.89}, {370, 0.90}, {380, 0.90}, {390, 0.91},
		{400, 0.92}, {500, 0.94}, {600, 0.96}, {700, 0.97}, {800, 0.99},
	}

	for _, d := range diffs {
		table[d.diff] = d.prob
		table[-d.diff] = 1 - d.prob
	}

	return table
}

// getExpectedScore returns expected score based on rating difference
func (c *NormCalculator) getExpectedScore(ratingDiff int) float64 {
	// Clamp to table bounds
	if ratingDiff > 800 {
		ratingDiff = 800
	} else if ratingDiff < -800 {
		ratingDiff = -800
	}

	// Find closest entry
	sign := 1
	if ratingDiff < 0 {
		sign = -1
		ratingDiff = -ratingDiff
	}

	// Round to nearest 10
	rounded := (ratingDiff / 10) * 10

	prob, ok := c.probabilityTable[rounded*sign]
	if ok {
		return prob
	}

	// Fallback calculation
	return 1.0 / (1.0 + math.Pow(10, float64(-ratingDiff)/400.0))
}

// CalculatePerformance calculates tournament performance rating
func (c *NormCalculator) CalculatePerformance(games []NormGameDetail) int {
	if len(games) == 0 {
		return 0
	}

	var totalRating int
	var totalScore float64

	for _, g := range games {
		totalRating += g.OpponentRating
		totalScore += g.Result
	}

	avgRating := float64(totalRating) / float64(len(games))
	percentage := totalScore / float64(len(games))

	// Performance = Average opponent rating + rating difference for percentage
	var ratingDiff float64
	if percentage >= 1.0 {
		ratingDiff = 800
	} else if percentage <= 0.0 {
		ratingDiff = -800
	} else {
		// Inverse of expected score formula
		ratingDiff = -400 * math.Log10(1/percentage - 1)
	}

	return int(math.Round(avgRating + ratingDiff))
}

// CheckNorm checks if a player achieved a specific norm
func (c *NormCalculator) CheckNorm(
	normType NormType,
	playerRating int,
	playerFederation string,
	games []NormGameDetail,
) *NormResult {
	req, ok := NormRequirementsTable[normType]
	if !ok {
		return nil
	}

	result := &NormResult{
		NormType:    normType,
		GamesPlayed: len(games),
		Details:     games,
	}

	// Calculate statistics
	var totalScore float64
	opponents := make(map[string]bool)
	federations := make(map[string]bool)
	titledCount := 0
	foreignCount := 0
	totalOppRating := 0

	federations[playerFederation] = true // Include own federation

	for _, g := range games {
		totalScore += g.Result
		opponents[g.OpponentFideID] = true
		totalOppRating += g.OpponentRating

		if g.OpponentFed != "" {
			federations[g.OpponentFed] = true
			if g.OpponentFed != playerFederation {
				foreignCount++
			}
		}

		if isTitled(g.OpponentTitle) {
			titledCount++
		}
	}

	result.Score = totalScore
	result.OpponentCount = len(opponents)
	result.FederationCount = len(federations)
	result.TitledOppCount = titledCount
	result.ForeignOppCount = foreignCount
	if len(games) > 0 {
		result.AverageOppRating = totalOppRating / len(games)
	}
	result.Performance = c.CalculatePerformance(games)

	// Check requirements
	var missing []string

	if playerRating < req.MinRating {
		missing = append(missing, fmt.Sprintf("Player rating %d below minimum %d", playerRating, req.MinRating))
	}
	if result.GamesPlayed < req.MinGames {
		missing = append(missing, fmt.Sprintf("Games played %d below minimum %d", result.GamesPlayed, req.MinGames))
	}
	if result.OpponentCount < req.MinOpponents {
		missing = append(missing, fmt.Sprintf("Opponent count %d below minimum %d", result.OpponentCount, req.MinOpponents))
	}
	if result.FederationCount < req.MinFederations {
		missing = append(missing, fmt.Sprintf("Federation count %d below minimum %d", result.FederationCount, req.MinFederations))
	}
	if result.TitledOppCount < req.MinTitledOpps {
		missing = append(missing, fmt.Sprintf("Titled opponents %d below minimum %d", result.TitledOppCount, req.MinTitledOpps))
	}
	if result.ForeignOppCount < req.MinForeignOpps {
		missing = append(missing, fmt.Sprintf("Foreign opponents %d below minimum %d", result.ForeignOppCount, req.MinForeignOpps))
	}
	if result.Performance < req.Performance {
		missing = append(missing, fmt.Sprintf("Performance %d below required %d", result.Performance, req.Performance))
	}

	result.MissingReasons = missing
	result.Achieved = len(missing) == 0

	return result
}

// isTitled checks if a title string represents a FIDE title
func isTitled(title string) bool {
	switch title {
	case "GM", "IM", "FM", "CM", "WGM", "WIM", "WFM", "WCM":
		return true
	default:
		return false
	}
}

// NormReport contains all norm results for a tournament
type NormReport struct {
	TournamentID   uuid.UUID
	TournamentName string
	GeneratedAt    time.Time
	NormResults    []NormResult
}

// NormReportService generates norm reports
type NormReportService struct {
	calculator *NormCalculator
	dataProvider TournamentDataProvider
}

// NewNormReportService creates a new norm report service
func NewNormReportService(provider TournamentDataProvider) *NormReportService {
	return &NormReportService{
		calculator: NewNormCalculator(),
		dataProvider: provider,
	}
}

// GenerateNormReport generates a complete norm report for a tournament
func (s *NormReportService) GenerateNormReport(ctx context.Context, tournamentID uuid.UUID) (*NormReport, error) {
	tournament, err := s.dataProvider.GetTournament(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament: %w", err)
	}

	participants, err := s.dataProvider.GetParticipants(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	games, err := s.dataProvider.GetGames(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get games: %w", err)
	}

	// Build participant lookup
	participantByID := make(map[uuid.UUID]*ParticipantData)
	for i := range participants {
		participantByID[participants[i].PlayerID] = &participants[i]
	}

	// Build games per player
	playerGames := make(map[uuid.UUID][]NormGameDetail)
	for _, game := range games {
		white := participantByID[game.WhiteID]
		black := participantByID[game.BlackID]

		if white != nil && black != nil {
			// Add game for white
			playerGames[game.WhiteID] = append(playerGames[game.WhiteID], NormGameDetail{
				Round:          game.Round,
				OpponentName:   black.LastName + ", " + black.FirstName,
				OpponentFideID: black.FideID,
				OpponentRating: black.FideRating,
				OpponentTitle:  "", // Would need to be added to ParticipantData
				OpponentFed:    "", // Would need to be added to ParticipantData
				Color:          "W",
				Result:         game.WhiteScore,
				RatingDiff:     white.FideRating - black.FideRating,
			})

			// Add game for black
			playerGames[game.BlackID] = append(playerGames[game.BlackID], NormGameDetail{
				Round:          game.Round,
				OpponentName:   white.LastName + ", " + white.FirstName,
				OpponentFideID: white.FideID,
				OpponentRating: white.FideRating,
				OpponentTitle:  "",
				OpponentFed:    "",
				Color:          "B",
				Result:         game.BlackScore,
				RatingDiff:     black.FideRating - white.FideRating,
			})
		}
	}

	report := &NormReport{
		TournamentID:   tournamentID,
		TournamentName: tournament.Name,
		GeneratedAt:    time.Now(),
	}

	// Check each player for norms
	normTypes := []NormType{NormGM, NormIM, NormWGM, NormWIM, NormFM, NormCM}

	for _, p := range participants {
		games := playerGames[p.PlayerID]
		if len(games) < 9 {
			continue // Not enough games for norm
		}

		for _, normType := range normTypes {
			result := s.calculator.CheckNorm(normType, p.FideRating, "", games)
			if result != nil {
				result.PlayerID = p.PlayerID
				result.PlayerName = p.LastName + ", " + p.FirstName
				result.PlayerFideID = p.FideID
				result.PlayerRating = p.FideRating

				// Only include if achieved or close to achieving
				if result.Achieved || (result.Performance >= NormRequirementsTable[normType].Performance-50) {
					report.NormResults = append(report.NormResults, *result)
				}
			}
		}
	}

	// Sort by norm type priority, then by performance
	sort.Slice(report.NormResults, func(i, j int) bool {
		if report.NormResults[i].Achieved != report.NormResults[j].Achieved {
			return report.NormResults[i].Achieved
		}
		return report.NormResults[i].Performance > report.NormResults[j].Performance
	})

	return report, nil
}
