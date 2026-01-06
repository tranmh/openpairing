package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tranmh/openpairing/pkg/response"
)

// Middleware provides authentication middleware
type Middleware struct {
	jwt *JWTService
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(jwt *JWTService) *Middleware {
	return &Middleware{jwt: jwt}
}

// Authenticate validates the access token and sets user info in context
func (m *Middleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			response.Unauthorized(c, "No authorization token provided")
			c.Abort()
			return
		}

		claims, err := m.jwt.ValidateAccessToken(token)
		if err != nil {
			switch err {
			case ErrExpiredToken:
				response.Unauthorized(c, "Token has expired")
			default:
				response.Unauthorized(c, "Invalid token")
			}
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// OptionalAuth tries to authenticate but doesn't fail if no token is provided
func (m *Middleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.Next()
			return
		}

		claims, err := m.jwt.ValidateAccessToken(token)
		if err == nil {
			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("user_role", claims.Role)
		}

		c.Next()
	}
}

// RequireRole checks if the user has one of the required roles
func (m *Middleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			response.Unauthorized(c, "Not authenticated")
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok {
			response.InternalServerError(c, "Invalid user role in context")
			c.Abort()
			return
		}

		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "Insufficient permissions")
		c.Abort()
	}
}

// RequireAdmin checks if the user is an admin
func (m *Middleware) RequireAdmin() gin.HandlerFunc {
	return m.RequireRole("admin")
}

// RequireArbiter checks if the user is an arbiter, organizer, or admin
func (m *Middleware) RequireArbiter() gin.HandlerFunc {
	return m.RequireRole("arbiter", "organizer", "admin")
}

// RequireOrganizer checks if the user is an organizer or admin
func (m *Middleware) RequireOrganizer() gin.HandlerFunc {
	return m.RequireRole("organizer", "admin")
}

// extractToken extracts the JWT token from the Authorization header
func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	// Check for "Bearer " prefix
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return parts[1]
}

// GetUserID gets the user ID from the context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	id, ok := userID.(string)
	if !ok {
		return "", false
	}
	return id, true
}

// GetUserRole gets the user role from the context
func GetUserRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("user_role")
	if !exists {
		return "", false
	}
	r, ok := role.(string)
	if !ok {
		return "", false
	}
	return r, true
}

// IsAuthenticated checks if the user is authenticated
func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get("user_id")
	return exists
}

// IsAdmin checks if the user is an admin
func IsAdmin(c *gin.Context) bool {
	role, exists := GetUserRole(c)
	return exists && role == "admin"
}

// IsArbiter checks if the user is an arbiter or higher
func IsArbiter(c *gin.Context) bool {
	role, exists := GetUserRole(c)
	return exists && (role == "arbiter" || role == "organizer" || role == "admin")
}
