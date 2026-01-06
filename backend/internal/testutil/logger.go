package testutil

import (
	"io"

	"github.com/rs/zerolog"
)

// NewTestLogger returns a no-op logger suitable for tests.
// It discards all log output to keep test output clean.
func NewTestLogger() zerolog.Logger {
	return zerolog.Nop()
}

// NewTestLoggerWithOutput returns a logger that writes to the given writer.
// Useful when you want to capture or inspect log output in tests.
func NewTestLoggerWithOutput(w io.Writer) zerolog.Logger {
	return zerolog.New(w).With().Timestamp().Logger()
}
