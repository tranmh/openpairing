package tiebreak

import (
	"sort"

	"github.com/google/uuid"
)

// TiebreakCode represents a tiebreak system
type TiebreakCode string

const (
	// Primary tiebreaks
	TBBuchholz        TiebreakCode = "buchholz"
	TBBuchholzCut1    TiebreakCode = "buchholz_cut1"
	TBBuchholzCut2    TiebreakCode = "buchholz_cut2"
	TBBuchholzMedian  TiebreakCode = "buchholz_median"
	TBSonnebornBerger TiebreakCode = "sonneborn_berger"
	TBDirectEncounter TiebreakCode = "direct_encounter"
	TBWins            TiebreakCode = "wins"
	TBGamesBlack      TiebreakCode = "games_black"
	TBProgress        TiebreakCode = "progress"
	TBAROC            TiebreakCode = "aroc"
	TBKoya            TiebreakCode = "koya"

	// Additional tiebreaks
	TBWinsBlack       TiebreakCode = "wins_black"
	TBRating          TiebreakCode = "rating"
	TBRatingPerf      TiebreakCode = "rating_performance"
	TBAvgOppRating    TiebreakCode = "avg_opponent_rating"
)

// PlayerResult represents a player's result against an opponent
type PlayerResult struct {
	OpponentID   uuid.UUID
	Score        float64 // 1.0, 0.5, 0.0
	OpponentPts  float64 // Opponent's total points
	PlayedWhite  bool
	RoundNumber  int
}

// PlayerData contains all data needed for tiebreak calculation
type PlayerData struct {
	ID             uuid.UUID
	StartRank      int
	Rating         int
	Points         float64
	Results        []PlayerResult
	GamesPlayed    int
	Wins           int
	Draws          int
	Losses         int
	GamesAsBlack   int
	WinsAsBlack    int
	CumulativePoints []float64 // Points after each round for progress
}

// Calculator calculates tiebreaks for a tournament
type Calculator struct {
	players   map[uuid.UUID]*PlayerData
	tiebreaks []TiebreakCode
	rounds    int
}

// NewCalculator creates a new tiebreak calculator
func NewCalculator(tiebreaks []TiebreakCode, rounds int) *Calculator {
	return &Calculator{
		players:   make(map[uuid.UUID]*PlayerData),
		tiebreaks: tiebreaks,
		rounds:    rounds,
	}
}

// AddPlayer adds a player to the calculator
func (c *Calculator) AddPlayer(player *PlayerData) {
	c.players[player.ID] = player
}

// Calculate computes all tiebreaks for all players
func (c *Calculator) Calculate() map[uuid.UUID]map[TiebreakCode]float64 {
	results := make(map[uuid.UUID]map[TiebreakCode]float64)

	for id, player := range c.players {
		results[id] = make(map[TiebreakCode]float64)

		for _, tb := range c.tiebreaks {
			results[id][tb] = c.calculateTiebreak(player, tb)
		}
	}

	return results
}

// calculateTiebreak calculates a single tiebreak for a player
func (c *Calculator) calculateTiebreak(player *PlayerData, tb TiebreakCode) float64 {
	switch tb {
	case TBBuchholz:
		return c.buchholz(player, 0)
	case TBBuchholzCut1:
		return c.buchholz(player, 1)
	case TBBuchholzCut2:
		return c.buchholz(player, 2)
	case TBBuchholzMedian:
		return c.buchholzMedian(player)
	case TBSonnebornBerger:
		return c.sonnebornBerger(player)
	case TBDirectEncounter:
		return c.directEncounter(player)
	case TBWins:
		return float64(player.Wins)
	case TBGamesBlack:
		return float64(player.GamesAsBlack)
	case TBWinsBlack:
		return float64(player.WinsAsBlack)
	case TBProgress:
		return c.progress(player)
	case TBAROC:
		return c.aroc(player)
	case TBKoya:
		return c.koya(player)
	case TBRating:
		return float64(player.Rating)
	case TBAvgOppRating:
		return c.avgOpponentRating(player)
	default:
		return 0
	}
}

// Buchholz calculates the Buchholz score (sum of opponents' points)
// cutLowest specifies how many lowest opponent scores to exclude
func (c *Calculator) buchholz(player *PlayerData, cutLowest int) float64 {
	if len(player.Results) == 0 {
		return 0
	}

	var opponentPoints []float64
	for _, result := range player.Results {
		opponentPoints = append(opponentPoints, result.OpponentPts)
	}

	// Handle unplayed rounds (virtual opponents)
	for i := len(opponentPoints); i < c.rounds; i++ {
		// Virtual opponent gets points based on player's result
		// If player got a bye, virtual opponent has average points
		avgPoints := player.Points / float64(len(player.Results))
		opponentPoints = append(opponentPoints, avgPoints*float64(c.rounds)/float64(len(player.Results)))
	}

	// Sort to find lowest scores
	sort.Float64s(opponentPoints)

	// Cut lowest scores
	if cutLowest >= len(opponentPoints) {
		cutLowest = len(opponentPoints) - 1
	}
	if cutLowest > 0 {
		opponentPoints = opponentPoints[cutLowest:]
	}

	// Sum remaining
	var sum float64
	for _, pts := range opponentPoints {
		sum += pts
	}

	return sum
}

// buchholzMedian calculates median Buchholz (cut highest and lowest)
func (c *Calculator) buchholzMedian(player *PlayerData) float64 {
	if len(player.Results) < 3 {
		return c.buchholz(player, 0)
	}

	var opponentPoints []float64
	for _, result := range player.Results {
		opponentPoints = append(opponentPoints, result.OpponentPts)
	}

	sort.Float64s(opponentPoints)

	// Remove highest and lowest
	if len(opponentPoints) > 2 {
		opponentPoints = opponentPoints[1 : len(opponentPoints)-1]
	}

	var sum float64
	for _, pts := range opponentPoints {
		sum += pts
	}

	return sum
}

// sonnebornBerger calculates the Sonneborn-Berger score
// Sum of: opponents' points for wins + half opponents' points for draws
func (c *Calculator) sonnebornBerger(player *PlayerData) float64 {
	var sum float64

	for _, result := range player.Results {
		if result.Score == 1.0 {
			sum += result.OpponentPts
		} else if result.Score == 0.5 {
			sum += result.OpponentPts / 2
		}
		// Losses contribute 0
	}

	return sum
}

// directEncounter returns the score in head-to-head games among tied players
// Note: This requires knowing which players are tied, so it returns
// the player's total score for now (proper implementation needs tied group)
func (c *Calculator) directEncounter(player *PlayerData) float64 {
	// This is a simplified version - proper implementation requires
	// knowing which players are tied at this point
	return player.Points
}

// progress calculates the progressive/cumulative score
// Sum of points after each round
func (c *Calculator) progress(player *PlayerData) float64 {
	if len(player.CumulativePoints) == 0 {
		// Calculate from results if not provided
		var cumulative, sum float64
		for _, result := range player.Results {
			sum += result.Score
			cumulative += sum
		}
		return cumulative
	}

	var sum float64
	for _, pts := range player.CumulativePoints {
		sum += pts
	}
	return sum
}

// aroc calculates Average Rating of Opponents Cut (lowest removed)
func (c *Calculator) aroc(player *PlayerData) float64 {
	if len(player.Results) == 0 {
		return 0
	}

	var ratings []int
	for _, result := range player.Results {
		if opp, ok := c.players[result.OpponentID]; ok {
			ratings = append(ratings, opp.Rating)
		}
	}

	if len(ratings) == 0 {
		return 0
	}

	// Sort and remove lowest
	sort.Ints(ratings)
	if len(ratings) > 1 {
		ratings = ratings[1:]
	}

	var sum int
	for _, r := range ratings {
		sum += r
	}

	return float64(sum) / float64(len(ratings))
}

// koya calculates the Koya system score
// Points scored against opponents with >= 50% score
func (c *Calculator) koya(player *PlayerData) float64 {
	halfPoints := float64(c.rounds) / 2

	var sum float64
	for _, result := range player.Results {
		if result.OpponentPts >= halfPoints {
			sum += result.Score
		}
	}

	return sum
}

// avgOpponentRating calculates the average rating of all opponents
func (c *Calculator) avgOpponentRating(player *PlayerData) float64 {
	if len(player.Results) == 0 {
		return 0
	}

	var totalRating int
	var count int
	for _, result := range player.Results {
		if opp, ok := c.players[result.OpponentID]; ok {
			totalRating += opp.Rating
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return float64(totalRating) / float64(count)
}

// Standings represents sorted player standings
type Standing struct {
	PlayerID   uuid.UUID
	Rank       int
	Points     float64
	Tiebreaks  map[TiebreakCode]float64
}

// GetStandings returns players sorted by points and tiebreaks
func (c *Calculator) GetStandings() []Standing {
	tiebreakResults := c.Calculate()

	standings := make([]Standing, 0, len(c.players))
	for id, player := range c.players {
		standings = append(standings, Standing{
			PlayerID:  id,
			Points:    player.Points,
			Tiebreaks: tiebreakResults[id],
		})
	}

	// Sort by points, then by tiebreaks
	sort.Slice(standings, func(i, j int) bool {
		if standings[i].Points != standings[j].Points {
			return standings[i].Points > standings[j].Points
		}

		// Apply tiebreaks in order
		for _, tb := range c.tiebreaks {
			vi := standings[i].Tiebreaks[tb]
			vj := standings[j].Tiebreaks[tb]
			if vi != vj {
				return vi > vj
			}
		}

		// All equal, maintain original order
		return false
	})

	// Assign ranks
	for i := range standings {
		standings[i].Rank = i + 1
	}

	return standings
}

// AvailableTiebreaks returns all available tiebreak codes
func AvailableTiebreaks() []TiebreakCode {
	return []TiebreakCode{
		TBBuchholz,
		TBBuchholzCut1,
		TBBuchholzCut2,
		TBBuchholzMedian,
		TBSonnebornBerger,
		TBDirectEncounter,
		TBWins,
		TBGamesBlack,
		TBWinsBlack,
		TBProgress,
		TBAROC,
		TBKoya,
		TBRating,
		TBRatingPerf,
		TBAvgOppRating,
	}
}

// TiebreakName returns the display name for a tiebreak
func TiebreakName(code TiebreakCode) string {
	names := map[TiebreakCode]string{
		TBBuchholz:        "Buchholz",
		TBBuchholzCut1:    "Buchholz Cut 1",
		TBBuchholzCut2:    "Buchholz Cut 2",
		TBBuchholzMedian:  "Buchholz Median",
		TBSonnebornBerger: "Sonneborn-Berger",
		TBDirectEncounter: "Direkter Vergleich",
		TBWins:            "Anzahl Siege",
		TBGamesBlack:      "Partien mit Schwarz",
		TBWinsBlack:       "Siege mit Schwarz",
		TBProgress:        "Progressionswertung",
		TBAROC:            "AROC",
		TBKoya:            "Koya-System",
		TBRating:          "Wertungszahl",
		TBRatingPerf:      "Leistungswertung",
		TBAvgOppRating:    "Durchschnitt Gegnerwertung",
	}

	if name, ok := names[code]; ok {
		return name
	}
	return string(code)
}
