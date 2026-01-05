package team_tournament

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/openpairing/openpairing/internal/team"
	"github.com/openpairing/openpairing/internal/tournament"
	"github.com/rs/zerolog"
)

// Service provides team tournament business logic
type Service struct {
	repo           *Repository
	tournamentRepo tournament.Repository
	teamRepo       *team.Repository
	logger         zerolog.Logger
}

// NewService creates a new team tournament service
func NewService(repo *Repository, tournamentRepo tournament.Repository, teamRepo *team.Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:           repo,
		tournamentRepo: tournamentRepo,
		teamRepo:       teamRepo,
		logger:         logger.With().Str("service", "team_tournament").Logger(),
	}
}

// GetSettings retrieves team tournament settings, creating defaults if needed
func (s *Service) GetSettings(ctx context.Context, tournamentID uuid.UUID) (*TeamTournamentSettings, error) {
	settings, err := s.repo.GetSettings(ctx, tournamentID)
	if err != nil {
		// Return default settings
		defaults := DefaultTeamTournamentSettings()
		defaults.TournamentID = tournamentID
		return defaults, nil
	}
	return settings, nil
}

// UpdateSettings updates team tournament settings
func (s *Service) UpdateSettings(ctx context.Context, settings *TeamTournamentSettings) error {
	return s.repo.CreateSettings(ctx, settings)
}

// RegisterTeam registers a team for a tournament
func (s *Service) RegisterTeam(ctx context.Context, req *RegisterTeamRequest, userID uuid.UUID) (*TeamParticipant, error) {
	// Verify tournament exists and is a team tournament
	tourney, err := s.tournamentRepo.GetByID(ctx, req.TournamentID)
	if err != nil {
		return nil, fmt.Errorf("tournament not found: %w", err)
	}

	if tourney.Type != "team_swiss" && tourney.Type != "team_round_robin" {
		return nil, fmt.Errorf("tournament is not a team tournament")
	}

	// Verify team exists
	teamData, err := s.teamRepo.GetTeamByID(ctx, req.TeamID)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}
	if teamData == nil {
		return nil, fmt.Errorf("team not found")
	}

	// Calculate seed rating from team roster
	members, err := s.teamRepo.GetTeamMembers(ctx, req.TeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	seedRating := s.calculateTeamSeedRating(members)

	participant := &TeamParticipant{
		ID:                 uuid.New(),
		TournamentID:       req.TournamentID,
		TeamID:             req.TeamID,
		RegistrationStatus: "registered",
		RegisteredAt:       time.Now(),
		StartRank:          0,
		SeedRating:         seedRating,
		MatchPoints:        0,
		BoardPoints:        0,
		Buchholz:           0,
		SonnebornBerger:    0,
		LineupLocked:       false,
		TeamName:           teamData.Name,
		TeamShortName:      teamData.ShortName,
	}

	if err := s.repo.RegisterTeam(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to register team: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", req.TournamentID.String()).
		Str("team_id", req.TeamID.String()).
		Msg("Team registered for tournament")

	return participant, nil
}

func (s *Service) calculateTeamSeedRating(members []team.TeamMemberWithPlayer) int {
	if len(members) == 0 {
		return 0
	}

	// Use average rating of top 8 board players
	boardMembers := make([]team.TeamMemberWithPlayer, 0)
	for _, m := range members {
		if !m.IsReserve {
			boardMembers = append(boardMembers, m)
		}
	}

	if len(boardMembers) == 0 {
		boardMembers = members
	}

	// Sort by board number (ascending) - handle nil board numbers
	sort.Slice(boardMembers, func(i, j int) bool {
		if boardMembers[i].BoardNumber == nil && boardMembers[j].BoardNumber == nil {
			return false
		}
		if boardMembers[i].BoardNumber == nil {
			return false
		}
		if boardMembers[j].BoardNumber == nil {
			return true
		}
		return *boardMembers[i].BoardNumber < *boardMembers[j].BoardNumber
	})

	// Take top 8
	count := 8
	if len(boardMembers) < count {
		count = len(boardMembers)
	}

	totalRating := 0
	for i := 0; i < count; i++ {
		totalRating += boardMembers[i].PlayerRating
	}

	if count == 0 {
		return 0
	}
	return totalRating / count
}

// ListTeamParticipants lists all teams in a tournament
func (s *Service) ListTeamParticipants(ctx context.Context, tournamentID uuid.UUID) ([]TeamParticipant, error) {
	return s.repo.ListTeamParticipants(ctx, tournamentID)
}

// WithdrawTeam withdraws a team from a tournament
func (s *Service) WithdrawTeam(ctx context.Context, participantID uuid.UUID, userID uuid.UUID) error {
	participant, err := s.repo.GetTeamParticipant(ctx, participantID)
	if err != nil {
		return err
	}

	if participant.RegistrationStatus == "withdrawn" {
		return fmt.Errorf("team already withdrawn")
	}

	if err := s.repo.WithdrawTeam(ctx, participantID); err != nil {
		return fmt.Errorf("failed to withdraw team: %w", err)
	}

	s.logger.Info().
		Str("participant_id", participantID.String()).
		Str("user_id", userID.String()).
		Msg("Team withdrawn from tournament")

	return nil
}

// SubmitLineup submits or updates a team lineup for a round
func (s *Service) SubmitLineup(ctx context.Context, req *SubmitLineupRequest, userID uuid.UUID) (*TeamLineup, error) {
	// Get tournament settings
	settings, err := s.GetSettings(ctx, req.TournamentID)
	if err != nil {
		return nil, err
	}

	// Get round
	round, err := s.tournamentRepo.GetRoundByNumber(ctx, req.TournamentID, req.RoundNumber)
	if err != nil {
		return nil, fmt.Errorf("round not found: %w", err)
	}

	// Check if lineup already locked
	existingLineup, _ := s.repo.GetLineup(ctx, round.ID, req.TeamParticipantID)
	if existingLineup != nil && existingLineup.IsLocked {
		return nil, fmt.Errorf("lineup is already locked")
	}

	// Validate lineup
	validationErrors := s.validateLineup(ctx, req.Lineup, settings, req.TeamParticipantID)

	now := time.Now()
	lineup := &TeamLineup{
		ID:                uuid.New(),
		TournamentID:      req.TournamentID,
		RoundID:           round.ID,
		TeamParticipantID: req.TeamParticipantID,
		Lineup:            req.Lineup,
		IsValid:           len(validationErrors) == 0,
		ValidationErrors:  validationErrors,
		SubmittedAt:       &now,
		SubmittedBy:       &userID,
		IsLocked:          false,
	}

	if existingLineup != nil {
		lineup.ID = existingLineup.ID
	}

	if err := s.repo.SaveLineup(ctx, lineup); err != nil {
		return nil, fmt.Errorf("failed to save lineup: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", req.TournamentID.String()).
		Int("round", req.RoundNumber).
		Str("team", req.TeamParticipantID.String()).
		Bool("valid", lineup.IsValid).
		Msg("Lineup submitted")

	return lineup, nil
}

func (s *Service) validateLineup(ctx context.Context, lineup []LineupEntry, settings *TeamTournamentSettings, teamParticipantID uuid.UUID) []string {
	var errors []string

	// Check board count
	if len(lineup) != settings.BoardsPerMatch {
		errors = append(errors, fmt.Sprintf("lineup must have exactly %d boards, got %d", settings.BoardsPerMatch, len(lineup)))
	}

	// Check for duplicate players
	playerIDs := make(map[uuid.UUID]bool)
	for _, entry := range lineup {
		if playerIDs[entry.PlayerID] {
			errors = append(errors, fmt.Sprintf("player %s appears multiple times", entry.PlayerID))
		}
		playerIDs[entry.PlayerID] = true
	}

	// Check rating order if enforced
	if settings.EnforceRatingOrder && len(lineup) > 1 {
		for i := 1; i < len(lineup); i++ {
			if lineup[i].Rating > lineup[i-1].Rating+settings.RatingOrderTolerance {
				errors = append(errors, fmt.Sprintf("board %d has higher rating than board %d (rating order violation)", lineup[i].Board, lineup[i-1].Board))
			}
		}
	}

	return errors
}

// GetLineup retrieves a team's lineup for a round
func (s *Service) GetLineup(ctx context.Context, tournamentID uuid.UUID, roundNumber int, teamParticipantID uuid.UUID) (*TeamLineup, error) {
	round, err := s.tournamentRepo.GetRoundByNumber(ctx, tournamentID, roundNumber)
	if err != nil {
		return nil, fmt.Errorf("round not found: %w", err)
	}

	return s.repo.GetLineup(ctx, round.ID, teamParticipantID)
}

// LockLineup locks a team's lineup (typically when deadline passes or arbiter action)
func (s *Service) LockLineup(ctx context.Context, tournamentID uuid.UUID, roundNumber int, teamParticipantID uuid.UUID, userID uuid.UUID) error {
	round, err := s.tournamentRepo.GetRoundByNumber(ctx, tournamentID, roundNumber)
	if err != nil {
		return fmt.Errorf("round not found: %w", err)
	}

	if err := s.repo.LockLineup(ctx, round.ID, teamParticipantID); err != nil {
		return fmt.Errorf("failed to lock lineup: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Int("round", roundNumber).
		Str("team", teamParticipantID.String()).
		Msg("Lineup locked")

	return nil
}

// GetTeamMatch retrieves a team match with boards
func (s *Service) GetTeamMatch(ctx context.Context, matchID uuid.UUID) (*TeamMatchResponse, error) {
	match, err := s.repo.GetTeamMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}

	boards, err := s.repo.GetTeamMatchBoards(ctx, matchID)
	if err != nil {
		return nil, err
	}

	return &TeamMatchResponse{
		Match:  *match,
		Boards: boards,
	}, nil
}

// ListTeamMatches lists all matches for a tournament
func (s *Service) ListTeamMatches(ctx context.Context, tournamentID uuid.UUID) ([]TeamMatch, error) {
	return s.repo.ListTeamMatches(ctx, tournamentID)
}

// ListTeamMatchesByRound lists matches for a specific round
func (s *Service) ListTeamMatchesByRound(ctx context.Context, tournamentID uuid.UUID, roundNumber int) ([]TeamMatch, error) {
	round, err := s.tournamentRepo.GetRoundByNumber(ctx, tournamentID, roundNumber)
	if err != nil {
		return nil, fmt.Errorf("round not found: %w", err)
	}

	return s.repo.ListTeamMatchesByRound(ctx, round.ID)
}

// EnterBoardResult enters or updates a board result
func (s *Service) EnterBoardResult(ctx context.Context, req *EnterBoardResultRequest, userID uuid.UUID) error {
	board, err := s.repo.GetBoardByMatchAndNumber(ctx, req.TeamMatchID, req.BoardNumber)
	if err != nil {
		return err
	}

	// Parse result
	homeScore, awayScore, err := parseResult(req.Result)
	if err != nil {
		return err
	}

	board.HomeScore = &homeScore
	board.AwayScore = &awayScore
	board.Result = &req.Result

	if err := s.repo.UpdateTeamMatchBoard(ctx, board); err != nil {
		return fmt.Errorf("failed to update board: %w", err)
	}

	// Recalculate match result
	if err := s.recalculateMatchResult(ctx, req.TeamMatchID); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to recalculate match result")
	}

	s.logger.Info().
		Str("match_id", req.TeamMatchID.String()).
		Int("board", req.BoardNumber).
		Str("result", req.Result).
		Msg("Board result entered")

	return nil
}

func parseResult(result string) (float64, float64, error) {
	switch result {
	case "1-0", "1-0F":
		return 1, 0, nil
	case "0-1", "0-1F":
		return 0, 1, nil
	case "1/2-1/2":
		return 0.5, 0.5, nil
	case "0-0F":
		return 0, 0, nil
	default:
		return 0, 0, fmt.Errorf("invalid result: %s", result)
	}
}

func (s *Service) recalculateMatchResult(ctx context.Context, matchID uuid.UUID) error {
	match, err := s.repo.GetTeamMatch(ctx, matchID)
	if err != nil {
		return err
	}

	boards, err := s.repo.GetTeamMatchBoards(ctx, matchID)
	if err != nil {
		return err
	}

	settings, err := s.GetSettings(ctx, match.TournamentID)
	if err != nil {
		return err
	}

	// Calculate board points
	var homeBoardPoints, awayBoardPoints float64
	allCompleted := true

	for _, board := range boards {
		if board.HomeScore != nil && board.AwayScore != nil {
			homeBoardPoints += *board.HomeScore
			awayBoardPoints += *board.AwayScore
		} else {
			allCompleted = false
		}
	}

	match.HomeBoardPoints = &homeBoardPoints
	match.AwayBoardPoints = &awayBoardPoints

	// Calculate match points if all boards completed
	if allCompleted {
		var homeMatchPoints, awayMatchPoints float64

		if homeBoardPoints > awayBoardPoints {
			homeMatchPoints = settings.MatchWinPoints
			awayMatchPoints = settings.MatchLossPoints
		} else if homeBoardPoints < awayBoardPoints {
			homeMatchPoints = settings.MatchLossPoints
			awayMatchPoints = settings.MatchWinPoints
		} else {
			homeMatchPoints = settings.MatchDrawPoints
			awayMatchPoints = settings.MatchDrawPoints
		}

		match.HomeMatchPoints = &homeMatchPoints
		match.AwayMatchPoints = &awayMatchPoints
		match.Status = MatchStatusCompleted

		// Update team standings
		s.updateTeamStandings(ctx, match)
	} else {
		match.Status = MatchStatusOngoing
	}

	return s.repo.UpdateTeamMatch(ctx, match)
}

func (s *Service) updateTeamStandings(ctx context.Context, match *TeamMatch) {
	if match.HomeTeamID != nil && match.HomeMatchPoints != nil {
		home, err := s.repo.GetTeamParticipant(ctx, *match.HomeTeamID)
		if err == nil {
			home.MatchPoints += *match.HomeMatchPoints
			if match.HomeBoardPoints != nil {
				home.BoardPoints += *match.HomeBoardPoints
			}
			s.repo.UpdateTeamParticipant(ctx, home)
		}
	}

	if match.AwayTeamID != nil && match.AwayMatchPoints != nil {
		away, err := s.repo.GetTeamParticipant(ctx, *match.AwayTeamID)
		if err == nil {
			away.MatchPoints += *match.AwayMatchPoints
			if match.AwayBoardPoints != nil {
				away.BoardPoints += *match.AwayBoardPoints
			}
			s.repo.UpdateTeamParticipant(ctx, away)
		}
	}
}

// GetTeamStandings retrieves team standings
func (s *Service) GetTeamStandings(ctx context.Context, tournamentID uuid.UUID) (*TeamTournamentStandingsResponse, error) {
	participants, err := s.repo.ListTeamParticipants(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	settings, err := s.GetSettings(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	matches, err := s.repo.ListTeamMatches(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Build standings
	standings := make([]TeamStanding, 0, len(participants))

	for _, p := range participants {
		if p.RegistrationStatus == "withdrawn" {
			continue
		}

		standing := TeamStanding{
			TeamParticipant: p,
		}

		// Count match results
		for _, m := range matches {
			if m.Status != MatchStatusCompleted {
				continue
			}

			if m.HomeTeamID != nil && *m.HomeTeamID == p.ID {
				standing.MatchesPlayed++
				if m.HomeMatchPoints != nil {
					if *m.HomeMatchPoints == settings.MatchWinPoints {
						standing.MatchWins++
					} else if *m.HomeMatchPoints == settings.MatchDrawPoints {
						standing.MatchDraws++
					} else {
						standing.MatchLosses++
					}
				}
			} else if m.AwayTeamID != nil && *m.AwayTeamID == p.ID {
				standing.MatchesPlayed++
				if m.AwayMatchPoints != nil {
					if *m.AwayMatchPoints == settings.MatchWinPoints {
						standing.MatchWins++
					} else if *m.AwayMatchPoints == settings.MatchDrawPoints {
						standing.MatchDraws++
					} else {
						standing.MatchLosses++
					}
				}
			}
		}

		standings = append(standings, standing)
	}

	// Sort by match points, then board points
	sort.Slice(standings, func(i, j int) bool {
		if standings[i].TeamParticipant.MatchPoints != standings[j].TeamParticipant.MatchPoints {
			return standings[i].TeamParticipant.MatchPoints > standings[j].TeamParticipant.MatchPoints
		}
		return standings[i].TeamParticipant.BoardPoints > standings[j].TeamParticipant.BoardPoints
	})

	// Assign ranks
	for i := range standings {
		standings[i].Rank = i + 1
	}

	return &TeamTournamentStandingsResponse{
		TournamentID: tournamentID,
		Standings:    standings,
		Tiebreaks:    settings.TeamTiebreaks,
	}, nil
}

// CreateTeamMatches creates team matches from pairings (used by pairing engine)
func (s *Service) CreateTeamMatches(ctx context.Context, tournamentID, roundID uuid.UUID, pairings []TeamMatch) error {
	for i := range pairings {
		pairings[i].ID = uuid.New()
		pairings[i].TournamentID = tournamentID
		pairings[i].RoundID = roundID
		pairings[i].Status = MatchStatusScheduled
		pairings[i].CreatedAt = time.Now()

		if err := s.repo.CreateTeamMatch(ctx, &pairings[i]); err != nil {
			return fmt.Errorf("failed to create match %d: %w", pairings[i].MatchNumber, err)
		}
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("round_id", roundID.String()).
		Int("count", len(pairings)).
		Msg("Team matches created")

	return nil
}

// CreateBoardPairings creates board pairings from lineups for a match
func (s *Service) CreateBoardPairings(ctx context.Context, matchID uuid.UUID) error {
	match, err := s.repo.GetTeamMatch(ctx, matchID)
	if err != nil {
		return err
	}

	settings, err := s.GetSettings(ctx, match.TournamentID)
	if err != nil {
		return err
	}

	// Get lineups
	var homeLineup, awayLineup *TeamLineup

	if match.HomeTeamID != nil {
		homeLineup, _ = s.repo.GetLineup(ctx, match.RoundID, *match.HomeTeamID)
	}
	if match.AwayTeamID != nil {
		awayLineup, _ = s.repo.GetLineup(ctx, match.RoundID, *match.AwayTeamID)
	}

	// Create boards
	for board := 1; board <= settings.BoardsPerMatch; board++ {
		boardEntry := &TeamMatchBoard{
			ID:          uuid.New(),
			TeamMatchID: matchID,
			BoardNumber: board,
		}

		// Get home player
		if homeLineup != nil {
			for _, entry := range homeLineup.Lineup {
				if entry.Board == board {
					boardEntry.HomePlayerID = &entry.PlayerID
					boardEntry.HomeRating = entry.Rating
					break
				}
			}
		}

		// Get away player
		if awayLineup != nil {
			for _, entry := range awayLineup.Lineup {
				if entry.Board == board {
					boardEntry.AwayPlayerID = &entry.PlayerID
					boardEntry.AwayRating = entry.Rating
					break
				}
			}
		}

		if err := s.repo.CreateTeamMatchBoard(ctx, boardEntry); err != nil {
			return fmt.Errorf("failed to create board %d: %w", board, err)
		}
	}

	s.logger.Info().
		Str("match_id", matchID.String()).
		Int("boards", settings.BoardsPerMatch).
		Msg("Board pairings created")

	return nil
}
