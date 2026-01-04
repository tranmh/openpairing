package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Error represents an API error response
type Error struct {
	Message string            `json:"message"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// Success sends a successful JSON response
func Success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

// Created sends a 201 Created response
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Err sends an error JSON response
func Err(c *gin.Context, status int, message string, code string) {
	c.JSON(status, gin.H{
		"error": Error{
			Message: message,
			Code:    code,
		},
	})
}

// ErrWithDetails sends an error JSON response with validation details
func ErrWithDetails(c *gin.Context, status int, message string, code string, details map[string]string) {
	c.JSON(status, gin.H{
		"error": Error{
			Message: message,
			Code:    code,
			Details: details,
		},
	})
}

// BadRequest sends a 400 Bad Request response
func BadRequest(c *gin.Context, message string) {
	Err(c, http.StatusBadRequest, message, "BAD_REQUEST")
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(c *gin.Context, message string) {
	Err(c, http.StatusUnauthorized, message, "UNAUTHORIZED")
}

// Forbidden sends a 403 Forbidden response
func Forbidden(c *gin.Context, message string) {
	Err(c, http.StatusForbidden, message, "FORBIDDEN")
}

// NotFound sends a 404 Not Found response
func NotFound(c *gin.Context, message string) {
	Err(c, http.StatusNotFound, message, "NOT_FOUND")
}

// Conflict sends a 409 Conflict response
func Conflict(c *gin.Context, message string) {
	Err(c, http.StatusConflict, message, "CONFLICT")
}

// UnprocessableEntity sends a 422 Unprocessable Entity response
func UnprocessableEntity(c *gin.Context, message string, details map[string]string) {
	ErrWithDetails(c, http.StatusUnprocessableEntity, message, "VALIDATION_ERROR", details)
}

// InternalServerError sends a 500 Internal Server Error response
func InternalServerError(c *gin.Context, message string) {
	Err(c, http.StatusInternalServerError, message, "INTERNAL_ERROR")
}

// ServiceUnavailable sends a 503 Service Unavailable response
func ServiceUnavailable(c *gin.Context, message string) {
	Err(c, http.StatusServiceUnavailable, message, "SERVICE_UNAVAILABLE")
}
