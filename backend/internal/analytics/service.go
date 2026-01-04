package analytics

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Service provides analytics and statistics
type Service struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewService creates a new analytics service
func NewService(db *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger.With().Str("component", "analytics").Logger(),
	}
}

// DashboardStats contains overall platform statistics
type DashboardStats struct {
	TotalTournaments   int            `json:"totalTournaments"`
	ActiveTournaments  int            `json:"activeTournaments"`
	TotalPlayers       int            `json:"totalPlayers"`
	TotalGames         int            `json:"totalGames"`
	TotalClubs         int            `json:"totalClubs"`
	TotalTeams         int            `json:"totalTeams"`
	RecentTournaments  []TournamentSummary `json:"recentTournaments"`
	UpcomingTournaments []TournamentSummary `json:"upcomingTournaments"`
	TopPlayers         []PlayerSummary     `json:"topPlayers"`
	ActivityByMonth    []MonthlyActivity   `json:"activityByMonth"`
}

// TournamentSummary contains summary info for a tournament
type TournamentSummary struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	StartDate     time.Time `json:"startDate"`
	EndDate       time.Time `json:"endDate"`
	Participants  int       `json:"participants"`
	Rounds        int       `json:"rounds"`
	Status        string    `json:"status"`
}

// PlayerSummary contains summary info for a player
type PlayerSummary struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Rating    int       `json:"rating"`
	Title     string    `json:"title,omitempty"`
	Club      string    `json:"club,omitempty"`
	GamesPlayed int     `json:"gamesPlayed"`
}

// MonthlyActivity contains activity stats for a month
type MonthlyActivity struct {
	Month       string `json:"month"`
	Tournaments int    `json:"tournaments"`
	Games       int    `json:"games"`
	Players     int    `json:"players"`
}

// GetDashboardStats retrieves overall platform statistics
func (s *Service) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Get total counts
	countQueries := []struct {
		query string
		dest  *int
	}{
		{"SELECT COUNT(*) FROM tournaments", &stats.TotalTournaments},
		{"SELECT COUNT(*) FROM tournaments WHERE status IN ('registration', 'ongoing')", &stats.ActiveTournaments},
		{"SELECT COUNT(*) FROM players", &stats.TotalPlayers},
		{"SELECT COUNT(*) FROM games", &stats.TotalGames},
		{"SELECT COUNT(*) FROM clubs", &stats.TotalClubs},
		{"SELECT COUNT(*) FROM teams", &stats.TotalTeams},
	}

	for _, q := range countQueries {
		if err := s.db.QueryRow(ctx, q.query).Scan(q.dest); err != nil {
			s.logger.Warn().Err(err).Str("query", q.query).Msg("Failed to get count")
		}
	}

	// Get recent tournaments
	recentQuery := `
		SELECT id, name, start_date, end_date,
			(SELECT COUNT(*) FROM tournament_participants WHERE tournament_id = t.id) as participants,
			rounds, status
		FROM tournaments t
		WHERE status = 'completed'
		ORDER BY end_date DESC
		LIMIT 5`

	rows, err := s.db.Query(ctx, recentQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t TournamentSummary
			if err := rows.Scan(&t.ID, &t.Name, &t.StartDate, &t.EndDate, &t.Participants, &t.Rounds, &t.Status); err == nil {
				stats.RecentTournaments = append(stats.RecentTournaments, t)
			}
		}
	}

	// Get upcoming tournaments
	upcomingQuery := `
		SELECT id, name, start_date, end_date,
			(SELECT COUNT(*) FROM tournament_participants WHERE tournament_id = t.id) as participants,
			rounds, status
		FROM tournaments t
		WHERE start_date > NOW() AND status IN ('draft', 'registration')
		ORDER BY start_date ASC
		LIMIT 5`

	rows, err = s.db.Query(ctx, upcomingQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t TournamentSummary
			if err := rows.Scan(&t.ID, &t.Name, &t.StartDate, &t.EndDate, &t.Participants, &t.Rounds, &t.Status); err == nil {
				stats.UpcomingTournaments = append(stats.UpcomingTournaments, t)
			}
		}
	}

	// Get top players by rating
	topPlayersQuery := `
		SELECT p.id, p.first_name || ' ' || p.last_name as name,
			COALESCE(p.fide_standard, p.dwz, 0) as rating,
			COALESCE(p.fide_title, '') as title,
			COALESCE(c.name, '') as club,
			(SELECT COUNT(*) FROM games WHERE white_player_id = p.id OR black_player_id = p.id) as games
		FROM players p
		LEFT JOIN clubs c ON p.club_id = c.id
		WHERE p.fide_standard > 0 OR p.dwz > 0
		ORDER BY COALESCE(p.fide_standard, p.dwz, 0) DESC
		LIMIT 10`

	rows, err = s.db.Query(ctx, topPlayersQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p PlayerSummary
			if err := rows.Scan(&p.ID, &p.Name, &p.Rating, &p.Title, &p.Club, &p.GamesPlayed); err == nil {
				stats.TopPlayers = append(stats.TopPlayers, p)
			}
		}
	}

	// Get monthly activity for last 12 months
	activityQuery := `
		SELECT
			to_char(date_trunc('month', t.start_date), 'YYYY-MM') as month,
			COUNT(DISTINCT t.id) as tournaments,
			COUNT(DISTINCT g.id) as games,
			COUNT(DISTINCT tp.player_id) as players
		FROM tournaments t
		LEFT JOIN games g ON g.tournament_id = t.id
		LEFT JOIN tournament_participants tp ON tp.tournament_id = t.id
		WHERE t.start_date >= NOW() - INTERVAL '12 months'
		GROUP BY date_trunc('month', t.start_date)
		ORDER BY month DESC`

	rows, err = s.db.Query(ctx, activityQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var a MonthlyActivity
			if err := rows.Scan(&a.Month, &a.Tournaments, &a.Games, &a.Players); err == nil {
				stats.ActivityByMonth = append(stats.ActivityByMonth, a)
			}
		}
	}

	return stats, nil
}

// TournamentStats contains statistics for a single tournament
type TournamentStats struct {
	TournamentID      uuid.UUID        `json:"tournamentId"`
	TournamentName    string           `json:"tournamentName"`
	TotalParticipants int              `json:"totalParticipants"`
	TotalGames        int              `json:"totalGames"`
	CompletedGames    int              `json:"completedGames"`
	AverageRating     int              `json:"averageRating"`
	HighestRating     int              `json:"highestRating"`
	LowestRating      int              `json:"lowestRating"`
	TitledPlayers     int              `json:"titledPlayers"`
	FederationCount   int              `json:"federationCount"`
	ResultDistribution ResultDistribution `json:"resultDistribution"`
	RoundStats        []RoundStats     `json:"roundStats"`
	RatingDistribution []RatingBucket  `json:"ratingDistribution"`
}

// ResultDistribution contains game result distribution
type ResultDistribution struct {
	WhiteWins   int     `json:"whiteWins"`
	BlackWins   int     `json:"blackWins"`
	Draws       int     `json:"draws"`
	WhiteWinPct float64 `json:"whiteWinPct"`
	BlackWinPct float64 `json:"blackWinPct"`
	DrawPct     float64 `json:"drawPct"`
}

// RoundStats contains statistics for a single round
type RoundStats struct {
	Round         int     `json:"round"`
	TotalGames    int     `json:"totalGames"`
	CompletedGames int    `json:"completedGames"`
	WhiteWins     int     `json:"whiteWins"`
	BlackWins     int     `json:"blackWins"`
	Draws         int     `json:"draws"`
	AverageGameLength int `json:"averageGameLength"`
}

// RatingBucket contains a rating distribution bucket
type RatingBucket struct {
	Min   int `json:"min"`
	Max   int `json:"max"`
	Count int `json:"count"`
}

// GetTournamentStats retrieves statistics for a tournament
func (s *Service) GetTournamentStats(ctx context.Context, tournamentID uuid.UUID) (*TournamentStats, error) {
	stats := &TournamentStats{
		TournamentID: tournamentID,
	}

	// Get basic tournament info
	infoQuery := `SELECT name FROM tournaments WHERE id = $1`
	if err := s.db.QueryRow(ctx, infoQuery, tournamentID).Scan(&stats.TournamentName); err != nil {
		return nil, fmt.Errorf("tournament not found: %w", err)
	}

	// Get participant stats
	participantQuery := `
		SELECT
			COUNT(*) as total,
			AVG(COALESCE(p.fide_standard, p.dwz, 0))::int as avg_rating,
			MAX(COALESCE(p.fide_standard, p.dwz, 0)) as max_rating,
			MIN(NULLIF(COALESCE(p.fide_standard, p.dwz, 0), 0)) as min_rating,
			COUNT(CASE WHEN p.fide_title IS NOT NULL AND p.fide_title != '' THEN 1 END) as titled,
			COUNT(DISTINCT p.federation) as federations
		FROM tournament_participants tp
		JOIN players p ON tp.player_id = p.id
		WHERE tp.tournament_id = $1`

	var minRating *int
	if err := s.db.QueryRow(ctx, participantQuery, tournamentID).Scan(
		&stats.TotalParticipants, &stats.AverageRating, &stats.HighestRating,
		&minRating, &stats.TitledPlayers, &stats.FederationCount); err == nil {
		if minRating != nil {
			stats.LowestRating = *minRating
		}
	}

	// Get game stats
	gameQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN white_score = 1 THEN 1 END) as white_wins,
			COUNT(CASE WHEN black_score = 1 THEN 1 END) as black_wins,
			COUNT(CASE WHEN white_score = 0.5 THEN 1 END) as draws
		FROM games
		WHERE tournament_id = $1`

	var whiteWins, blackWins, draws int
	if err := s.db.QueryRow(ctx, gameQuery, tournamentID).Scan(
		&stats.TotalGames, &stats.CompletedGames, &whiteWins, &blackWins, &draws); err == nil {
		stats.ResultDistribution = ResultDistribution{
			WhiteWins: whiteWins,
			BlackWins: blackWins,
			Draws:     draws,
		}
		if stats.CompletedGames > 0 {
			stats.ResultDistribution.WhiteWinPct = float64(whiteWins) / float64(stats.CompletedGames) * 100
			stats.ResultDistribution.BlackWinPct = float64(blackWins) / float64(stats.CompletedGames) * 100
			stats.ResultDistribution.DrawPct = float64(draws) / float64(stats.CompletedGames) * 100
		}
	}

	// Get round stats
	roundQuery := `
		SELECT
			r.round_number,
			COUNT(g.id) as total_games,
			COUNT(CASE WHEN g.status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN g.white_score = 1 THEN 1 END) as white_wins,
			COUNT(CASE WHEN g.black_score = 1 THEN 1 END) as black_wins,
			COUNT(CASE WHEN g.white_score = 0.5 THEN 1 END) as draws
		FROM rounds r
		LEFT JOIN games g ON g.round_id = r.id
		WHERE r.tournament_id = $1
		GROUP BY r.round_number
		ORDER BY r.round_number`

	rows, err := s.db.Query(ctx, roundQuery, tournamentID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rs RoundStats
			if err := rows.Scan(&rs.Round, &rs.TotalGames, &rs.CompletedGames,
				&rs.WhiteWins, &rs.BlackWins, &rs.Draws); err == nil {
				stats.RoundStats = append(stats.RoundStats, rs)
			}
		}
	}

	// Get rating distribution
	ratingQuery := `
		SELECT
			FLOOR(COALESCE(p.fide_standard, p.dwz, 0) / 200) * 200 as bucket_min,
			COUNT(*) as count
		FROM tournament_participants tp
		JOIN players p ON tp.player_id = p.id
		WHERE tp.tournament_id = $1 AND (p.fide_standard > 0 OR p.dwz > 0)
		GROUP BY bucket_min
		ORDER BY bucket_min`

	rows, err = s.db.Query(ctx, ratingQuery, tournamentID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var bucket RatingBucket
			if err := rows.Scan(&bucket.Min, &bucket.Count); err == nil {
				bucket.Max = bucket.Min + 199
				stats.RatingDistribution = append(stats.RatingDistribution, bucket)
			}
		}
	}

	return stats, nil
}

// PlayerStats contains statistics for a player
type PlayerStats struct {
	PlayerID          uuid.UUID         `json:"playerId"`
	PlayerName        string            `json:"playerName"`
	CurrentRating     int               `json:"currentRating"`
	HighestRating     int               `json:"highestRating"`
	LowestRating      int               `json:"lowestRating"`
	TotalGames        int               `json:"totalGames"`
	Wins              int               `json:"wins"`
	Losses            int               `json:"losses"`
	Draws             int               `json:"draws"`
	WinPercentage     float64           `json:"winPercentage"`
	TournamentsPlayed int               `json:"tournamentsPlayed"`
	PerformanceByYear []YearlyPerformance `json:"performanceByYear"`
	RecentGames       []GameSummary     `json:"recentGames"`
	OpponentHistory   []OpponentStat    `json:"opponentHistory"`
}

// YearlyPerformance contains performance stats for a year
type YearlyPerformance struct {
	Year          int     `json:"year"`
	Games         int     `json:"games"`
	Wins          int     `json:"wins"`
	Losses        int     `json:"losses"`
	Draws         int     `json:"draws"`
	AvgPerformance int    `json:"avgPerformance"`
	RatingChange  int     `json:"ratingChange"`
}

// GameSummary contains summary info for a game
type GameSummary struct {
	ID              uuid.UUID `json:"id"`
	TournamentName  string    `json:"tournamentName"`
	Round           int       `json:"round"`
	OpponentName    string    `json:"opponentName"`
	OpponentRating  int       `json:"opponentRating"`
	Color           string    `json:"color"`
	Result          string    `json:"result"`
	Date            time.Time `json:"date"`
}

// OpponentStat contains stats against a specific opponent
type OpponentStat struct {
	OpponentID     uuid.UUID `json:"opponentId"`
	OpponentName   string    `json:"opponentName"`
	Games          int       `json:"games"`
	Wins           int       `json:"wins"`
	Losses         int       `json:"losses"`
	Draws          int       `json:"draws"`
	Score          float64   `json:"score"`
}

// GetPlayerStats retrieves statistics for a player
func (s *Service) GetPlayerStats(ctx context.Context, playerID uuid.UUID) (*PlayerStats, error) {
	stats := &PlayerStats{
		PlayerID: playerID,
	}

	// Get basic player info
	playerQuery := `
		SELECT first_name || ' ' || last_name, COALESCE(fide_standard, dwz, 0)
		FROM players WHERE id = $1`
	if err := s.db.QueryRow(ctx, playerQuery, playerID).Scan(&stats.PlayerName, &stats.CurrentRating); err != nil {
		return nil, fmt.Errorf("player not found: %w", err)
	}

	// Get game stats
	gameStatsQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN
				(white_player_id = $1 AND white_score = 1) OR
				(black_player_id = $1 AND black_score = 1) THEN 1 END) as wins,
			COUNT(CASE WHEN
				(white_player_id = $1 AND white_score = 0) OR
				(black_player_id = $1 AND black_score = 0) THEN 1 END) as losses,
			COUNT(CASE WHEN
				(white_player_id = $1 OR black_player_id = $1) AND
				white_score = 0.5 THEN 1 END) as draws
		FROM games
		WHERE (white_player_id = $1 OR black_player_id = $1) AND status = 'completed'`

	if err := s.db.QueryRow(ctx, gameStatsQuery, playerID).Scan(
		&stats.TotalGames, &stats.Wins, &stats.Losses, &stats.Draws); err == nil {
		if stats.TotalGames > 0 {
			stats.WinPercentage = float64(stats.Wins) / float64(stats.TotalGames) * 100
		}
	}

	// Get tournaments played
	tournamentQuery := `SELECT COUNT(DISTINCT tournament_id) FROM tournament_participants WHERE player_id = $1`
	s.db.QueryRow(ctx, tournamentQuery, playerID).Scan(&stats.TournamentsPlayed)

	// Get recent games
	recentQuery := `
		SELECT g.id, t.name, r.round_number,
			CASE WHEN g.white_player_id = $1 THEN
				(SELECT first_name || ' ' || last_name FROM players WHERE id = g.black_player_id)
			ELSE
				(SELECT first_name || ' ' || last_name FROM players WHERE id = g.white_player_id)
			END as opponent,
			CASE WHEN g.white_player_id = $1 THEN
				(SELECT COALESCE(fide_standard, dwz, 0) FROM players WHERE id = g.black_player_id)
			ELSE
				(SELECT COALESCE(fide_standard, dwz, 0) FROM players WHERE id = g.white_player_id)
			END as opp_rating,
			CASE WHEN g.white_player_id = $1 THEN 'W' ELSE 'B' END as color,
			CASE
				WHEN g.white_player_id = $1 AND g.white_score = 1 THEN 'Win'
				WHEN g.black_player_id = $1 AND g.black_score = 1 THEN 'Win'
				WHEN g.white_score = 0.5 THEN 'Draw'
				ELSE 'Loss'
			END as result,
			g.created_at
		FROM games g
		JOIN rounds r ON g.round_id = r.id
		JOIN tournaments t ON g.tournament_id = t.id
		WHERE (g.white_player_id = $1 OR g.black_player_id = $1) AND g.status = 'completed'
		ORDER BY g.created_at DESC
		LIMIT 10`

	rows, err := s.db.Query(ctx, recentQuery, playerID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var g GameSummary
			if err := rows.Scan(&g.ID, &g.TournamentName, &g.Round, &g.OpponentName,
				&g.OpponentRating, &g.Color, &g.Result, &g.Date); err == nil {
				stats.RecentGames = append(stats.RecentGames, g)
			}
		}
	}

	// Get opponent history (most played opponents)
	opponentQuery := `
		SELECT
			CASE WHEN g.white_player_id = $1 THEN g.black_player_id ELSE g.white_player_id END as opp_id,
			(SELECT first_name || ' ' || last_name FROM players WHERE id =
				CASE WHEN g.white_player_id = $1 THEN g.black_player_id ELSE g.white_player_id END) as opp_name,
			COUNT(*) as games,
			SUM(CASE WHEN
				(g.white_player_id = $1 AND g.white_score = 1) OR
				(g.black_player_id = $1 AND g.black_score = 1) THEN 1 ELSE 0 END) as wins,
			SUM(CASE WHEN
				(g.white_player_id = $1 AND g.white_score = 0) OR
				(g.black_player_id = $1 AND g.black_score = 0) THEN 1 ELSE 0 END) as losses,
			SUM(CASE WHEN g.white_score = 0.5 THEN 1 ELSE 0 END) as draws
		FROM games g
		WHERE (g.white_player_id = $1 OR g.black_player_id = $1) AND g.status = 'completed'
		GROUP BY opp_id
		HAVING COUNT(*) >= 2
		ORDER BY games DESC
		LIMIT 10`

	rows, err = s.db.Query(ctx, opponentQuery, playerID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var o OpponentStat
			if err := rows.Scan(&o.OpponentID, &o.OpponentName, &o.Games, &o.Wins, &o.Losses, &o.Draws); err == nil {
				o.Score = float64(o.Wins) + float64(o.Draws)*0.5
				stats.OpponentHistory = append(stats.OpponentHistory, o)
			}
		}
	}

	// Sort opponent history by number of games
	sort.Slice(stats.OpponentHistory, func(i, j int) bool {
		return stats.OpponentHistory[i].Games > stats.OpponentHistory[j].Games
	})

	return stats, nil
}
