package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openpairing/openpairing/pkg/response"
)

// RateLimiter implements a simple in-memory rate limiter
// For production, use Redis-based rate limiting
type RateLimiter struct {
	mu        sync.RWMutex
	clients   map[string]*clientInfo
	limit     int
	window    time.Duration
	cleanupAt time.Time
}

type clientInfo struct {
	count    int
	windowAt time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		clients:   make(map[string]*clientInfo),
		limit:     limit,
		window:    window,
		cleanupAt: time.Now().Add(window),
	}
}

// Allow checks if the request should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Cleanup old entries periodically
	if now.After(rl.cleanupAt) {
		for k, v := range rl.clients {
			if now.Sub(v.windowAt) > rl.window {
				delete(rl.clients, k)
			}
		}
		rl.cleanupAt = now.Add(rl.window)
	}

	client, exists := rl.clients[key]
	if !exists || now.Sub(client.windowAt) > rl.window {
		rl.clients[key] = &clientInfo{
			count:    1,
			windowAt: now,
		}
		return true
	}

	if client.count >= rl.limit {
		return false
	}

	client.count++
	return true
}

// RateLimit returns a gin middleware for rate limiting
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(limit, window)

	return func(c *gin.Context) {
		key := c.ClientIP()

		if !limiter.Allow(key) {
			response.Err(c, http.StatusTooManyRequests, "Too many requests. Please try again later.", "RATE_LIMIT_EXCEEDED")
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuthRateLimit returns a stricter rate limiter for auth endpoints
func AuthRateLimit() gin.HandlerFunc {
	// 5 requests per minute for auth endpoints
	return RateLimit(5, time.Minute)
}

// APIRateLimit returns a standard rate limiter for API endpoints
func APIRateLimit() gin.HandlerFunc {
	// 100 requests per minute for general API
	return RateLimit(100, time.Minute)
}
