package standings

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/tranmh/openpairing/internal/tournament/tiebreak"
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

	// Get special rankings and eligibility data for badges
	specialRankings, _ := s.repo.GetSpecialRankings(ctx, tournamentID)
	eligibilityData, _ := s.repo.GetParticipantEligibilityData(ctx, tournamentID)

	// Calculate tiebreaks
	tiebreakCodes := make([]tiebreak.TiebreakCode, len(info.Tiebreaks))
	for i, tb := range info.Tiebreaks {
		tiebreakCodes[i] = tiebreak.TiebreakCode(tb)
	}

	// Create calculator with scoring settings
	scoringConfig := &tiebreak.ScoringConfig{
		WinPoints:           info.Scoring.WinPoints,
		DrawPoints:          info.Scoring.DrawPoints,
		LossPoints:          info.Scoring.LossPoints,
		SonnebornBergerMode: info.Scoring.SonnebornBergerMode,
	}
	calculator := tiebreak.NewCalculatorWithScoring(tiebreakCodes, info.TotalRounds, scoringConfig)

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
					// Check for win using scoring settings
					if *g.BlackScore == info.Scoring.WinPoints {
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

	// Pre-calculate rank-in-group for each special ranking
	// Maps: rankingID -> participantID -> rankInGroup
	rankInGroupMap := s.calculateRankInGroups(calculatedStandings, specialRankings, eligibilityData)

	for i, cs := range calculatedStandings {
		p := participantMap[cs.PlayerID]

		tiebreakMap := make(map[string]float64)
		for code, value := range cs.Tiebreaks {
			tiebreakMap[string(code)] = value
		}

		// Calculate performance rating
		perfRating := s.calculatePerformanceRating(ctx, tournamentID, p.ID, p.Points, p.Wins+p.Draws+p.Losses)

		// Calculate eligible rankings with rank-in-group
		var eligibleRankings []EligibleRanking
		if data, ok := eligibilityData[p.ID]; ok {
			for _, sr := range specialRankings {
				if isEligibleForRanking(data, sr) {
					rankInGroup := 0
					if ranks, ok := rankInGroupMap[sr.ID]; ok {
						if r, ok := ranks[p.ID]; ok {
							rankInGroup = r
						}
					}
					eligibleRankings = append(eligibleRankings, EligibleRanking{
						ID:          sr.ID,
						Name:        sr.Name,
						DisplayName: sr.DisplayName,
						RankInGroup: rankInGroup,
					})
				}
			}
		}

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
			EligibleRankings:  eligibleRankings,
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

// calculateRankInGroups calculates the rank within each special ranking group
func (s *Service) calculateRankInGroups(
	standings []tiebreak.Standing,
	rankings []SpecialRankingData,
	eligibility map[uuid.UUID]ParticipantEligibilityData,
) map[uuid.UUID]map[uuid.UUID]int {
	result := make(map[uuid.UUID]map[uuid.UUID]int)

	for _, sr := range rankings {
		result[sr.ID] = make(map[uuid.UUID]int)
		rank := 0
		for _, standing := range standings {
			if data, ok := eligibility[standing.PlayerID]; ok {
				if isEligibleForRanking(data, sr) {
					rank++
					result[sr.ID][standing.PlayerID] = rank
				}
			}
		}
	}

	return result
}

// isEligibleForRanking checks if a participant is eligible for a special ranking
func isEligibleForRanking(data ParticipantEligibilityData, sr SpecialRankingData) bool {
	if sr.MinBirthYear != nil && data.BirthYear < *sr.MinBirthYear {
		return false
	}
	if sr.MaxBirthYear != nil && data.BirthYear > *sr.MaxBirthYear {
		return false
	}
	if sr.MinRating != nil && data.Rating < *sr.MinRating {
		return false
	}
	if sr.MaxRating != nil && data.Rating > *sr.MaxRating {
		return false
	}
	if sr.Gender != nil && *sr.Gender != "" && data.Gender != *sr.Gender {
		return false
	}
	return true
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

// RankingCriteria contains filter criteria for special rankings
type RankingCriteria struct {
	MinBirthYear *int
	MaxBirthYear *int
	MinRating    *int
	MaxRating    *int
	Gender       *string
}

// IsEligible checks if a player matches the criteria
func (c *RankingCriteria) IsEligible(birthYear int, rating int, gender string) bool {
	if c.MinBirthYear != nil && birthYear < *c.MinBirthYear {
		return false
	}
	if c.MaxBirthYear != nil && birthYear > *c.MaxBirthYear {
		return false
	}
	if c.MinRating != nil && rating < *c.MinRating {
		return false
	}
	if c.MaxRating != nil && rating > *c.MaxRating {
		return false
	}
	if c.Gender != nil && *c.Gender != "" && gender != *c.Gender {
		return false
	}
	return true
}

// GetFilteredStandings retrieves standings filtered by special ranking criteria
func (s *Service) GetFilteredStandings(ctx context.Context, tournamentID uuid.UUID, criteria *RankingCriteria) (*StandingsResponse, error) {
	// Get tournament info
	info, err := s.repo.GetTournamentInfo(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament info: %w", err)
	}

	// Get all participants
	participants, err := s.repo.GetParticipants(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	// Get eligibility data
	eligibilityData, err := s.repo.GetParticipantEligibilityData(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get eligibility data: %w", err)
	}

	// Filter participants by criteria
	var filteredParticipants []ParticipantData
	for _, p := range participants {
		data, ok := eligibilityData[p.ID]
		if !ok {
			continue
		}
		if criteria.IsEligible(data.BirthYear, data.Rating, data.Gender) {
			filteredParticipants = append(filteredParticipants, p)
		}
	}

	if len(filteredParticipants) == 0 {
		return &StandingsResponse{
			TournamentID:   tournamentID,
			TournamentName: info.Name,
			CurrentRound:   info.CurrentRound,
			TotalRounds:    info.TotalRounds,
			LastUpdated:    time.Now(),
			Standings:      []Standing{},
			TiebreakOrder:  info.Tiebreaks,
		}, nil
	}

	// Get all games for tiebreak calculation
	games, err := s.repo.GetGames(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get games: %w", err)
	}

	// Calculate tiebreaks for filtered participants
	tiebreakCodes := make([]tiebreak.TiebreakCode, len(info.Tiebreaks))
	for i, tb := range info.Tiebreaks {
		tiebreakCodes[i] = tiebreak.TiebreakCode(tb)
	}

	scoringConfig := &tiebreak.ScoringConfig{
		WinPoints:           info.Scoring.WinPoints,
		DrawPoints:          info.Scoring.DrawPoints,
		LossPoints:          info.Scoring.LossPoints,
		SonnebornBergerMode: info.Scoring.SonnebornBergerMode,
	}
	calculator := tiebreak.NewCalculatorWithScoring(tiebreakCodes, info.TotalRounds, scoringConfig)

	// Build player data - need points for ALL participants for opponent points calculation
	allParticipantPoints := make(map[uuid.UUID]float64)
	for _, p := range participants {
		allParticipantPoints[p.ID] = p.Points
	}

	// Build filtered participant set for ranking
	filteredIDs := make(map[uuid.UUID]bool)
	for _, p := range filteredParticipants {
		filteredIDs[p.ID] = true
	}

	for _, p := range filteredParticipants {
		playerData := &tiebreak.PlayerData{
			ID:          p.ID,
			StartRank:   p.StartRank,
			Rating:      p.Rating,
			Points:      p.Points,
			Wins:        p.Wins,
			Draws:       p.Draws,
			Losses:      p.Losses,
			GamesPlayed: p.Wins + p.Draws + p.Losses,
			Results:     []tiebreak.PlayerResult{},
		}

		// Add game results
		for _, g := range games {
			var result *tiebreak.PlayerResult

			if g.WhiteParticipantID != nil && *g.WhiteParticipantID == p.ID {
				if g.BlackParticipantID != nil && g.WhiteScore != nil {
					result = &tiebreak.PlayerResult{
						OpponentID:  *g.BlackParticipantID,
						Score:       *g.WhiteScore,
						OpponentPts: allParticipantPoints[*g.BlackParticipantID],
						PlayedWhite: true,
						RoundNumber: g.RoundNumber,
					}
				}
			} else if g.BlackParticipantID != nil && *g.BlackParticipantID == p.ID {
				if g.WhiteParticipantID != nil && g.BlackScore != nil {
					result = &tiebreak.PlayerResult{
						OpponentID:  *g.WhiteParticipantID,
						Score:       *g.BlackScore,
						OpponentPts: allParticipantPoints[*g.WhiteParticipantID],
						PlayedWhite: false,
						RoundNumber: g.RoundNumber,
					}
					playerData.GamesAsBlack++
					if *g.BlackScore == info.Scoring.WinPoints {
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

	// Get calculated standings
	calculatedStandings := calculator.GetStandings()

	// Build response
	participantMap := make(map[uuid.UUID]ParticipantData)
	for _, p := range filteredParticipants {
		participantMap[p.ID] = p
	}

	standings := make([]Standing, len(calculatedStandings))
	for i, cs := range calculatedStandings {
		p := participantMap[cs.PlayerID]

		tiebreakMap := make(map[string]float64)
		for code, value := range cs.Tiebreaks {
			tiebreakMap[string(code)] = value
		}

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
		tiebreak.TBGamesPlayed:     "Total number of games played",
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
