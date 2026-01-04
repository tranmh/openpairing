package team

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service handles team business logic
type Service struct {
	repo   *Repository
	logger zerolog.Logger
}

// NewService creates a new team service
func NewService(repo *Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("service", "team").Logger(),
	}
}

// Team CRUD

// CreateTeam creates a new team
func (s *Service) CreateTeam(ctx context.Context, req *CreateTeamRequest) (*Team, error) {
	team := &Team{
		ID:        uuid.New(),
		ClubID:    req.ClubID,
		Name:      req.Name,
		ShortName: req.ShortName,
		Division:  req.Division,
		Region:    req.Region,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateTeam(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	s.logger.Info().
		Str("team_id", team.ID.String()).
		Str("name", team.Name).
		Msg("Team created")

	return team, nil
}

// GetTeam retrieves a team by ID
func (s *Service) GetTeam(ctx context.Context, id uuid.UUID) (*Team, error) {
	return s.repo.GetTeamByID(ctx, id)
}

// GetTeamsByClub retrieves all teams for a club
func (s *Service) GetTeamsByClub(ctx context.Context, clubID uuid.UUID) ([]Team, error) {
	return s.repo.GetTeamsByClubID(ctx, clubID)
}

// UpdateTeam updates a team
func (s *Service) UpdateTeam(ctx context.Context, id uuid.UUID, req *UpdateTeamRequest) (*Team, error) {
	team, err := s.repo.GetTeamByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, fmt.Errorf("team not found")
	}

	if req.Name != "" {
		team.Name = req.Name
	}
	if req.ShortName != "" {
		team.ShortName = req.ShortName
	}
	if req.Division != "" {
		team.Division = req.Division
	}
	if req.Region != "" {
		team.Region = req.Region
	}
	team.UpdatedAt = time.Now()

	if err := s.repo.UpdateTeam(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}

	return team, nil
}

// DeleteTeam deletes a team
func (s *Service) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteTeam(ctx, id)
}

// Team Roster Management

// AddPlayerToRoster adds a player to a team's roster
func (s *Service) AddPlayerToRoster(ctx context.Context, teamID uuid.UUID, req *AddRosterMemberRequest) (*TeamMember, error) {
	member := &TeamMember{
		ID:          uuid.New(),
		TeamID:      teamID,
		PlayerID:    req.PlayerID,
		BoardNumber: req.BoardNumber,
		IsCaptain:   req.IsCaptain,
		IsReserve:   req.IsReserve,
		ValidFrom:   req.ValidFrom,
		ValidUntil:  req.ValidUntil,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.AddTeamMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add roster member: %w", err)
	}

	s.logger.Info().
		Str("team_id", teamID.String()).
		Str("player_id", req.PlayerID.String()).
		Msg("Player added to roster")

	return member, nil
}

// GetRoster retrieves a team's roster
func (s *Service) GetRoster(ctx context.Context, teamID uuid.UUID) ([]TeamMemberWithPlayer, error) {
	return s.repo.GetTeamMembers(ctx, teamID)
}

// RemovePlayerFromRoster removes a player from a team's roster
func (s *Service) RemovePlayerFromRoster(ctx context.Context, teamID, playerID uuid.UUID) error {
	return s.repo.RemoveTeamMember(ctx, teamID, playerID)
}

// UpdateRosterMember updates a roster member
func (s *Service) UpdateRosterMember(ctx context.Context, teamID uuid.UUID, req *UpdateRosterMemberRequest) error {
	member := &TeamMember{
		TeamID:      teamID,
		PlayerID:    req.PlayerID,
		BoardNumber: req.BoardNumber,
		IsCaptain:   req.IsCaptain,
		IsReserve:   req.IsReserve,
		ValidFrom:   req.ValidFrom,
		ValidUntil:  req.ValidUntil,
	}
	return s.repo.UpdateTeamMember(ctx, member)
}

// Tournament Registration

// RegisterTeamForTournament registers a team for a tournament
func (s *Service) RegisterTeamForTournament(ctx context.Context, tournamentID, teamID uuid.UUID, seedRating *int) (*TeamTournamentParticipant, error) {
	participant := &TeamTournamentParticipant{
		ID:                 uuid.New(),
		TournamentID:       tournamentID,
		TeamID:             teamID,
		RegistrationStatus: TeamRegistrationStatusRegistered,
		RegisteredAt:       time.Now(),
		SeedRating:         seedRating,
	}

	if err := s.repo.RegisterTeamForTournament(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to register team: %w", err)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("team_id", teamID.String()).
		Msg("Team registered for tournament")

	return participant, nil
}

// GetTournamentTeams retrieves all teams registered for a tournament
func (s *Service) GetTournamentTeams(ctx context.Context, tournamentID uuid.UUID) ([]TeamTournamentParticipantWithTeam, error) {
	return s.repo.GetTeamParticipants(ctx, tournamentID)
}

// WithdrawTeamFromTournament withdraws a team from a tournament
func (s *Service) WithdrawTeamFromTournament(ctx context.Context, participantID uuid.UUID) error {
	return s.repo.WithdrawTeamFromTournament(ctx, participantID)
}

// Lineup Management

// SubmitLineup submits or updates a team's lineup for a round
func (s *Service) SubmitLineup(ctx context.Context, req *SubmitLineupRequest, submittedBy uuid.UUID) (*TeamLineup, error) {
	// Get team tournament settings for validation
	settings, err := s.repo.GetTeamTournamentSettings(ctx, req.TournamentID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		settings = DefaultTeamTournamentSettings(req.TournamentID)
	}

	// Get team roster for validation
	participant, err := s.repo.GetTeamParticipantByID(ctx, req.TeamParticipantID)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, fmt.Errorf("team participant not found")
	}

	roster, err := s.repo.GetTeamMembers(ctx, participant.TeamID)
	if err != nil {
		return nil, err
	}

	// Validate lineup
	validationErrors := s.validateLineup(req.Lineup, roster, settings)

	now := time.Now()
	lineup := &TeamLineup{
		ID:                uuid.New(),
		TournamentID:      req.TournamentID,
		RoundID:           req.RoundID,
		TeamParticipantID: req.TeamParticipantID,
		Lineup:            req.Lineup,
		IsValid:           len(validationErrors) == 0,
		ValidationErrors:  validationErrors,
		SubmittedAt:       &now,
		SubmittedBy:       &submittedBy,
	}

	if err := s.repo.SaveLineup(ctx, lineup); err != nil {
		return nil, fmt.Errorf("failed to save lineup: %w", err)
	}

	s.logger.Info().
		Str("team_participant_id", req.TeamParticipantID.String()).
		Str("round_id", req.RoundID.String()).
		Bool("is_valid", lineup.IsValid).
		Msg("Lineup submitted")

	return lineup, nil
}

// validateLineup validates a lineup against rules
func (s *Service) validateLineup(lineup []LineupEntry, roster []TeamMemberWithPlayer, settings *TeamTournamentSettings) []string {
	var errors []string

	// Check board count
	if len(lineup) != settings.BoardsPerMatch {
		errors = append(errors, fmt.Sprintf("lineup must have exactly %d boards, got %d", settings.BoardsPerMatch, len(lineup)))
	}

	// Create roster map for quick lookup
	rosterMap := make(map[uuid.UUID]TeamMemberWithPlayer)
	for _, m := range roster {
		rosterMap[m.PlayerID] = m
	}

	// Check all players are on roster
	playerIDs := make(map[uuid.UUID]bool)
	for _, entry := range lineup {
		if _, exists := rosterMap[entry.PlayerID]; !exists {
			errors = append(errors, fmt.Sprintf("player %s is not on team roster", entry.PlayerID))
		}
		if playerIDs[entry.PlayerID] {
			errors = append(errors, fmt.Sprintf("player %s appears multiple times in lineup", entry.PlayerID))
		}
		playerIDs[entry.PlayerID] = true
	}

	// Check rating order if enforced
	if settings.EnforceRatingOrder && len(lineup) > 1 {
		// Sort by board number
		sortedLineup := make([]LineupEntry, len(lineup))
		copy(sortedLineup, lineup)
		sort.Slice(sortedLineup, func(i, j int) bool {
			return sortedLineup[i].Board < sortedLineup[j].Board
		})

		for i := 1; i < len(sortedLineup); i++ {
			prevRating := sortedLineup[i-1].Rating
			currRating := sortedLineup[i].Rating

			// Lower board should have higher or equal rating (with tolerance)
			if currRating > prevRating+settings.RatingOrderTolerance {
				errors = append(errors, fmt.Sprintf(
					"rating order violation: board %d (%d) has higher rating than board %d (%d)",
					sortedLineup[i].Board, currRating, sortedLineup[i-1].Board, prevRating,
				))
			}
		}
	}

	return errors
}

// GetLineup retrieves a team's lineup for a round
func (s *Service) GetLineup(ctx context.Context, roundID, teamParticipantID uuid.UUID) (*TeamLineup, error) {
	return s.repo.GetLineup(ctx, roundID, teamParticipantID)
}

// GetLineupsForRound retrieves all lineups for a round
func (s *Service) GetLineupsForRound(ctx context.Context, roundID uuid.UUID) ([]TeamLineup, error) {
	return s.repo.GetLineupsForRound(ctx, roundID)
}

// Match Management

// CreateTeamMatches creates team matches for a round (after pairing)
func (s *Service) CreateTeamMatches(ctx context.Context, tournamentID, roundID uuid.UUID, pairings []TeamPairing) ([]TeamMatch, error) {
	var matches []TeamMatch

	for i, pairing := range pairings {
		match := TeamMatch{
			ID:           uuid.New(),
			TournamentID: tournamentID,
			RoundID:      roundID,
			MatchNumber:  i + 1,
			HomeTeamID:   pairing.HomeTeamID,
			AwayTeamID:   pairing.AwayTeamID,
			Status:       TeamMatchStatusScheduled,
			CreatedAt:    time.Now(),
		}

		if err := s.repo.CreateTeamMatch(ctx, &match); err != nil {
			return nil, fmt.Errorf("failed to create team match: %w", err)
		}

		matches = append(matches, match)
	}

	s.logger.Info().
		Str("round_id", roundID.String()).
		Int("match_count", len(matches)).
		Msg("Team matches created")

	return matches, nil
}

// GetTeamMatch retrieves a team match by ID
func (s *Service) GetTeamMatch(ctx context.Context, id uuid.UUID) (*TeamMatchWithTeams, error) {
	return s.repo.GetTeamMatch(ctx, id)
}

// GetTeamMatchesForRound retrieves all team matches for a round
func (s *Service) GetTeamMatchesForRound(ctx context.Context, roundID uuid.UUID) ([]TeamMatchWithTeams, error) {
	return s.repo.GetTeamMatchesForRound(ctx, roundID)
}

// CreateBoardPairings creates board pairings for a team match based on lineups
func (s *Service) CreateBoardPairings(ctx context.Context, matchID uuid.UUID) ([]TeamMatchBoard, error) {
	match, err := s.repo.GetTeamMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if match == nil {
		return nil, fmt.Errorf("match not found")
	}

	settings, err := s.repo.GetTeamTournamentSettings(ctx, match.TournamentID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		settings = DefaultTeamTournamentSettings(match.TournamentID)
	}

	// Get lineups
	var homeLineup, awayLineup *TeamLineup
	if match.HomeTeamID != nil {
		homeLineup, _ = s.repo.GetLineup(ctx, match.RoundID, *match.HomeTeamID)
	}
	if match.AwayTeamID != nil {
		awayLineup, _ = s.repo.GetLineup(ctx, match.RoundID, *match.AwayTeamID)
	}

	var boards []TeamMatchBoard

	for boardNum := 1; boardNum <= settings.BoardsPerMatch; boardNum++ {
		board := TeamMatchBoard{
			ID:          uuid.New(),
			TeamMatchID: matchID,
			BoardNumber: boardNum,
		}

		// Find players for this board
		if homeLineup != nil {
			for _, entry := range homeLineup.Lineup {
				if entry.Board == boardNum {
					board.HomePlayerID = &entry.PlayerID
					board.HomeRating = &entry.Rating
					break
				}
			}
		}
		if awayLineup != nil {
			for _, entry := range awayLineup.Lineup {
				if entry.Board == boardNum {
					board.AwayPlayerID = &entry.PlayerID
					board.AwayRating = &entry.Rating
					break
				}
			}
		}

		if err := s.repo.CreateTeamMatchBoard(ctx, &board); err != nil {
			return nil, fmt.Errorf("failed to create board pairing: %w", err)
		}

		boards = append(boards, board)
	}

	s.logger.Info().
		Str("match_id", matchID.String()).
		Int("board_count", len(boards)).
		Msg("Board pairings created")

	return boards, nil
}

// GetMatchBoards retrieves all boards for a team match
func (s *Service) GetMatchBoards(ctx context.Context, matchID uuid.UUID) ([]TeamMatchBoardWithPlayers, error) {
	return s.repo.GetTeamMatchBoards(ctx, matchID)
}

// EnterBoardResult enters the result for a board
func (s *Service) EnterBoardResult(ctx context.Context, boardID uuid.UUID, result string) error {
	var homeScore, awayScore float64

	switch result {
	case "1-0":
		homeScore, awayScore = 1.0, 0.0
	case "0-1":
		homeScore, awayScore = 0.0, 1.0
	case "1/2-1/2":
		homeScore, awayScore = 0.5, 0.5
	case "0-0":
		homeScore, awayScore = 0.0, 0.0
	default:
		return fmt.Errorf("invalid result: %s", result)
	}

	return s.repo.UpdateBoardResult(ctx, boardID, homeScore, awayScore)
}

// CalculateMatchResult calculates the match result from board results
func (s *Service) CalculateMatchResult(ctx context.Context, matchID uuid.UUID) error {
	match, err := s.repo.GetTeamMatch(ctx, matchID)
	if err != nil {
		return err
	}
	if match == nil {
		return fmt.Errorf("match not found")
	}

	settings, err := s.repo.GetTeamTournamentSettings(ctx, match.TournamentID)
	if err != nil {
		return err
	}
	if settings == nil {
		settings = DefaultTeamTournamentSettings(match.TournamentID)
	}

	boards, err := s.repo.GetTeamMatchBoards(ctx, matchID)
	if err != nil {
		return err
	}

	var homeBoardPoints, awayBoardPoints float64
	completedBoards := 0

	for _, board := range boards {
		if board.HomeScore != nil && board.AwayScore != nil {
			homeBoardPoints += *board.HomeScore
			awayBoardPoints += *board.AwayScore
			completedBoards++
		}
	}

	// Only calculate if all boards have results
	if completedBoards < settings.BoardsPerMatch {
		return fmt.Errorf("not all boards have results: %d/%d completed", completedBoards, settings.BoardsPerMatch)
	}

	// Calculate match points
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

	if err := s.repo.UpdateTeamMatchResult(ctx, matchID, homeMatchPoints, awayMatchPoints, homeBoardPoints, awayBoardPoints); err != nil {
		return fmt.Errorf("failed to update match result: %w", err)
	}

	s.logger.Info().
		Str("match_id", matchID.String()).
		Float64("home_board_points", homeBoardPoints).
		Float64("away_board_points", awayBoardPoints).
		Msg("Match result calculated")

	return nil
}

// GetTeamStandings calculates and returns team standings
func (s *Service) GetTeamStandings(ctx context.Context, tournamentID uuid.UUID) ([]TeamStanding, error) {
	participants, err := s.repo.GetTeamParticipants(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	standings := make([]TeamStanding, 0, len(participants))
	for _, p := range participants {
		if p.RegistrationStatus == TeamRegistrationStatusWithdrawn {
			continue
		}
		standings = append(standings, TeamStanding{
			TeamParticipantID: p.ID,
			TeamID:            p.TeamID,
			TeamName:          p.TeamName,
			MatchPoints:       p.MatchPoints,
			BoardPoints:       p.BoardPoints,
			Buchholz:          p.Buchholz,
		})
	}

	// Sort by match points, then board points, then buchholz
	sort.Slice(standings, func(i, j int) bool {
		if standings[i].MatchPoints != standings[j].MatchPoints {
			return standings[i].MatchPoints > standings[j].MatchPoints
		}
		if standings[i].BoardPoints != standings[j].BoardPoints {
			return standings[i].BoardPoints > standings[j].BoardPoints
		}
		return standings[i].Buchholz > standings[j].Buchholz
	})

	// Assign ranks
	for i := range standings {
		standings[i].Rank = i + 1
	}

	return standings, nil
}

// Settings Management

// SaveTeamTournamentSettings saves team tournament settings
func (s *Service) SaveTeamTournamentSettings(ctx context.Context, settings *TeamTournamentSettings) error {
	return s.repo.SaveTeamTournamentSettings(ctx, settings)
}

// GetTeamTournamentSettings retrieves team tournament settings
func (s *Service) GetTeamTournamentSettings(ctx context.Context, tournamentID uuid.UUID) (*TeamTournamentSettings, error) {
	settings, err := s.repo.GetTeamTournamentSettings(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return DefaultTeamTournamentSettings(tournamentID), nil
	}
	return settings, nil
}

// TeamPairing represents a pairing between two teams
type TeamPairing struct {
	HomeTeamID *uuid.UUID
	AwayTeamID *uuid.UUID
}

// Request types

// CreateTeamRequest contains data for creating a team
type CreateTeamRequest struct {
	ClubID    *uuid.UUID `json:"clubId"`
	Name      string     `json:"name" binding:"required"`
	ShortName string     `json:"shortName"`
	Division  string     `json:"division"`
	Region    string     `json:"region"`
}

// UpdateTeamRequest contains data for updating a team
type UpdateTeamRequest struct {
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	Division  string `json:"division"`
	Region    string `json:"region"`
}

// AddRosterMemberRequest contains data for adding a roster member
type AddRosterMemberRequest struct {
	PlayerID    uuid.UUID  `json:"playerId" binding:"required"`
	BoardNumber *int       `json:"boardNumber"`
	IsCaptain   bool       `json:"isCaptain"`
	IsReserve   bool       `json:"isReserve"`
	ValidFrom   *time.Time `json:"validFrom"`
	ValidUntil  *time.Time `json:"validUntil"`
}

// UpdateRosterMemberRequest contains data for updating a roster member
type UpdateRosterMemberRequest struct {
	PlayerID    uuid.UUID  `json:"playerId" binding:"required"`
	BoardNumber *int       `json:"boardNumber"`
	IsCaptain   bool       `json:"isCaptain"`
	IsReserve   bool       `json:"isReserve"`
	ValidFrom   *time.Time `json:"validFrom"`
	ValidUntil  *time.Time `json:"validUntil"`
}

// SubmitLineupRequest contains data for submitting a lineup
type SubmitLineupRequest struct {
	TournamentID      uuid.UUID     `json:"tournamentId" binding:"required"`
	RoundID           uuid.UUID     `json:"roundId" binding:"required"`
	TeamParticipantID uuid.UUID     `json:"teamParticipantId" binding:"required"`
	Lineup            []LineupEntry `json:"lineup" binding:"required"`
}
