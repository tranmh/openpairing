package standings

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/tournament/tiebreak"
)

// Service provides standings functionality
type Service struct {
	repo *Repository
}

// NewService creates a new standings service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetStandings retrieves current standings for a tournament
func (s *Service) GetStandings(ctx context.Context, tournamentID uuid.UUID) (*StandingsResponse, error) {
	// Get tournament info
	info, err := s.repo.GetTournamentInfo(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament info: %w", err)
	}

	// Get participants
	participants, err := s.repo.GetParticipants(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	// Get all games for tiebreak calculation
	games, err := s.repo.GetGames(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get games: %w", err)
	}

	// Calculate tiebreaks
	tiebreakCodes := make([]tiebreak.TiebreakCode, len(info.Tiebreaks))
	for i, tb := range info.Tiebreaks {
		tiebreakCodes[i] = tiebreak.TiebreakCode(tb)
	}

	calculator := tiebreak.NewCalculator(tiebreakCodes, info.TotalRounds)

	// Build player data for calculator
	participantPoints := make(map[uuid.UUID]float64)
	for _, p := range participants {
		participantPoints[p.ID] = p.Points
	}

	for _, p := range participants {
		playerData := &tiebreak.PlayerData{
			ID:           p.ID,
			StartRank:    p.StartRank,
			Rating:       p.Rating,
			Points:       p.Points,
			Wins:         p.Wins,
			Draws:        p.Draws,
			Losses:       p.Losses,
			GamesPlayed:  p.Wins + p.Draws + p.Losses,
			Results:      []tiebreak.PlayerResult{},
		}

		// Add game results
		for _, g := range games {
			var result *tiebreak.PlayerResult

			if g.WhiteParticipantID != nil && *g.WhiteParticipantID == p.ID {
				if g.BlackParticipantID != nil && g.WhiteScore != nil {
					result = &tiebreak.PlayerResult{
						OpponentID:  *g.BlackParticipantID,
						Score:       *g.WhiteScore,
						OpponentPts: participantPoints[*g.BlackParticipantID],
						PlayedWhite: true,
						RoundNumber: g.RoundNumber,
					}
				}
			} else if g.BlackParticipantID != nil && *g.BlackParticipantID == p.ID {
				if g.WhiteParticipantID != nil && g.BlackScore != nil {
					result = &tiebreak.PlayerResult{
						OpponentID:  *g.WhiteParticipantID,
						Score:       *g.BlackScore,
						OpponentPts: participantPoints[*g.WhiteParticipantID],
						PlayedWhite: false,
						RoundNumber: g.RoundNumber,
					}
					playerData.GamesAsBlack++
					if *g.BlackScore == 1.0 {
						playerData.WinsAsBlack++
					}
				}
			}

			if result != nil {
				playerData.Results = append(playerData.Results, *result)
			}
		}

		calculator.AddPlayer(playerData)
	}

	// Get calculated tiebreaks
	calculatedStandings := calculator.GetStandings()

	// Build response standings
	standings := make([]Standing, len(calculatedStandings))
	participantMap := make(map[uuid.UUID]ParticipantData)
	for _, p := range participants {
		participantMap[p.ID] = p
	}

	for i, cs := range calculatedStandings {
		p := participantMap[cs.PlayerID]

		tiebreakMap := make(map[string]float64)
		for code, value := range cs.Tiebreaks {
			tiebreakMap[string(code)] = value
		}

		// Calculate performance rating
		perfRating := s.calculatePerformanceRating(ctx, tournamentID, p.ID, p.Points, p.Wins+p.Draws+p.Losses)

		standings[i] = Standing{
			Rank:              cs.Rank,
			ParticipantID:     p.ID,
			PlayerID:          p.PlayerID,
			PlayerName:        p.PlayerName,
			PlayerFideID:      p.PlayerFideID,
			PlayerTitle:       p.PlayerTitle,
			ClubName:          p.ClubName,
			Rating:            p.Rating,
			Points:            p.Points,
			GamesPlayed:       p.Wins + p.Draws + p.Losses,
			Wins:              p.Wins,
			Draws:             p.Draws,
			Losses:            p.Losses,
			Tiebreaks:         tiebreakMap,
			IsActive:          p.IsActive,
			PerformanceRating: perfRating,
		}
	}

	return &StandingsResponse{
		TournamentID:   tournamentID,
		TournamentName: info.Name,
		CurrentRound:   info.CurrentRound,
		TotalRounds:    info.TotalRounds,
		LastUpdated:    time.Now(),
		Standings:      standings,
		TiebreakOrder:  info.Tiebreaks,
	}, nil
}

// GetCrosstable retrieves the crosstable for a tournament
func (s *Service) GetCrosstable(ctx context.Context, tournamentID uuid.UUID) (*CrosstableResponse, error) {
	// Get tournament info
	info, err := s.repo.GetTournamentInfo(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament info: %w", err)
	}

	// Get crosstable data
	rows, totalRounds, err := s.repo.GetCrosstableData(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get crosstable data: %w", err)
	}

	return &CrosstableResponse{
		TournamentID:   tournamentID,
		TournamentName: info.Name,
		CurrentRound:   info.CurrentRound,
		TotalRounds:    totalRounds,
		Rows:           rows,
	}, nil
}

// RecalculateStandings recalculates all tiebreaks and updates the database
func (s *Service) RecalculateStandings(ctx context.Context, tournamentID uuid.UUID) error {
	standings, err := s.GetStandings(ctx, tournamentID)
	if err != nil {
		return fmt.Errorf("failed to get standings: %w", err)
	}

	// Update each participant's tiebreaks in the database
	for _, standing := range standings.Standings {
		buchholz := standing.Tiebreaks["buchholz"]
		buchholzCut1 := standing.Tiebreaks["buchholz_cut1"]
		sonnebornBerger := standing.Tiebreaks["sonneborn_berger"]

		err := s.repo.UpdateParticipantTiebreaks(ctx, standing.ParticipantID, buchholz, buchholzCut1, sonnebornBerger)
		if err != nil {
			return fmt.Errorf("failed to update tiebreaks for participant %s: %w", standing.ParticipantID, err)
		}
	}

	return nil
}

// GetPlayerProgress retrieves a player's progress through the tournament
func (s *Service) GetPlayerProgress(ctx context.Context, tournamentID, participantID uuid.UUID) (*PlayerProgress, error) {
	// Get tournament info
	info, err := s.repo.GetTournamentInfo(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament info: %w", err)
	}

	// Get all games
	games, err := s.repo.GetGames(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get games: %w", err)
	}

	// Get participants for opponent names
	participants, err := s.repo.GetParticipants(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	participantMap := make(map[uuid.UUID]ParticipantData)
	for _, p := range participants {
		participantMap[p.ID] = p
	}

	// Find the participant
	participant, ok := participantMap[participantID]
	if !ok {
		return nil, fmt.Errorf("participant not found")
	}

	progress := &PlayerProgress{
		ParticipantID: participantID,
		PlayerName:    participant.PlayerName,
		RoundPoints:   make([]float64, info.TotalRounds),
		RoundRanks:    make([]int, info.TotalRounds),
	}

	// Calculate round-by-round points
	cumulativePoints := 0.0
	for round := 1; round <= info.TotalRounds; round++ {
		for _, g := range games {
			if g.RoundNumber == round {
				if g.WhiteParticipantID != nil && *g.WhiteParticipantID == participantID {
					if g.WhiteScore != nil {
						cumulativePoints += *g.WhiteScore
					}
					if g.BlackParticipantID != nil && g.Result != nil {
						opp := participantMap[*g.BlackParticipantID]
						progress.Opponents = append(progress.Opponents, struct {
							RoundNumber int     `json:"roundNumber"`
							Name        string  `json:"name"`
							Rating      int     `json:"rating"`
							Result      string  `json:"result"`
							Color       string  `json:"color"`
						}{
							RoundNumber: round,
							Name:        opp.PlayerName,
							Rating:      opp.Rating,
							Result:      *g.Result,
							Color:       "w",
						})
					}
				} else if g.BlackParticipantID != nil && *g.BlackParticipantID == participantID {
					if g.BlackScore != nil {
						cumulativePoints += *g.BlackScore
					}
					if g.WhiteParticipantID != nil && g.Result != nil {
						opp := participantMap[*g.WhiteParticipantID]
						progress.Opponents = append(progress.Opponents, struct {
							RoundNumber int     `json:"roundNumber"`
							Name        string  `json:"name"`
							Rating      int     `json:"rating"`
							Result      string  `json:"result"`
							Color       string  `json:"color"`
						}{
							RoundNumber: round,
							Name:        opp.PlayerName,
							Rating:      opp.Rating,
							Result:      *g.Result,
							Color:       "b",
						})
					}
				}
			}
		}
		progress.RoundPoints[round-1] = cumulativePoints
	}

	// Sort opponents by round
	sort.Slice(progress.Opponents, func(i, j int) bool {
		return progress.Opponents[i].RoundNumber < progress.Opponents[j].RoundNumber
	})

	return progress, nil
}

// GetAvailableTiebreaks returns information about all available tiebreak systems
func (s *Service) GetAvailableTiebreaks() []TiebreakInfo {
	tiebreaks := tiebreak.AvailableTiebreaks()
	result := make([]TiebreakInfo, len(tiebreaks))

	descriptions := map[tiebreak.TiebreakCode]string{
		tiebreak.TBBuchholz:        "Sum of all opponents' final points",
		tiebreak.TBBuchholzCut1:    "Buchholz minus the lowest opponent score",
		tiebreak.TBBuchholzCut2:    "Buchholz minus the two lowest opponent scores",
		tiebreak.TBBuchholzMedian:  "Buchholz minus highest and lowest opponent scores",
		tiebreak.TBSonnebornBerger: "Sum of defeated opponents' points + half of drawn opponents' points",
		tiebreak.TBDirectEncounter: "Result in games between tied players",
		tiebreak.TBWins:            "Total number of wins",
		tiebreak.TBGamesBlack:      "Total number of games played with black pieces",
		tiebreak.TBWinsBlack:       "Number of wins with black pieces",
		tiebreak.TBProgress:        "Sum of cumulative scores after each round",
		tiebreak.TBAROC:            "Average rating of opponents, excluding lowest",
		tiebreak.TBKoya:            "Points against opponents with 50%+ score",
		tiebreak.TBRating:          "Player's starting rating",
		tiebreak.TBRatingPerf:      "Performance rating based on results",
		tiebreak.TBAvgOppRating:    "Average rating of all opponents",
	}

	for i, tb := range tiebreaks {
		result[i] = TiebreakInfo{
			Code:        string(tb),
			Name:        tiebreak.TiebreakName(tb),
			Description: descriptions[tb],
		}
	}

	return result
}

// calculatePerformanceRating calculates a player's performance rating
func (s *Service) calculatePerformanceRating(ctx context.Context, tournamentID, participantID uuid.UUID, points float64, gamesPlayed int) *int {
	if gamesPlayed == 0 {
		return nil
	}

	ratings, err := s.repo.GetOpponentRatings(ctx, tournamentID, participantID)
	if err != nil || len(ratings) == 0 {
		return nil
	}

	// Calculate average opponent rating
	totalRating := 0
	for _, r := range ratings {
		totalRating += r
	}
	avgOpponentRating := float64(totalRating) / float64(len(ratings))

	// Calculate percentage score
	percentage := points / float64(gamesPlayed)

	// Performance rating formula: avgOpponentRating + (percentage - 0.5) * 800
	// This is a simplified version of the FIDE formula
	var perfRating int
	if percentage >= 1.0 {
		perfRating = int(avgOpponentRating) + 400
	} else if percentage <= 0.0 {
		perfRating = int(avgOpponentRating) - 400
	} else {
		// Use rating difference table approximation
		dp := getRatingDifferenceFromPercentage(percentage)
		perfRating = int(math.Round(avgOpponentRating + dp))
	}

	return &perfRating
}

// getRatingDifferenceFromPercentage approximates the FIDE dp table
func getRatingDifferenceFromPercentage(p float64) float64 {
	// This is a polynomial approximation of the FIDE expected score table
	// For p in [0, 1], returns the rating difference
	if p >= 1.0 {
		return 400
	}
	if p <= 0.0 {
		return -400
	}
	if p == 0.5 {
		return 0
	}

	// Use inverse of expected score formula: E = 1 / (1 + 10^(-D/400))
	// Solving for D: D = -400 * log10((1/E) - 1)
	if p > 0 && p < 1 {
		return -400 * math.Log10((1/p)-1)
	}
	return 0
}
