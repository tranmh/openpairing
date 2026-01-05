package portal64

import "errors"

var (
	// ErrPlayerNotFound is returned when a player cannot be found
	ErrPlayerNotFound = errors.New("player not found")

	// ErrClubNotFound is returned when a club cannot be found
	ErrClubNotFound = errors.New("club not found")

	// ErrAPIUnavailable is returned when the portal64 API is unreachable
	ErrAPIUnavailable = errors.New("portal64 API unavailable")

	// ErrInvalidResponse is returned when the API response cannot be parsed
	ErrInvalidResponse = errors.New("invalid API response")

	// ErrCacheError is returned when there's a cache operation failure
	ErrCacheError = errors.New("cache operation failed")
)
