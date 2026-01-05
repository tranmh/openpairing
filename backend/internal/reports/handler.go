package reports

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpairing/openpairing/pkg/response"
	"github.com/rs/zerolog"
)

// Handler handles report-related HTTP requests
type Handler struct {
	dwzService  *DWZReportService
	normService *NormReportService
	trfService  *TRFReportService
	logger      zerolog.Logger
}

// NewHandler creates a new reports handler
func NewHandler(dwzService *DWZReportService, normService *NormReportService, trfService *TRFReportService, logger zerolog.Logger) *Handler {
	return &Handler{
		dwzService:  dwzService,
		normService: normService,
		trfService:  trfService,
		logger:      logger.With().Str("component", "reports_handler").Logger(),
	}
}

// RegisterRoutes registers the report routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc, arbiterMiddleware gin.HandlerFunc) {
	reports := r.Group("/reports")

	// Protected routes - require arbiter auth for viewing report data
	protected := reports.Group("")
	protected.Use(authMiddleware)
	protected.Use(arbiterMiddleware)
	{
		// DWZ reports - JSON data
		protected.GET("/dwz/:tournamentId", h.GetDWZReport)

		// TRF reports - JSON data
		protected.GET("/trf/:tournamentId", h.GetTRFReport)

		// Norm reports - JSON data
		protected.GET("/norms/:tournamentId", h.GetNormReport)
		protected.GET("/norms/:tournamentId/player/:playerId", h.GetPlayerNormStatus)
	}

	// Public download routes - file downloads (same as other exports)
	{
		reports.GET("/dwz/:tournamentId/csv", h.DownloadDWZCSV)
		reports.GET("/dwz/:tournamentId/xml", h.DownloadDWZXML)
		reports.GET("/trf/:tournamentId/download", h.DownloadTRF)
	}
}

// GetDWZReport returns DWZ report data for a tournament
// @Summary Get DWZ report data
// @Tags Reports
// @Param tournamentId path string true "Tournament ID"
// @Success 200 {object} DWZReport
// @Router /reports/dwz/{tournamentId} [get]
func (h *Handler) GetDWZReport(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("tournamentId"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	report, err := h.dwzService.GenerateReport(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to generate DWZ report")
		response.InternalServerError(c, "Failed to generate report")
		return
	}

	// Validate report
	gen := &DWZGenerator{report: report}
	validationErrors := gen.Validate()

	response.Success(c, http.StatusOK, gin.H{
		"report":           report,
		"validationErrors": validationErrors,
		"isValid":          len(validationErrors) == 0,
	})
}

// DownloadDWZCSV downloads DWZ report as CSV
// @Summary Download DWZ report as CSV
// @Tags Reports
// @Param tournamentId path string true "Tournament ID"
// @Produce text/csv
// @Success 200 {file} file
// @Router /reports/dwz/{tournamentId}/csv [get]
func (h *Handler) DownloadDWZCSV(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("tournamentId"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	report, err := h.dwzService.GenerateReport(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to generate DWZ report")
		response.InternalServerError(c, "Failed to generate report")
		return
	}

	gen := &DWZGenerator{report: report}
	csvData, err := gen.GenerateCSV()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate CSV")
		response.InternalServerError(c, "Failed to generate CSV")
		return
	}

	filename := fmt.Sprintf("dwz_%s.csv", sanitizeFilename(report.Tournament.Name))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Data(http.StatusOK, "text/csv", csvData)
}

// DownloadDWZXML downloads DWZ report as XML
// @Summary Download DWZ report as XML
// @Tags Reports
// @Param tournamentId path string true "Tournament ID"
// @Produce application/xml
// @Success 200 {file} file
// @Router /reports/dwz/{tournamentId}/xml [get]
func (h *Handler) DownloadDWZXML(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("tournamentId"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	report, err := h.dwzService.GenerateReport(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to generate DWZ report")
		response.InternalServerError(c, "Failed to generate report")
		return
	}

	gen := &DWZGenerator{report: report}
	xmlData, err := gen.GenerateXML()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate XML")
		response.InternalServerError(c, "Failed to generate XML")
		return
	}

	filename := fmt.Sprintf("dwz_%s.xml", sanitizeFilename(report.Tournament.Name))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Data(http.StatusOK, "application/xml", xmlData)
}

// GetTRFReport returns TRF report data for a tournament
// @Summary Get TRF report data
// @Tags Reports
// @Param tournamentId path string true "Tournament ID"
// @Success 200 {object} TRFReport
// @Router /reports/trf/{tournamentId} [get]
func (h *Handler) GetTRFReport(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("tournamentId"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	report, err := h.trfService.GenerateReport(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to generate TRF report")
		response.InternalServerError(c, "Failed to generate report")
		return
	}

	// Validate report
	validationErrors := report.Validate()

	response.Success(c, http.StatusOK, gin.H{
		"report":           report,
		"validationErrors": validationErrors,
		"isValid":          len(validationErrors) == 0,
	})
}

// DownloadTRF downloads TRF file
// @Summary Download TRF file
// @Tags Reports
// @Param tournamentId path string true "Tournament ID"
// @Produce text/plain
// @Success 200 {file} file
// @Router /reports/trf/{tournamentId}/download [get]
func (h *Handler) DownloadTRF(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("tournamentId"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	report, err := h.trfService.GenerateReport(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to generate TRF report")
		response.InternalServerError(c, "Failed to generate report")
		return
	}

	trfData, err := report.Generate()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate TRF")
		response.InternalServerError(c, "Failed to generate TRF")
		return
	}

	filename := fmt.Sprintf("%s.trf", sanitizeFilename(report.TournamentName))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Data(http.StatusOK, "text/plain", trfData)
}

// GetNormReport returns norm calculations for a tournament
// @Summary Get norm report
// @Tags Reports
// @Param tournamentId path string true "Tournament ID"
// @Success 200 {object} NormReport
// @Router /reports/norms/{tournamentId} [get]
func (h *Handler) GetNormReport(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("tournamentId"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	report, err := h.normService.GenerateNormReport(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to generate norm report")
		response.InternalServerError(c, "Failed to generate report")
		return
	}

	// Summarize results
	achieved := 0
	nearMiss := 0
	for _, r := range report.NormResults {
		if r.Achieved {
			achieved++
		} else {
			nearMiss++
		}
	}

	response.Success(c, http.StatusOK, gin.H{
		"report":   report,
		"summary": gin.H{
			"achieved": achieved,
			"nearMiss": nearMiss,
			"total":    len(report.NormResults),
		},
	})
}

// GetPlayerNormStatus returns norm status for a specific player
// @Summary Get player norm status
// @Tags Reports
// @Param tournamentId path string true "Tournament ID"
// @Param playerId path string true "Player ID"
// @Success 200 {object} map[string]NormResult
// @Router /reports/norms/{tournamentId}/player/{playerId} [get]
func (h *Handler) GetPlayerNormStatus(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("tournamentId"))
	if err != nil {
		response.BadRequest(c, "Invalid tournament ID")
		return
	}

	playerID, err := uuid.Parse(c.Param("playerId"))
	if err != nil {
		response.BadRequest(c, "Invalid player ID")
		return
	}

	report, err := h.normService.GenerateNormReport(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Str("tournamentId", tournamentID.String()).Msg("Failed to generate norm report")
		response.InternalServerError(c, "Failed to generate report")
		return
	}

	// Filter for specific player
	var playerNorms []NormResult
	for _, r := range report.NormResults {
		if r.PlayerID == playerID {
			playerNorms = append(playerNorms, r)
		}
	}

	if len(playerNorms) == 0 {
		response.NotFound(c, "No norm data found for player")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"playerNorms": playerNorms,
	})
}

// sanitizeFilename removes characters that are invalid in filenames
func sanitizeFilename(name string) string {
	result := make([]byte, 0, len(name))
	for _, b := range []byte(name) {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-' || b == '_' {
			result = append(result, b)
		} else if b == ' ' {
			result = append(result, '_')
		}
	}
	return string(result)
}

// TRFReport represents a FIDE Tournament Report Format report
type TRFReport struct {
	TournamentID    uuid.UUID  `json:"tournamentId"`
	TournamentName  string     `json:"tournamentName"`
	TournamentCode  string     `json:"tournamentCode"`
	City            string     `json:"city"`
	Federation      string     `json:"federation"`
	StartDate       string     `json:"startDate"`
	EndDate         string     `json:"endDate"`
	NumPlayers      int        `json:"numPlayers"`
	NumRatedPlayers int        `json:"numRatedPlayers"`
	NumRounds       int        `json:"numRounds"`
	RatingType      string     `json:"ratingType"`
	ChiefArbiter    string     `json:"chiefArbiter"`
	DeputyArbiter   string     `json:"deputyArbiter"`
	TimeControl     string     `json:"timeControl"`
	Players         []TRFPlayer `json:"players"`
	dataProvider    TournamentDataProvider `json:"-"`
}

// TRFPlayer represents a player in TRF format
type TRFPlayer struct {
	StartNo      int        `json:"startNo"`
	Gender       string     `json:"gender"`
	Title        string     `json:"title"`
	Name         string     `json:"name"`
	FIDERating   int        `json:"fideRating"`
	Federation   string     `json:"federation"`
	FIDEID       string     `json:"fideId"`
	BirthDate    string     `json:"birthDate"`
	Points       float64    `json:"points"`
	Rank         int        `json:"rank"`
	RoundResults []TRFRound `json:"roundResults"`
}

// TRFRound represents a single round result in TRF format
type TRFRound struct {
	OpponentNo int    `json:"opponentNo"`
	Color      string `json:"color"`  // w, b, or -
	Result     string `json:"result"` // 1, 0, =, +, -, etc.
}

// TRFReportService generates TRF reports
type TRFReportService struct {
	dataProvider TournamentDataProvider
}

// NewTRFReportService creates a new TRF report service
func NewTRFReportService(provider TournamentDataProvider) *TRFReportService {
	return &TRFReportService{
		dataProvider: provider,
	}
}

// GenerateReport generates a TRF report for a tournament
func (s *TRFReportService) GenerateReport(ctx context.Context, tournamentID uuid.UUID) (*TRFReport, error) {
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

	report := &TRFReport{
		TournamentID:   tournamentID,
		TournamentName: tournament.Name,
		City:           tournament.City,
		StartDate:      tournament.StartDate.Format("2006/01/02"),
		EndDate:        tournament.EndDate.Format("2006/01/02"),
		NumPlayers:     len(participants),
		NumRounds:      tournament.Rounds,
		ChiefArbiter:   tournament.ChiefArbiter,
		TimeControl:    tournament.TimeControl,
		RatingType:     "Standard",
	}

	// Build participant lookup
	participantByID := make(map[uuid.UUID]*ParticipantData)
	for i := range participants {
		participantByID[participants[i].PlayerID] = &participants[i]
		if participants[i].FideRating > 0 {
			report.NumRatedPlayers++
		}
	}

	// Build round results for each player
	playerRounds := make(map[int][]TRFRound)
	for _, game := range games {
		white := participantByID[game.WhiteID]
		black := participantByID[game.BlackID]

		// Only add game results if both players exist
		if white != nil && black != nil {
			playerRounds[white.StartNumber] = append(playerRounds[white.StartNumber], TRFRound{
				OpponentNo: black.StartNumber,
				Color:      "w",
				Result:     trfResult(game.WhiteScore),
			})
			playerRounds[black.StartNumber] = append(playerRounds[black.StartNumber], TRFRound{
				OpponentNo: white.StartNumber,
				Color:      "b",
				Result:     trfResult(game.BlackScore),
			})
		} else if white != nil {
			// Bye or forfeit for white
			playerRounds[white.StartNumber] = append(playerRounds[white.StartNumber], TRFRound{
				OpponentNo: 0,
				Color:      "-",
				Result:     trfResult(game.WhiteScore),
			})
		} else if black != nil {
			// Bye or forfeit for black
			playerRounds[black.StartNumber] = append(playerRounds[black.StartNumber], TRFRound{
				OpponentNo: 0,
				Color:      "-",
				Result:     trfResult(game.BlackScore),
			})
		}
	}

	// Add players
	for _, p := range participants {
		gender := "m"
		if p.Gender == "F" {
			gender = "f"
		}

		report.Players = append(report.Players, TRFPlayer{
			StartNo:      p.StartNumber,
			Gender:       gender,
			Name:         fmt.Sprintf("%s, %s", p.LastName, p.FirstName),
			FIDERating:   p.FideRating,
			FIDEID:       p.FideID,
			BirthDate:    fmt.Sprintf("%d", p.BirthYear),
			Points:       p.Score,
			Rank:         p.Rank,
			RoundResults: playerRounds[p.StartNumber],
		})
	}

	return report, nil
}

// Generate generates the TRF file content
func (r *TRFReport) Generate() ([]byte, error) {
	var result []byte

	// Header section
	result = append(result, []byte(fmt.Sprintf("012 %s\n", r.TournamentName))...)
	result = append(result, []byte(fmt.Sprintf("022 %s\n", r.City))...)
	result = append(result, []byte(fmt.Sprintf("032 %s\n", r.Federation))...)
	result = append(result, []byte(fmt.Sprintf("042 %s\n", r.StartDate))...)
	result = append(result, []byte(fmt.Sprintf("052 %s\n", r.EndDate))...)
	result = append(result, []byte(fmt.Sprintf("062 %d\n", r.NumPlayers))...)
	result = append(result, []byte(fmt.Sprintf("072 %d\n", r.NumRatedPlayers))...)
	result = append(result, []byte(fmt.Sprintf("092 %d\n", r.NumRounds))...)
	result = append(result, []byte(fmt.Sprintf("102 %s\n", r.ChiefArbiter))...)
	if r.DeputyArbiter != "" {
		result = append(result, []byte(fmt.Sprintf("112 %s\n", r.DeputyArbiter))...)
	}
	result = append(result, []byte(fmt.Sprintf("122 %s\n", r.TimeControl))...)

	// Player section
	for _, p := range r.Players {
		// Format: 001 StartNo Sex Title Name Rating Fed ID-No. Birth Pts Rank
		line := fmt.Sprintf("001 %4d %s %3s %-33s %4d %3s %11s %10s %5.1f %4d",
			p.StartNo, p.Gender, p.Title, p.Name, p.FIDERating, p.Federation,
			p.FIDEID, p.BirthDate, p.Points, p.Rank)

		// Add round results
		for _, round := range p.RoundResults {
			line += fmt.Sprintf("  %4d %s %s", round.OpponentNo, round.Color, round.Result)
		}
		result = append(result, []byte(line+"\n")...)
	}

	return result, nil
}

// Validate validates the TRF report
func (r *TRFReport) Validate() []string {
	var errors []string

	if r.TournamentName == "" {
		errors = append(errors, "Tournament name is required")
	}
	if r.StartDate == "" {
		errors = append(errors, "Start date is required")
	}
	if r.EndDate == "" {
		errors = append(errors, "End date is required")
	}
	if len(r.Players) == 0 {
		errors = append(errors, "At least one player is required")
	}

	// Check for missing FIDE IDs for rated players
	for _, p := range r.Players {
		if p.FIDERating > 0 && p.FIDEID == "" {
			errors = append(errors, fmt.Sprintf("Player %s has rating but no FIDE ID", p.Name))
		}
	}

	return errors
}

func trfResult(score float64) string {
	if score == 1.0 {
		return "1"
	} else if score == 0.5 {
		return "="
	}
	return "0"
}
