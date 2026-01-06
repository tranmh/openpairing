package federation

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/tranmh/openpairing/internal/federation/dsb"
	"github.com/tranmh/openpairing/internal/federation/fide"
	"github.com/tranmh/openpairing/internal/federation/oesb"
	"github.com/tranmh/openpairing/internal/federation/portal64"
	"github.com/tranmh/openpairing/internal/federation/ssb"
	"github.com/tranmh/openpairing/internal/player"
	"github.com/tranmh/openpairing/pkg/response"
)

// Handler handles federation-related HTTP requests
type Handler struct {
	fideImporter    *fide.Importer
	fideScheduler   *fide.Scheduler
	dsbImporter     *dsb.Importer
	dsbScheduler    *dsb.Scheduler
	portal64Service *portal64.Service
	dsbProvider     string // "portal64" or "dewis"
	oesbImporter    *oesb.Importer
	oesbScheduler   *oesb.Scheduler
	ssbImporter     *ssb.Importer
	ssbScheduler    *ssb.Scheduler
	playerRepo      player.Repository
	logger          zerolog.Logger
}

// HandlerConfig contains configuration for the federation handler
type HandlerConfig struct {
	FideImporter    *fide.Importer
	FideScheduler   *fide.Scheduler
	DSBImporter     *dsb.Importer
	DSBScheduler    *dsb.Scheduler
	Portal64Service *portal64.Service
	DSBProvider     string // "portal64" or "dewis" (default: "portal64")
	OESBImporter    *oesb.Importer
	OESBScheduler   *oesb.Scheduler
	SSBImporter     *ssb.Importer
	SSBScheduler    *ssb.Scheduler
	PlayerRepo      player.Repository
}

// NewHandler creates a new federation handler
func NewHandler(cfg *HandlerConfig, logger zerolog.Logger) *Handler {
	dsbProvider := cfg.DSBProvider
	if dsbProvider == "" {
		dsbProvider = "portal64" // default to portal64
	}

	return &Handler{
		fideImporter:    cfg.FideImporter,
		fideScheduler:   cfg.FideScheduler,
		dsbImporter:     cfg.DSBImporter,
		dsbScheduler:    cfg.DSBScheduler,
		portal64Service: cfg.Portal64Service,
		dsbProvider:     dsbProvider,
		oesbImporter:    cfg.OESBImporter,
		oesbScheduler:   cfg.OESBScheduler,
		ssbImporter:     cfg.SSBImporter,
		ssbScheduler:    cfg.SSBScheduler,
		playerRepo:      cfg.PlayerRepo,
		logger:          logger.With().Str("component", "federation_handler").Logger(),
	}
}

// RegisterRoutes registers the federation routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc, adminMiddleware gin.HandlerFunc) {
	federation := r.Group("/federation")
	{
		// Public routes - search across federations
		federation.GET("/search", h.SearchFederation)
		federation.GET("/status", h.GetSyncStatus)

		// Admin only routes
		admin := federation.Group("")
		admin.Use(authMiddleware)
		admin.Use(adminMiddleware)
		{
			// Import endpoints
			admin.POST("/import/fide", h.ImportFIDE)
			admin.POST("/import/dsb", h.ImportDSB)
			admin.POST("/import/dsb/player/:pkz", h.ImportDSBPlayer)
			admin.POST("/import/dsb/club/:zps", h.ImportDSBClub)
			admin.POST("/import/oesb", h.ImportOESB)
			admin.POST("/import/oesb/player/:id", h.ImportOESBPlayer)
			admin.POST("/import/ssb", h.ImportSSB)
			admin.POST("/import/ssb/player/:id", h.ImportSSBPlayer)

			// Sync trigger endpoints
			admin.POST("/sync/fide", h.TriggerFIDESync)
			admin.POST("/sync/dsb", h.TriggerDSBSync)
			admin.POST("/sync/oesb", h.TriggerOESBSync)
			admin.POST("/sync/ssb", h.TriggerSSBSync)
		}
	}
}

// SearchFederationRequest represents a federation search request
type SearchFederationRequest struct {
	Query      string `form:"q" binding:"required,min=2"`
	Federation string `form:"federation"` // fide, dsb, oesb, ssb
	Limit      int    `form:"limit"`
}

// SearchFederationResponse contains search results from federation APIs
type SearchFederationResponse struct {
	Results []FederationPlayer `json:"results"`
	Source  string             `json:"source"`
}

// FederationPlayer represents a player from any federation
type FederationPlayer struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	FirstName  string `json:"firstName,omitempty"`
	LastName   string `json:"lastName,omitempty"`
	Federation string `json:"federation"`
	Rating     int    `json:"rating,omitempty"`
	FideID     string `json:"fideId,omitempty"`
	FideRating int    `json:"fideRating,omitempty"`
	Title      string `json:"title,omitempty"`
	Club       string `json:"club,omitempty"`
}

// SearchFederation searches for players in federation databases
func (h *Handler) SearchFederation(c *gin.Context) {
	var req SearchFederationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	results := make([]FederationPlayer, 0)

	// Search based on federation or all federations
	switch req.Federation {
	case "fide":
		// FIDE search is done through our database after import
		response.BadRequest(c, "FIDE search requires database lookup - use /players endpoint")
		return

	case "dsb":
		if h.dsbProvider == "portal64" && h.portal64Service != nil {
			// Use portal64 API
			players, err := h.portal64Service.SearchPlayers(ctx, req.Query, req.Limit)
			if err != nil {
				h.logger.Warn().Err(err).Msg("Portal64 DSB search failed")
			} else {
				for _, p := range players {
					fideID := ""
					if p.FideID > 0 {
						fideID = fmt.Sprintf("%d", p.FideID)
					}
					results = append(results, FederationPlayer{
						ID:         p.ID,  // Use VKZ-Spielernummer (e.g., "56024-173") for GetPlayer API
						Name:       p.Name,
						FirstName:  p.Firstname,
						LastName:   p.Name,
						Federation: "DSB",
						Rating:     p.CurrentDWZ,
						FideID:     fideID,
						Club:       p.Club,
					})
				}
			}
		} else if h.dsbImporter != nil {
			// Fallback to DeWIS API
			dsbClient := dsb.NewClient(&dsb.ClientConfig{}, h.logger)
			players, err := dsbClient.SearchPlayers(ctx, req.Query, req.Limit)
			if err != nil {
				h.logger.Warn().Err(err).Msg("DSB search failed")
			} else {
				for _, p := range players {
					results = append(results, FederationPlayer{
						ID:         p.PKZ,
						Name:       p.Name,
						FirstName:  p.FirstName,
						LastName:   p.LastName,
						Federation: "DSB",
						Rating:     p.DWZ,
						FideID:     p.FideID,
						FideRating: p.FideElo,
						Title:      p.Title,
						Club:       p.ClubName,
					})
				}
			}
		}

	case "oesb":
		if h.oesbImporter != nil {
			oesbClient := oesb.NewClient(&oesb.ClientConfig{}, h.logger)
			players, err := oesbClient.SearchPlayers(ctx, req.Query, req.Limit)
			if err != nil {
				h.logger.Warn().Err(err).Msg("OESB search failed")
			} else {
				for _, p := range players {
					results = append(results, FederationPlayer{
						ID:         p.ID,
						Name:       p.Name,
						FirstName:  p.FirstName,
						LastName:   p.LastName,
						Federation: "OESB",
						Rating:     p.Rating,
						FideID:     p.FideID,
						FideRating: p.FideRating,
						Title:      p.Title,
						Club:       p.ClubName,
					})
				}
			}
		}

	case "ssb":
		if h.ssbImporter != nil {
			ssbClient := ssb.NewClient(&ssb.ClientConfig{}, h.logger)
			players, err := ssbClient.SearchPlayers(ctx, req.Query, req.Limit)
			if err != nil {
				h.logger.Warn().Err(err).Msg("SSB search failed")
			} else {
				for _, p := range players {
					results = append(results, FederationPlayer{
						ID:         p.ID,
						Name:       p.Name,
						FirstName:  p.FirstName,
						LastName:   p.LastName,
						Federation: "SSB",
						Rating:     p.Rating,
						FideID:     p.FideID,
						FideRating: p.FideRating,
						Title:      p.Title,
						Club:       p.ClubName,
					})
				}
			}
		}

	default:
		// Search all DACH federations
		if h.dsbProvider == "portal64" && h.portal64Service != nil {
			// Use portal64 API for DSB
			players, err := h.portal64Service.SearchPlayers(ctx, req.Query, req.Limit/3)
			if err == nil {
				for _, p := range players {
					fideID := ""
					if p.FideID > 0 {
						fideID = fmt.Sprintf("%d", p.FideID)
					}
					results = append(results, FederationPlayer{
						ID:         p.ID,  // Use VKZ-Spielernummer for GetPlayer API
						Name:       p.Name,
						FirstName:  p.Firstname,
						LastName:   p.Name,
						Federation: "DSB",
						Rating:     p.CurrentDWZ,
						FideID:     fideID,
						Club:       p.Club,
					})
				}
			}
		} else if h.dsbImporter != nil {
			// Fallback to DeWIS API
			dsbClient := dsb.NewClient(&dsb.ClientConfig{}, h.logger)
			players, err := dsbClient.SearchPlayers(ctx, req.Query, req.Limit/3)
			if err == nil {
				for _, p := range players {
					results = append(results, FederationPlayer{
						ID:         p.PKZ,
						Name:       p.Name,
						FirstName:  p.FirstName,
						LastName:   p.LastName,
						Federation: "DSB",
						Rating:     p.DWZ,
						FideID:     p.FideID,
						FideRating: p.FideElo,
						Title:      p.Title,
						Club:       p.ClubName,
					})
				}
			}
		}
		if h.oesbImporter != nil {
			oesbClient := oesb.NewClient(&oesb.ClientConfig{}, h.logger)
			players, err := oesbClient.SearchPlayers(ctx, req.Query, req.Limit/3)
			if err == nil {
				for _, p := range players {
					results = append(results, FederationPlayer{
						ID:         p.ID,
						Name:       p.Name,
						FirstName:  p.FirstName,
						LastName:   p.LastName,
						Federation: "OESB",
						Rating:     p.Rating,
						FideID:     p.FideID,
						FideRating: p.FideRating,
						Title:      p.Title,
						Club:       p.ClubName,
					})
				}
			}
		}
		if h.ssbImporter != nil {
			ssbClient := ssb.NewClient(&ssb.ClientConfig{}, h.logger)
			players, err := ssbClient.SearchPlayers(ctx, req.Query, req.Limit/3)
			if err == nil {
				for _, p := range players {
					results = append(results, FederationPlayer{
						ID:         p.ID,
						Name:       p.Name,
						FirstName:  p.FirstName,
						LastName:   p.LastName,
						Federation: "SSB",
						Rating:     p.Rating,
						FideID:     p.FideID,
						FideRating: p.FideRating,
						Title:      p.Title,
						Club:       p.ClubName,
					})
				}
			}
		}
	}

	c.JSON(http.StatusOK, SearchFederationResponse{
		Results: results,
		Source:  req.Federation,
	})
}

// SyncStatus represents the sync status for all federations
type SyncStatus struct {
	FIDE *FederationSyncStatus `json:"fide,omitempty"`
	DSB  *FederationSyncStatus `json:"dsb,omitempty"`
	OESB *FederationSyncStatus `json:"oesb,omitempty"`
	SSB  *FederationSyncStatus `json:"ssb,omitempty"`
}

// FederationSyncStatus represents the sync status for a single federation
type FederationSyncStatus struct {
	LastSync       *time.Time `json:"lastSync,omitempty"`
	PlayersUpdated int        `json:"playersUpdated,omitempty"`
	Errors         int        `json:"errors,omitempty"`
	Duration       string     `json:"duration,omitempty"`
	Healthy        bool       `json:"healthy"`
}

// GetSyncStatus returns the sync status for all federations
func (h *Handler) GetSyncStatus(c *gin.Context) {
	status := SyncStatus{}

	if h.fideScheduler != nil {
		lastSync, result := h.fideScheduler.GetLastSyncStatus()
		healthy := h.fideScheduler.IsHealthy()
		status.FIDE = &FederationSyncStatus{
			LastSync: &lastSync,
			Healthy:  healthy,
		}
		if result != nil {
			status.FIDE.PlayersUpdated = result.NewPlayers + result.UpdatedPlayers
			status.FIDE.Errors = result.Errors
			status.FIDE.Duration = result.ImportTime.String()
		}
	}

	if h.dsbScheduler != nil {
		lastSync, result := h.dsbScheduler.GetStatus()
		status.DSB = &FederationSyncStatus{
			LastSync: &lastSync,
			Healthy:  !lastSync.IsZero() && time.Since(lastSync) < 14*24*time.Hour,
		}
		if result != nil {
			status.DSB.PlayersUpdated = result.UpdatedPlayers
			status.DSB.Errors = result.Errors
			status.DSB.Duration = result.Duration.String()
		}
	}

	if h.oesbScheduler != nil {
		lastSync, result := h.oesbScheduler.GetStatus()
		status.OESB = &FederationSyncStatus{
			LastSync: &lastSync,
			Healthy:  !lastSync.IsZero() && time.Since(lastSync) < 14*24*time.Hour,
		}
		if result != nil {
			status.OESB.PlayersUpdated = result.UpdatedPlayers
			status.OESB.Errors = result.Errors
			status.OESB.Duration = result.Duration.String()
		}
	}

	if h.ssbScheduler != nil {
		lastSync, result := h.ssbScheduler.GetStatus()
		status.SSB = &FederationSyncStatus{
			LastSync: &lastSync,
			Healthy:  !lastSync.IsZero() && time.Since(lastSync) < 14*24*time.Hour,
		}
		if result != nil {
			status.SSB.PlayersUpdated = result.UpdatedPlayers
			status.SSB.Errors = result.Errors
			status.SSB.Duration = result.Duration.String()
		}
	}

	c.JSON(http.StatusOK, status)
}

// ImportResponse contains the result of an import operation
type ImportResponse struct {
	Success        bool   `json:"success"`
	TotalPlayers   int    `json:"totalPlayers"`
	UpdatedPlayers int    `json:"updatedPlayers"`
	NewPlayers     int    `json:"newPlayers"`
	Errors         int    `json:"errors"`
	Duration       string `json:"duration"`
	Message        string `json:"message,omitempty"`
}

// ImportFIDE triggers a full FIDE import
func (h *Handler) ImportFIDE(c *gin.Context) {
	if h.fideImporter == nil {
		response.BadRequest(c, "FIDE importer not configured")
		return
	}

	ctx := c.Request.Context()
	result, err := h.fideImporter.ImportLatest(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("FIDE import failed")
		response.InternalServerError(c, "FIDE import failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, ImportResponse{
		Success:        true,
		TotalPlayers:   result.TotalPlayers,
		UpdatedPlayers: result.UpdatedPlayers,
		NewPlayers:     result.NewPlayers,
		Errors:         result.Errors,
		Duration:       result.ImportTime.String(),
	})
}

// ImportDSB triggers a full DSB import (via search)
func (h *Handler) ImportDSB(c *gin.Context) {
	var req struct {
		Query string `json:"query" binding:"required"`
		Limit int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if h.dsbImporter == nil {
		response.BadRequest(c, "DSB importer not configured")
		return
	}

	if req.Limit <= 0 {
		req.Limit = 100
	}

	ctx := c.Request.Context()
	result, err := h.dsbImporter.SearchAndImport(ctx, req.Query, req.Limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("DSB import failed")
		response.InternalServerError(c, "DSB import failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, ImportResponse{
		Success:        true,
		TotalPlayers:   result.TotalPlayers,
		UpdatedPlayers: result.UpdatedPlayers,
		Errors:         result.Errors,
		Duration:       result.Duration.String(),
	})
}

// ImportDSBPlayer imports a single player from DSB by PKZ
func (h *Handler) ImportDSBPlayer(c *gin.Context) {
	pkz := c.Param("pkz")
	if pkz == "" {
		response.BadRequest(c, "PKZ required")
		return
	}

	ctx := c.Request.Context()

	// Use portal64 if configured
	if h.dsbProvider == "portal64" && h.portal64Service != nil {
		player, err := h.portal64Service.ImportPlayer(ctx, pkz)
		if err != nil {
			h.logger.Warn().Err(err).Str("pkz", pkz).Msg("Portal64 player import failed")
			response.NotFound(c, "Player not found or import failed")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"player":  player,
			"source":  "portal64",
		})
		return
	}

	// Fallback to DeWIS
	if h.dsbImporter == nil {
		response.BadRequest(c, "DSB importer not configured")
		return
	}

	player, err := h.dsbImporter.ImportPlayer(ctx, pkz)
	if err != nil {
		h.logger.Warn().Err(err).Str("pkz", pkz).Msg("DSB player import failed")
		response.NotFound(c, "Player not found or import failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"player":  player,
	})
}

// ImportDSBClub imports all members of a DSB club
func (h *Handler) ImportDSBClub(c *gin.Context) {
	zps := c.Param("zps")
	if zps == "" {
		response.BadRequest(c, "ZPS required")
		return
	}

	if h.dsbImporter == nil {
		response.BadRequest(c, "DSB importer not configured")
		return
	}

	ctx := c.Request.Context()
	result, err := h.dsbImporter.ImportClubMembers(ctx, zps)
	if err != nil {
		h.logger.Error().Err(err).Str("zps", zps).Msg("DSB club import failed")
		response.InternalServerError(c, "DSB club import failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, ImportResponse{
		Success:        true,
		TotalPlayers:   result.TotalPlayers,
		UpdatedPlayers: result.UpdatedPlayers,
		Errors:         result.Errors,
		Duration:       result.Duration.String(),
	})
}

// ImportOESB triggers a full Austrian federation import
func (h *Handler) ImportOESB(c *gin.Context) {
	if h.oesbImporter == nil {
		response.BadRequest(c, "OESB importer not configured")
		return
	}

	ctx := c.Request.Context()
	result, err := h.oesbImporter.ImportRatingsList(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("OESB import failed")
		response.InternalServerError(c, "OESB import failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, ImportResponse{
		Success:        true,
		TotalPlayers:   result.TotalPlayers,
		UpdatedPlayers: result.UpdatedPlayers,
		Errors:         result.Errors,
		Duration:       result.Duration.String(),
	})
}

// ImportOESBPlayer imports a single player from OESB by ID
func (h *Handler) ImportOESBPlayer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "ID required")
		return
	}

	if h.oesbImporter == nil {
		response.BadRequest(c, "OESB importer not configured")
		return
	}

	ctx := c.Request.Context()
	player, err := h.oesbImporter.ImportPlayer(ctx, id)
	if err != nil {
		h.logger.Warn().Err(err).Str("id", id).Msg("OESB player import failed")
		response.NotFound(c, "Player not found or import failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"player":  player,
	})
}

// ImportSSB triggers a full Swiss federation import
func (h *Handler) ImportSSB(c *gin.Context) {
	if h.ssbImporter == nil {
		response.BadRequest(c, "SSB importer not configured")
		return
	}

	ctx := c.Request.Context()
	result, err := h.ssbImporter.ImportRatingsList(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("SSB import failed")
		response.InternalServerError(c, "SSB import failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, ImportResponse{
		Success:        true,
		TotalPlayers:   result.TotalPlayers,
		UpdatedPlayers: result.UpdatedPlayers,
		Errors:         result.Errors,
		Duration:       result.Duration.String(),
	})
}

// ImportSSBPlayer imports a single player from SSB by ID
func (h *Handler) ImportSSBPlayer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "ID required")
		return
	}

	if h.ssbImporter == nil {
		response.BadRequest(c, "SSB importer not configured")
		return
	}

	ctx := c.Request.Context()
	player, err := h.ssbImporter.ImportPlayer(ctx, id)
	if err != nil {
		h.logger.Warn().Err(err).Str("id", id).Msg("SSB player import failed")
		response.NotFound(c, "Player not found or import failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"player":  player,
	})
}

// TriggerFIDESync manually triggers a FIDE sync
func (h *Handler) TriggerFIDESync(c *gin.Context) {
	if h.fideScheduler == nil {
		response.BadRequest(c, "FIDE scheduler not configured")
		return
	}

	ctx := c.Request.Context()
	result, err := h.fideScheduler.TriggerSync(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("FIDE sync trigger failed")
		response.InternalServerError(c, "FIDE sync failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, ImportResponse{
		Success:        true,
		TotalPlayers:   result.TotalPlayers,
		UpdatedPlayers: result.UpdatedPlayers,
		NewPlayers:     result.NewPlayers,
		Errors:         result.Errors,
		Duration:       result.ImportTime.String(),
		Message:        "FIDE sync triggered successfully",
	})
}

// TriggerDSBSync manually triggers a DSB sync
func (h *Handler) TriggerDSBSync(c *gin.Context) {
	if h.dsbScheduler == nil {
		response.BadRequest(c, "DSB scheduler not configured")
		return
	}

	ctx := c.Request.Context()
	result, err := h.dsbScheduler.TriggerSync(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("DSB sync trigger failed")
		response.InternalServerError(c, "DSB sync failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, ImportResponse{
		Success:        true,
		TotalPlayers:   result.TotalPlayers,
		UpdatedPlayers: result.UpdatedPlayers,
		Errors:         result.Errors,
		Duration:       result.Duration.String(),
		Message:        "DSB sync triggered successfully",
	})
}

// TriggerOESBSync manually triggers an OESB sync
func (h *Handler) TriggerOESBSync(c *gin.Context) {
	if h.oesbScheduler == nil {
		response.BadRequest(c, "OESB scheduler not configured")
		return
	}

	ctx := c.Request.Context()
	result, err := h.oesbScheduler.TriggerSync(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("OESB sync trigger failed")
		response.InternalServerError(c, "OESB sync failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, ImportResponse{
		Success:        true,
		TotalPlayers:   result.TotalPlayers,
		UpdatedPlayers: result.UpdatedPlayers,
		Errors:         result.Errors,
		Duration:       result.Duration.String(),
		Message:        "OESB sync triggered successfully",
	})
}

// TriggerSSBSync manually triggers an SSB sync
func (h *Handler) TriggerSSBSync(c *gin.Context) {
	if h.ssbScheduler == nil {
		response.BadRequest(c, "SSB scheduler not configured")
		return
	}

	ctx := c.Request.Context()
	result, err := h.ssbScheduler.TriggerSync(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("SSB sync trigger failed")
		response.InternalServerError(c, "SSB sync failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, ImportResponse{
		Success:        true,
		TotalPlayers:   result.TotalPlayers,
		UpdatedPlayers: result.UpdatedPlayers,
		Errors:         result.Errors,
		Duration:       result.Duration.String(),
		Message:        "SSB sync triggered successfully",
	})
}
