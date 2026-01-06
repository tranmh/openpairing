package wam

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tranmh/openpairing/internal/pairing/engine"
	"github.com/rs/zerolog"
)

var (
	ErrTournamentNotFound   = errors.New("tournament not found")
	ErrInvalidPhase         = errors.New("invalid operation for current phase")
	ErrGroupsNotComplete    = errors.New("not all groups are complete")
	ErrPlayoffsNotEnabled   = errors.New("playoffs are not enabled")
	ErrNoParticipants       = errors.New("no participants found")
)

// Service handles WAM tournament business logic
type Service struct {
	repo   Repository
	engine *engine.WAM
	logger zerolog.Logger
}

// NewService creates a new WAM service
func NewService(repo Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		engine: engine.NewWAM(logger),
		logger: logger.With().Str("service", "wam").Logger(),
	}
}

// ConfigureWAM creates or updates WAM configuration for a tournament
func (s *Service) ConfigureWAM(ctx context.Context, tournamentID uuid.UUID, req *ConfigureWAMRequest) (*WAMConfig, error) {
	// Build config
	config := &WAMConfig{
		TournamentID:      tournamentID,
		GroupSize:         req.GroupSize,
		DefaultGroupSize:  req.DefaultGroupSize,
		AllowedGroupSizes: req.AllowedGroupSizes,
		AssignmentMethod:  engine.WAMGroupAssignment(req.AssignmentMethod),
		EnablePlayoffs:    req.EnablePlayoffs,
		PlayoffsTopN:      req.PlayoffsTopN,
		ThirdPlaceMatch:   req.ThirdPlaceMatch,
		YouthAgeLimit:     req.YouthAgeLimit,
		CurrentPhase:      engine.WAMPhaseSetup,
		Tiebreaks:         req.Tiebreaks,
		Season:            req.Season,
	}

	// Set defaults
	if config.DefaultGroupSize == 0 {
		config.DefaultGroupSize = req.GroupSize
	}
	if config.DefaultGroupSize == 0 {
		config.DefaultGroupSize = 4
	}
	if len(config.AllowedGroupSizes) == 0 {
		config.AllowedGroupSizes = []int{4, 6}
	}
	if config.PlayoffsTopN == 0 {
		config.PlayoffsTopN = 1
	}
	if config.YouthAgeLimit == 0 {
		config.YouthAgeLimit = 12
	}
	if len(config.Tiebreaks) == 0 {
		config.Tiebreaks = []string{"direct_encounter", "sonneborn_berger", "buchholz"}
	}

	// Check if config exists
	existing, err := s.repo.GetConfig(ctx, tournamentID)
	if err != nil && !errors.Is(err, ErrConfigNotFound) {
		return nil, err
	}

	if existing != nil {
		// Update existing config
		config.CurrentPhase = existing.CurrentPhase // Preserve phase
		if err := s.repo.UpdateConfig(ctx, config); err != nil {
			return nil, err
		}
	} else {
		// Create new config
		if err := s.repo.CreateConfig(ctx, config); err != nil {
			return nil, err
		}
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Int("group_size", config.GroupSize).
		Str("assignment_method", string(config.AssignmentMethod)).
		Msg("WAM configured")

	return s.repo.GetConfig(ctx, tournamentID)
}

// GetConfig retrieves WAM configuration for a tournament
func (s *Service) GetConfig(ctx context.Context, tournamentID uuid.UUID) (*WAMConfig, error) {
	return s.repo.GetConfig(ctx, tournamentID)
}

// AssignGroups automatically assigns players to groups
func (s *Service) AssignGroups(ctx context.Context, tournamentID uuid.UUID, req *AssignGroupsRequest) (*AssignGroupsResponse, error) {
	config, err := s.repo.GetConfig(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if config.CurrentPhase != engine.WAMPhaseSetup {
		return nil, ErrInvalidPhase
	}

	// Get participants
	participants, err := s.repo.GetTournamentParticipants(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if len(participants) == 0 {
		return nil, ErrNoParticipants
	}

	// Convert to engine.Player format using effective rating (rating + joker)
	players := make([]engine.Player, len(participants))
	for i, p := range participants {
		effectiveRating := p.Rating + p.JokerPoints
		players[i] = engine.Player{
			StartNo:   i + 1,
			Name:      p.Name,
			Rating:    effectiveRating, // Use effective rating for sorting
			BirthDate: p.BirthDate,
		}
	}

	// Update config with assignment method from request
	config.AssignmentMethod = engine.WAMGroupAssignment(req.AssignmentMethod)

	// Delete existing groups
	if err := s.repo.DeleteAllGroups(ctx, tournamentID); err != nil {
		return nil, err
	}

	// Generate groups using engine
	engineGroups, warnings := s.engine.AssignGroups(players, config)

	// Save groups to database
	savedGroups := make([]WAMGroup, len(engineGroups))
	for i, eg := range engineGroups {
		group := WAMGroup{
			ID:           eg.ID,
			TournamentID: tournamentID,
			Name:         eg.Name,
			GroupNumber:  eg.GroupNumber,
			RoundsTotal:  eg.RoundsTotal,
			RoundsPlayed: 0,
			IsComplete:   false,
		}

		if err := s.repo.CreateGroup(ctx, &group); err != nil {
			return nil, err
		}

		// Add members
		for j, ep := range eg.Players {
			// Use OriginalIndex to map back to original participant
			// OriginalIndex is 0-indexed, set by the engine from player.StartNo - 1
			var participantID uuid.UUID
			var participant *TournamentParticipant
			if ep.OriginalIndex >= 0 && ep.OriginalIndex < len(participants) {
				participantID = participants[ep.OriginalIndex].ParticipantID
				participant = &participants[ep.OriginalIndex]
			}

			if participantID != uuid.Nil && participant != nil {
				member := &WAMGroupPlayer{
					GroupID:         group.ID,
					ParticipantID:   participantID,
					SortOrder:       j + 1,
					Rating:          participant.Rating,
					JokerPoints:     participant.JokerPoints,
					EffectiveRating: participant.Rating + participant.JokerPoints,
				}
				if err := s.repo.AddGroupMember(ctx, member); err != nil {
					return nil, err
				}
			}
		}

		savedGroups[i] = group
	}

	// Reload groups with members
	groups, err := s.repo.ListGroups(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Detect club conflicts
	conflicts := s.engine.DetectClubConflicts(groups)
	for _, conflict := range conflicts {
		warnings = append(warnings, fmt.Sprintf("Group %s: %d players from club %s",
			conflict.GroupName, len(conflict.Players), conflict.ClubName))
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Int("groups", len(groups)).
		Int("warnings", len(warnings)).
		Msg("Groups assigned")

	return &AssignGroupsResponse{
		Groups:   groups,
		Warnings: warnings,
	}, nil
}

// ListGroups returns all groups for a tournament
func (s *Service) ListGroups(ctx context.Context, tournamentID uuid.UUID) ([]WAMGroup, error) {
	return s.repo.ListGroups(ctx, tournamentID)
}

// GetGroup returns a single group with its members
func (s *Service) GetGroup(ctx context.Context, groupID uuid.UUID) (*WAMGroup, error) {
	return s.repo.GetGroup(ctx, groupID)
}

// AddPlayerToGroup adds a player to a group
func (s *Service) AddPlayerToGroup(ctx context.Context, groupID uuid.UUID, req *AddGroupMemberRequest) error {
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	config, err := s.repo.GetConfig(ctx, group.TournamentID)
	if err != nil {
		return err
	}

	if config.CurrentPhase != engine.WAMPhaseSetup && config.CurrentPhase != engine.WAMPhaseGroup {
		return ErrInvalidPhase
	}

	// Determine sort order
	sortOrder := req.SortOrder
	if sortOrder <= 0 {
		maxOrder, err := s.repo.GetMaxSortOrder(ctx, groupID)
		if err != nil {
			return err
		}
		sortOrder = maxOrder + 1
	}

	member := &WAMGroupPlayer{
		GroupID:       groupID,
		ParticipantID: req.ParticipantID,
		SortOrder:     sortOrder,
	}

	if err := s.repo.AddGroupMember(ctx, member); err != nil {
		return err
	}

	// Update group's total rounds
	group, err = s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	group.RoundsTotal = engine.GetTotalGroupRounds(len(group.Players))
	return s.repo.UpdateGroup(ctx, group)
}

// RemovePlayerFromGroup removes a player from a group
func (s *Service) RemovePlayerFromGroup(ctx context.Context, groupID, participantID uuid.UUID) error {
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	config, err := s.repo.GetConfig(ctx, group.TournamentID)
	if err != nil {
		return err
	}

	if config.CurrentPhase != engine.WAMPhaseSetup {
		return ErrInvalidPhase
	}

	if err := s.repo.RemoveGroupMember(ctx, groupID, participantID); err != nil {
		return err
	}

	// Update group's total rounds
	group, err = s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	group.RoundsTotal = engine.GetTotalGroupRounds(len(group.Players))
	return s.repo.UpdateGroup(ctx, group)
}

// MovePlayer moves a player from one group to another
func (s *Service) MovePlayer(ctx context.Context, tournamentID uuid.UUID, req *MovePlayerRequest) error {
	config, err := s.repo.GetConfig(ctx, tournamentID)
	if err != nil {
		return err
	}

	if config.CurrentPhase != engine.WAMPhaseSetup {
		return ErrInvalidPhase
	}

	// Find player's current group
	currentGroup, err := s.repo.GetPlayerGroup(ctx, tournamentID, req.ParticipantID)
	if err != nil {
		return err
	}

	if currentGroup.ID == req.TargetGroupID {
		return nil // Already in target group
	}

	if err := s.repo.MovePlayerToGroup(ctx, req.ParticipantID, currentGroup.ID, req.TargetGroupID, req.NewSortOrder); err != nil {
		return err
	}

	// Update rounds for both groups
	for _, groupID := range []uuid.UUID{currentGroup.ID, req.TargetGroupID} {
		group, err := s.repo.GetGroup(ctx, groupID)
		if err != nil {
			continue
		}
		group.RoundsTotal = engine.GetTotalGroupRounds(len(group.Players))
		_ = s.repo.UpdateGroup(ctx, group)
	}

	s.logger.Info().
		Str("participant_id", req.ParticipantID.String()).
		Str("from_group", currentGroup.Name).
		Str("to_group_id", req.TargetGroupID.String()).
		Msg("Player moved between groups")

	return nil
}

// GetGroupStandings calculates standings for a group
func (s *Service) GetGroupStandings(ctx context.Context, groupID uuid.UUID) (*GroupStandingsResponse, error) {
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	config, err := s.repo.GetConfig(ctx, group.TournamentID)
	if err != nil {
		return nil, err
	}

	// Get games for this group
	games, err := s.repo.GetGroupGames(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// Convert to engine format
	gameResults := make([]engine.GameResult, 0, len(games))
	for _, g := range games {
		if g.WhiteScore == nil || g.BlackScore == nil {
			continue // Skip incomplete games
		}

		// Find player sort orders
		var whiteOrder, blackOrder int
		for _, p := range group.Players {
			if g.WhiteParticipantID != nil && p.ParticipantID == *g.WhiteParticipantID {
				whiteOrder = p.SortOrder
			}
			if g.BlackParticipantID != nil && p.ParticipantID == *g.BlackParticipantID {
				blackOrder = p.SortOrder
			}
		}

		gameResults = append(gameResults, engine.GameResult{
			WhiteStartNo: whiteOrder,
			BlackStartNo: blackOrder,
			WhiteScore:   *g.WhiteScore,
			BlackScore:   *g.BlackScore,
		})
	}

	// Calculate standings
	standings := s.engine.CalculateGroupStandings(group, gameResults, config.Tiebreaks)

	// Detect club conflicts
	groups := []WAMGroup{*group}
	conflicts := s.engine.DetectClubConflicts(groups)

	return &GroupStandingsResponse{
		GroupID:       group.ID,
		GroupName:     group.Name,
		Standings:     standings,
		IsComplete:    group.IsComplete,
		ClubConflicts: conflicts,
	}, nil
}

// GetClubConflicts returns all club conflicts across groups
func (s *Service) GetClubConflicts(ctx context.Context, tournamentID uuid.UUID) ([]WAMClubConflict, error) {
	groups, err := s.repo.ListGroups(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	return s.engine.DetectClubConflicts(groups), nil
}

// StartGroupPhase transitions tournament to group phase
func (s *Service) StartGroupPhase(ctx context.Context, tournamentID uuid.UUID) error {
	config, err := s.repo.GetConfig(ctx, tournamentID)
	if err != nil {
		return err
	}

	if config.CurrentPhase != engine.WAMPhaseSetup {
		return ErrInvalidPhase
	}

	groups, err := s.repo.ListGroups(ctx, tournamentID)
	if err != nil {
		return err
	}

	if len(groups) == 0 {
		return errors.New("no groups created")
	}

	// Verify all groups have at least 2 players
	for _, g := range groups {
		if len(g.Players) < 2 {
			return fmt.Errorf("group %s has fewer than 2 players", g.Name)
		}
	}

	config.CurrentPhase = engine.WAMPhaseGroup
	return s.repo.UpdateConfig(ctx, config)
}

// GenerateGroupPairings generates pairings for a specific group round
func (s *Service) GenerateGroupPairings(ctx context.Context, groupID uuid.UUID, roundNumber int) ([]engine.Pairing, error) {
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	config, err := s.repo.GetConfig(ctx, group.TournamentID)
	if err != nil {
		return nil, err
	}

	if config.CurrentPhase != engine.WAMPhaseGroup {
		return nil, ErrInvalidPhase
	}

	// Determine round number
	if roundNumber <= 0 {
		roundNumber = group.RoundsPlayed + 1
	}

	if roundNumber > group.RoundsTotal {
		return nil, fmt.Errorf("round %d exceeds total rounds %d", roundNumber, group.RoundsTotal)
	}

	// Generate pairings
	pairings, err := s.engine.GenerateGroupPairings(group, roundNumber)
	if err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("group_id", groupID.String()).
		Str("group_name", group.Name).
		Int("round", roundNumber).
		Int("pairings", len(pairings)).
		Msg("Group pairings generated")

	return pairings, nil
}

// CompleteGroupRound marks a group round as complete and updates stats
func (s *Service) CompleteGroupRound(ctx context.Context, groupID uuid.UUID) error {
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	group.RoundsPlayed++
	if group.RoundsPlayed >= group.RoundsTotal {
		group.IsComplete = true
	}

	return s.repo.UpdateGroup(ctx, group)
}

// InitializePlayoffs creates the playoff bracket
func (s *Service) InitializePlayoffs(ctx context.Context, tournamentID uuid.UUID) (*WAMPlayoffBracket, error) {
	config, err := s.repo.GetConfig(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if !config.EnablePlayoffs {
		return nil, ErrPlayoffsNotEnabled
	}

	if config.CurrentPhase != engine.WAMPhaseGroup {
		return nil, ErrInvalidPhase
	}

	groups, err := s.repo.ListGroups(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Verify all groups are complete
	for _, g := range groups {
		if !g.IsComplete {
			return nil, ErrGroupsNotComplete
		}
	}

	// Generate bracket
	bracket, err := s.engine.GeneratePlayoffBracket(groups, config)
	if err != nil {
		return nil, err
	}

	// Delete existing playoff matches
	if err := s.repo.DeletePlayoffMatches(ctx, tournamentID); err != nil {
		return nil, err
	}

	// Save matches
	for _, match := range bracket.Matches {
		if err := s.repo.CreatePlayoffMatch(ctx, &match); err != nil {
			return nil, err
		}
	}

	// Update phase
	config.CurrentPhase = engine.WAMPhasePlayoff
	if err := s.repo.UpdateConfig(ctx, config); err != nil {
		return nil, err
	}

	// Record qualifications
	for _, g := range groups {
		groupStandings, err := s.GetGroupStandings(ctx, g.ID)
		if err != nil {
			continue
		}

		for _, standing := range groupStandings.Standings {
			if standing.Rank <= config.PlayoffsTopN {
				qual := &WAMQualification{
					TournamentID:  tournamentID,
					ParticipantID: standing.ParticipantID,
					GroupName:     g.Name,
					GroupRank:     standing.Rank,
					Season:        config.Season,
				}
				// Get player ID from participant
				for _, p := range g.Players {
					if p.ParticipantID == standing.ParticipantID {
						qual.PlayerID = p.PlayerID
						break
					}
				}
				_ = s.repo.AddQualification(ctx, qual)
			}
		}
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Int("matches", len(bracket.Matches)).
		Msg("Playoff bracket initialized")

	return bracket, nil
}

// GetPlayoffBracket returns the playoff bracket
func (s *Service) GetPlayoffBracket(ctx context.Context, tournamentID uuid.UUID) (*PlayoffBracketResponse, error) {
	config, err := s.repo.GetConfig(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	matches, err := s.repo.GetPlayoffMatches(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	bracket := &WAMPlayoffBracket{
		TournamentID: tournamentID,
		Matches:      matches,
	}

	// Determine completion status
	allComplete := true
	for _, m := range matches {
		if m.WinnerID == nil {
			allComplete = false
			break
		}
		if m.RoundName == "final" && m.WinnerID != nil {
			bracket.Champion = m.WinnerID
			bracket.RunnerUp = m.LoserID
		}
		if m.RoundName == "third_place" && m.WinnerID != nil {
			bracket.ThirdPlace = m.WinnerID
		}
	}
	bracket.IsComplete = allComplete && len(matches) > 0

	// Check if playoffs can be initialized
	canInitialize := false
	if config.EnablePlayoffs && config.CurrentPhase == engine.WAMPhaseGroup {
		groups, err := s.repo.ListGroups(ctx, tournamentID)
		if err == nil {
			canInitialize = true
			for _, g := range groups {
				if !g.IsComplete {
					canInitialize = false
					break
				}
			}
		}
	}

	return &PlayoffBracketResponse{
		Bracket:       bracket,
		CanInitialize: canInitialize,
		IsComplete:    bracket.IsComplete,
	}, nil
}

// GetQualifications returns qualified players
func (s *Service) GetQualifications(ctx context.Context, tournamentID uuid.UUID) (*QualificationsResponse, error) {
	config, err := s.repo.GetConfig(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	qualifications, err := s.repo.GetQualifications(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	return &QualificationsResponse{
		Season:         config.Season,
		Qualifications: qualifications,
		TotalCount:     len(qualifications),
	}, nil
}

// GetStatus returns overall WAM tournament status
func (s *Service) GetStatus(ctx context.Context, tournamentID uuid.UUID) (*WAMTournamentStatus, error) {
	config, err := s.repo.GetConfig(ctx, tournamentID)
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return &WAMTournamentStatus{
				TournamentID: tournamentID,
				Phase:        engine.WAMPhaseSetup,
			}, nil
		}
		return nil, err
	}

	groups, err := s.repo.ListGroups(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	participants, err := s.repo.GetTournamentParticipants(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Count assigned players
	assignedCount := 0
	for _, g := range groups {
		assignedCount += len(g.Players)
	}

	// Build group status
	groupStatus := make([]WAMGroupStatus, len(groups))
	allComplete := true
	for i, g := range groups {
		gamesForGroup, _ := s.repo.GetGroupGames(ctx, g.ID)
		completeGames := 0
		for _, game := range gamesForGroup {
			if game.Result != nil && *game.Result != "" {
				completeGames++
			}
		}

		totalGames := (len(g.Players) * (len(g.Players) - 1)) / 2

		groupStatus[i] = WAMGroupStatus{
			ID:            g.ID,
			Name:          g.Name,
			PlayerCount:   len(g.Players),
			RoundsTotal:   g.RoundsTotal,
			RoundsPlayed:  g.RoundsPlayed,
			GamesComplete: completeGames,
			GamesTotal:    totalGames,
			IsComplete:    g.IsComplete,
		}

		if !g.IsComplete {
			allComplete = false
		}
	}

	// Check warnings
	var warnings []string
	conflicts := s.engine.DetectClubConflicts(groups)
	for _, c := range conflicts {
		warnings = append(warnings, fmt.Sprintf("Group %s: %d players from %s",
			c.GroupName, len(c.Players), c.ClubName))
	}

	ageWarnings := s.engine.CheckAgeWarnings(groups, config, time.Now())
	for _, w := range ageWarnings {
		warnings = append(warnings, w.Message)
	}

	return &WAMTournamentStatus{
		TournamentID:     tournamentID,
		Config:           config,
		Phase:            config.CurrentPhase,
		Groups:           groupStatus,
		TotalPlayers:     len(participants),
		PlayersAssigned:  assignedCount,
		CanStartGroups:   config.CurrentPhase == engine.WAMPhaseSetup && assignedCount > 0,
		CanStartPlayoffs: config.EnablePlayoffs && config.CurrentPhase == engine.WAMPhaseGroup && allComplete,
		Warnings:         warnings,
	}, nil
}

// SetJokerPoints sets joker points for a participant (only allowed in setup phase)
func (s *Service) SetJokerPoints(ctx context.Context, tournamentID, participantID uuid.UUID, jokerPoints int) error {
	config, err := s.repo.GetConfig(ctx, tournamentID)
	if err != nil {
		return err
	}

	// Only allow in setup phase (before group assignment)
	if config.CurrentPhase != engine.WAMPhaseSetup {
		return ErrInvalidPhase
	}

	// Validate joker points
	if jokerPoints != 0 && jokerPoints != 50 && jokerPoints != 100 && jokerPoints != 150 {
		return fmt.Errorf("invalid joker points: must be 0, 50, 100, or 150")
	}

	if err := s.repo.SetJokerPoints(ctx, tournamentID, participantID, jokerPoints); err != nil {
		return err
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Str("participant_id", participantID.String()).
		Int("joker_points", jokerPoints).
		Msg("Joker points set")

	return nil
}

// ProposeGroups generates a group proposal for review
func (s *Service) ProposeGroups(ctx context.Context, tournamentID uuid.UUID) (*GroupProposal, error) {
	config, err := s.repo.GetConfig(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if config.CurrentPhase != engine.WAMPhaseSetup {
		return nil, ErrInvalidPhase
	}

	// Get participants
	participants, err := s.repo.GetTournamentParticipants(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if len(participants) == 0 {
		return nil, ErrNoParticipants
	}

	// Convert to engine.Player format with effective rating
	players := make([]engine.Player, len(participants))
	u12Flags := make(map[int]bool)
	refDate := time.Now()

	for i, p := range participants {
		effectiveRating := p.Rating + p.JokerPoints
		players[i] = engine.Player{
			StartNo:   i + 1,
			Name:      p.Name,
			Rating:    effectiveRating,
			BirthDate: p.BirthDate,
		}

		// Check U12 status
		if p.BirthDate != nil && config.YouthAgeLimit > 0 {
			if engine.IsU12Player(p.BirthDate, refDate, config.YouthAgeLimit) {
				u12Flags[i+1] = true
			}
		}
	}

	// Generate proposal using engine
	engineProposal, err := s.engine.ProposeGroups(players, config, u12Flags, refDate)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	proposal := &GroupProposal{
		TotalPlayers: engineProposal.TotalPlayers,
		Warnings:     engineProposal.Warnings,
	}

	// Convert groups
	for _, eg := range engineProposal.Groups {
		group := ProposedGroup{
			Name:       eg.Name,
			TargetSize: eg.TargetSize,
			IsU12Group: eg.IsU12Group,
			Players:    make([]TournamentParticipant, 0, len(eg.Players)),
		}

		for _, ep := range eg.Players {
			// Map back to participant
			if ep.OriginalIndex >= 0 && ep.OriginalIndex < len(participants) {
				p := participants[ep.OriginalIndex]
				p.IsU12 = ep.IsU12
				p.Age = ep.Age
				group.Players = append(group.Players, p)
			}
		}

		proposal.Groups = append(proposal.Groups, group)
	}

	// Convert U12 proposals
	for _, u12p := range engineProposal.U12Proposals {
		u12Proposal := U12GroupProposal{
			GroupName:        u12p.GroupName,
			Reason:           u12p.Reason,
			RequiresApproval: u12p.RequiresApproval,
			Players:          make([]TournamentParticipant, 0, len(u12p.Players)),
		}

		for _, ep := range u12p.Players {
			// BUGFIX: Use OriginalIndex instead of name matching to avoid duplicates
			if ep.OriginalIndex >= 0 && ep.OriginalIndex < len(participants) {
				p := participants[ep.OriginalIndex]
				p.IsU12 = true
				u12Proposal.Players = append(u12Proposal.Players, p)
			}
		}

		proposal.U12Proposals = append(proposal.U12Proposals, u12Proposal)
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Int("groups", len(proposal.Groups)).
		Int("u12_proposals", len(proposal.U12Proposals)).
		Msg("Group proposal generated")

	return proposal, nil
}

// ConfirmGroups confirms a proposed group configuration
func (s *Service) ConfirmGroups(ctx context.Context, tournamentID uuid.UUID, req *ConfirmGroupsRequest) (*AssignGroupsResponse, error) {
	config, err := s.repo.GetConfig(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if config.CurrentPhase != engine.WAMPhaseSetup {
		return nil, ErrInvalidPhase
	}

	// Get participants for validation
	participants, err := s.repo.GetTournamentParticipants(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Create participant lookup map
	participantMap := make(map[uuid.UUID]*TournamentParticipant)
	for i := range participants {
		participantMap[participants[i].ParticipantID] = &participants[i]
	}

	// Delete existing groups
	if err := s.repo.DeleteAllGroups(ctx, tournamentID); err != nil {
		return nil, err
	}

	var warnings []string

	// Create groups from confirmed proposal
	for i, confirmed := range req.Proposals {
		targetSize := confirmed.TargetSize
		if targetSize == 0 {
			targetSize = len(confirmed.ParticipantIDs)
		}

		group := WAMGroup{
			ID:           uuid.New(),
			TournamentID: tournamentID,
			Name:         confirmed.GroupName,
			GroupNumber:  i + 1,
			TargetSize:   targetSize,
			RoundsTotal:  engine.GetTotalGroupRounds(len(confirmed.ParticipantIDs)),
			RoundsPlayed: 0,
			IsComplete:   false,
		}

		if err := s.repo.CreateGroup(ctx, &group); err != nil {
			return nil, err
		}

		// Add members
		for j, pid := range confirmed.ParticipantIDs {
			participant := participantMap[pid]
			if participant == nil {
				warnings = append(warnings, fmt.Sprintf("Participant %s not found", pid))
				continue
			}

			member := &WAMGroupPlayer{
				GroupID:         group.ID,
				ParticipantID:   pid,
				SortOrder:       j + 1,
				Rating:          participant.Rating,
				JokerPoints:     participant.JokerPoints,
				EffectiveRating: participant.Rating + participant.JokerPoints,
			}
			if err := s.repo.AddGroupMember(ctx, member); err != nil {
				return nil, err
			}
		}
	}

	// Reload groups with members
	groups, err := s.repo.ListGroups(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Detect club conflicts
	conflicts := s.engine.DetectClubConflicts(groups)
	for _, conflict := range conflicts {
		warnings = append(warnings, fmt.Sprintf("Gruppe %s: %d Spieler vom Verein %s",
			conflict.GroupName, len(conflict.Players), conflict.ClubName))
	}

	s.logger.Info().
		Str("tournament_id", tournamentID.String()).
		Int("groups", len(groups)).
		Msg("Groups confirmed")

	return &AssignGroupsResponse{
		Groups:   groups,
		Warnings: warnings,
	}, nil
}
