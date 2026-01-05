package registration

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/openpairing/openpairing/pkg/response"
)

// Handler handles registration HTTP requests
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler creates a new registration handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("handler", "registration").Logger(),
	}
}

// RegisterRoutes registers registration routes
func (h *Handler) RegisterRoutes(r *gin.Engine, authMiddleware, optionalAuthMiddleware, organizerMiddleware gin.HandlerFunc) {
	// Public routes (no auth required, but optional auth supported)
	public := r.Group("/api/public")
	public.Use(optionalAuthMiddleware)
	{
		// Tournament info and registration
		public.GET("/tournaments/:slug", h.GetPublicTournament)
		public.POST("/tournaments/:slug/pre-register", h.PreRegister)
		public.POST("/tournaments/:slug/check-in", h.CheckIn)
	}

	// Protected routes (auth required)
	protected := r.Group("/api/tournaments/:id")
	protected.Use(authMiddleware)
	{
		// User's own registration
		protected.POST("/pre-register", h.PreRegisterAuthenticated)
		protected.POST("/check-in", h.CheckInAuthenticated)
		protected.GET("/my-registration", h.GetMyRegistration)
	}

	// Organizer routes
	organizer := r.Group("/api/tournaments/:id")
	organizer.Use(authMiddleware, organizerMiddleware)
	{
		// Registration management
		organizer.GET("/registrations", h.GetRegistrations)
		organizer.GET("/registrations/pending", h.GetPendingRegistrations)
		organizer.POST("/registrations/:regId/approve", h.ApproveRegistration)
		organizer.POST("/registrations/:regId/check-in", h.ManualCheckIn)
		organizer.GET("/check-in/status", h.GetCheckInStatus)

		// Settings
		organizer.GET("/registration-settings", h.GetRegistrationSettings)
		organizer.PUT("/registration-settings", h.UpdateRegistrationSettings)

		// Invite tokens
		organizer.POST("/invite-tokens", h.CreateInviteToken)
		organizer.GET("/invite-tokens", h.ListInviteTokens)
		organizer.DELETE("/invite-tokens/:tokenId", h.DeleteInviteToken)
	}
}

// --- Public Handlers ---

// GetPublicTournament returns public tournament info
func (h *Handler) GetPublicTournament(c *gin.Context) {
	slug := c.Param("slug")

	info, settings, err := h.service.GetPublicTournament(c.Request.Context(), slug)
	if err != nil {
		if err == ErrTournamentNotFound {
			response.NotFound(c, "Turnier nicht gefunden")
			return
		}
		h.logger.Error().Err(err).Str("slug", slug).Msg("Failed to get tournament")
		response.InternalServerError(c, "Fehler beim Laden des Turniers")
		return
	}

	// Include registration status info
	result := gin.H{
		"tournament":           info,
		"registrationSettings": settings,
		"isCheckInOpen":        h.service.IsCheckInWindowOpen(info, settings),
	}

	response.Success(c, http.StatusOK, result)
}

// PreRegister handles guest pre-registration
func (h *Handler) PreRegister(c *gin.Context) {
	slug := c.Param("slug")

	// Get tournament by slug to get ID
	info, _, err := h.service.GetPublicTournament(c.Request.Context(), slug)
	if err != nil {
		if err == ErrTournamentNotFound {
			response.NotFound(c, "Turnier nicht gefunden")
			return
		}
		h.logger.Error().Err(err).Msg("Failed to get tournament")
		response.InternalServerError(c, "Fehler beim Laden des Turniers")
		return
	}

	var req PreRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Ungültige Anfrage")
		return
	}

	// Get user ID if authenticated
	var userID *uuid.UUID
	if id, exists := c.Get("user_id"); exists {
		if idStr, ok := id.(string); ok {
			if uid, err := uuid.Parse(idStr); err == nil {
				userID = &uid
			}
		}
	}

	result, err := h.service.PreRegister(c.Request.Context(), info.ID, &req, userID)
	if err != nil {
		h.handleRegistrationError(c, err)
		return
	}

	response.Created(c, result)
}

// CheckIn handles self-service check-in
func (h *Handler) CheckIn(c *gin.Context) {
	slug := c.Param("slug")

	// Get tournament by slug to get ID
	info, _, err := h.service.GetPublicTournament(c.Request.Context(), slug)
	if err != nil {
		if err == ErrTournamentNotFound {
			response.NotFound(c, "Turnier nicht gefunden")
			return
		}
		h.logger.Error().Err(err).Msg("Failed to get tournament")
		response.InternalServerError(c, "Fehler beim Laden des Turniers")
		return
	}

	var req CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Ungültige Anfrage")
		return
	}

	// Get user ID if authenticated
	var userID *uuid.UUID
	if id, exists := c.Get("user_id"); exists {
		if idStr, ok := id.(string); ok {
			if uid, err := uuid.Parse(idStr); err == nil {
				userID = &uid
			}
		}
	}

	result, err := h.service.CheckIn(c.Request.Context(), info.ID, &req, userID)
	if err != nil {
		h.handleRegistrationError(c, err)
		return
	}

	response.Success(c, http.StatusOK, result)
}

// --- Authenticated User Handlers ---

// PreRegisterAuthenticated handles authenticated user pre-registration
func (h *Handler) PreRegisterAuthenticated(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Ungültige Turnier-ID")
		return
	}

	var req PreRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Ungültige Anfrage")
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	result, err := h.service.PreRegister(c.Request.Context(), tournamentID, &req, &userID)
	if err != nil {
		h.handleRegistrationError(c, err)
		return
	}

	response.Created(c, result)
}

// CheckInAuthenticated handles authenticated user check-in
func (h *Handler) CheckInAuthenticated(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Ungültige Turnier-ID")
		return
	}

	var req CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body for authenticated check-in
		req = CheckInRequest{Method: "self_service"}
	}

	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	result, err := h.service.CheckIn(c.Request.Context(), tournamentID, &req, &userID)
	if err != nil {
		h.handleRegistrationError(c, err)
		return
	}

	response.Success(c, http.StatusOK, result)
}

// GetMyRegistration returns the current user's registration status
func (h *Handler) GetMyRegistration(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Ungültige Turnier-ID")
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	// Get user's email for guest lookup
	userEmail, _ := c.Get("user_email")
	var email *string
	if e, ok := userEmail.(string); ok && e != "" {
		email = &e
	}

	result, err := h.service.GetMyRegistration(c.Request.Context(), tournamentID, &userID, email)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get registration status")
		response.InternalServerError(c, "Fehler beim Laden des Anmeldestatus")
		return
	}

	response.Success(c, http.StatusOK, result)
}

// --- Organizer Handlers ---

// GetRegistrations returns all registrations for a tournament
func (h *Handler) GetRegistrations(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Ungültige Turnier-ID")
		return
	}

	result, err := h.service.GetRegistrations(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get registrations")
		response.InternalServerError(c, "Fehler beim Laden der Anmeldungen")
		return
	}

	response.Success(c, http.StatusOK, result)
}

// GetPendingRegistrations returns registrations pending approval
func (h *Handler) GetPendingRegistrations(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Ungültige Turnier-ID")
		return
	}

	status := StatusPendingApproval
	guests, err := h.service.repo.ListGuestRegistrations(c.Request.Context(), tournamentID, &status)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get pending registrations")
		response.InternalServerError(c, "Fehler beim Laden der ausstehenden Anmeldungen")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"registrations": guests,
		"total":         len(guests),
	})
}

// ApproveRegistration approves or rejects a registration
func (h *Handler) ApproveRegistration(c *gin.Context) {
	regID, err := uuid.Parse(c.Param("regId"))
	if err != nil {
		response.BadRequest(c, "Ungültige Anmeldungs-ID")
		return
	}

	var req ApproveRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Ungültige Anfrage")
		return
	}

	userIDStr, _ := c.Get("user_id")
	approverID, _ := uuid.Parse(userIDStr.(string))

	if err := h.service.ApproveRegistration(c.Request.Context(), regID, &req, approverID); err != nil {
		h.handleRegistrationError(c, err)
		return
	}

	message := "Anmeldung bestätigt"
	if !req.Approved {
		message = "Anmeldung abgelehnt"
	}

	response.Success(c, http.StatusOK, gin.H{"message": message})
}

// ManualCheckIn performs manual check-in by organizer
func (h *Handler) ManualCheckIn(c *gin.Context) {
	regID, err := uuid.Parse(c.Param("regId"))
	if err != nil {
		response.BadRequest(c, "Ungültige Anmeldungs-ID")
		return
	}

	userIDStr, _ := c.Get("user_id")
	organizerID, _ := uuid.Parse(userIDStr.(string))

	// Try guest first
	if err := h.service.repo.SetGuestCheckIn(c.Request.Context(), regID, "organizer", &organizerID); err == nil {
		response.Success(c, http.StatusOK, gin.H{"message": "Eingecheckt"})
		return
	}

	// Try participant
	if err := h.service.repo.SetParticipantCheckIn(c.Request.Context(), regID, "organizer", &organizerID); err != nil {
		response.NotFound(c, "Anmeldung nicht gefunden")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Eingecheckt"})
}

// GetCheckInStatus returns check-in statistics
func (h *Handler) GetCheckInStatus(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Ungültige Turnier-ID")
		return
	}

	status, err := h.service.GetCheckInStatus(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get check-in status")
		response.InternalServerError(c, "Fehler beim Laden des Check-in Status")
		return
	}

	response.Success(c, http.StatusOK, status)
}

// GetRegistrationSettings returns registration settings
func (h *Handler) GetRegistrationSettings(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Ungültige Turnier-ID")
		return
	}

	settings, err := h.service.GetRegistrationSettings(c.Request.Context(), tournamentID)
	if err != nil {
		h.handleRegistrationError(c, err)
		return
	}

	response.Success(c, http.StatusOK, settings)
}

// UpdateRegistrationSettings updates registration settings
func (h *Handler) UpdateRegistrationSettings(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Ungültige Turnier-ID")
		return
	}

	var req UpdateRegistrationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Ungültige Anfrage")
		return
	}

	if err := h.service.UpdateRegistrationSettings(c.Request.Context(), tournamentID, &req); err != nil {
		h.handleRegistrationError(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Einstellungen aktualisiert"})
}

// CreateInviteToken creates a new invite token
func (h *Handler) CreateInviteToken(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Ungültige Turnier-ID")
		return
	}

	var req CreateInviteTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use defaults
		req = CreateInviteTokenRequest{}
	}

	userIDStr, _ := c.Get("user_id")
	creatorID, _ := uuid.Parse(userIDStr.(string))

	token, rawToken, err := h.service.CreateInviteToken(c.Request.Context(), tournamentID, &req, creatorID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create invite token")
		response.InternalServerError(c, "Fehler beim Erstellen des Einladungs-Tokens")
		return
	}

	response.Created(c, gin.H{
		"token": rawToken,
		"id":    token.ID,
		"name":  token.Name,
	})
}

// ListInviteTokens lists all invite tokens
func (h *Handler) ListInviteTokens(c *gin.Context) {
	tournamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Ungültige Turnier-ID")
		return
	}

	tokens, err := h.service.ListInviteTokens(c.Request.Context(), tournamentID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list invite tokens")
		response.InternalServerError(c, "Fehler beim Laden der Einladungs-Tokens")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"tokens": tokens})
}

// DeleteInviteToken deletes an invite token
func (h *Handler) DeleteInviteToken(c *gin.Context) {
	tokenID, err := uuid.Parse(c.Param("tokenId"))
	if err != nil {
		response.BadRequest(c, "Ungültige Token-ID")
		return
	}

	if err := h.service.DeleteInviteToken(c.Request.Context(), tokenID); err != nil {
		if err == ErrInvalidInviteToken {
			response.NotFound(c, "Token nicht gefunden")
			return
		}
		h.logger.Error().Err(err).Msg("Failed to delete invite token")
		response.InternalServerError(c, "Fehler beim Löschen des Tokens")
		return
	}

	response.NoContent(c)
}

// handleRegistrationError handles common registration errors
func (h *Handler) handleRegistrationError(c *gin.Context, err error) {
	switch err {
	case ErrRegistrationNotFound:
		response.NotFound(c, "Anmeldung nicht gefunden")
	case ErrAlreadyRegistered:
		response.Conflict(c, "Du bist bereits für dieses Turnier angemeldet")
	case ErrTournamentNotFound:
		response.NotFound(c, "Turnier nicht gefunden")
	case ErrInvalidInviteToken:
		response.BadRequest(c, "Ungültiger oder abgelaufener Einladungs-Token")
	case ErrRegistrationClosed:
		response.BadRequest(c, "Die Anmeldung ist geschlossen")
	case ErrTournamentFull:
		response.BadRequest(c, "Das Turnier ist voll")
	case ErrCheckInWindowClosed:
		response.BadRequest(c, "Das Check-in-Fenster ist geschlossen")
	case ErrGuestsNotAllowed:
		response.BadRequest(c, "Gastanmeldungen sind nicht erlaubt")
	case ErrPreRegistrationDisabled:
		response.BadRequest(c, "Die Voranmeldung ist deaktiviert")
	default:
		h.logger.Error().Err(err).Msg("Registration error")
		response.InternalServerError(c, "Ein Fehler ist aufgetreten")
	}
}
