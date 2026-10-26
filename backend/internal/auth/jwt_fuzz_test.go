package auth

import (
	"testing"
	"time"
	"unicode/utf8"

	"github.com/tranmh/openpairing/internal/config"
)

// FuzzJWTRoundTrip: an access token validates to its own claims, a refresh
// token is never accepted as an access token, and arbitrary strings never
// validate.
func FuzzJWTRoundTrip(f *testing.F) {
	f.Add("user-1", "a@b.de", "admin", "eyJhbGciOiJub25lIn0.e30.")
	f.Add("", "", "", "")
	f.Add("\xff", "\x00", "player", "a.b.c")
	svc := NewJWTService(&config.JWTConfig{
		Secret:        "access-secret",
		RefreshSecret: "access-secret", // same secret: only the type claim separates the tokens
		AccessExpiry:  time.Hour,
		RefreshExpiry: time.Hour,
		Issuer:        "fuzz",
	})
	f.Fuzz(func(t *testing.T, userID, email, role, raw string) {
		if c, err := svc.ValidateAccessToken(raw); err == nil {
			t.Fatalf("ValidateAccessToken(%q) accepted an unsigned input: %+v", raw, c)
		}
		if !utf8.ValidString(userID) || !utf8.ValidString(email) || !utf8.ValidString(role) {
			return // JSON claims carry text only
		}
		tok, _, err := svc.GenerateAccessToken(userID, email, role)
		if err != nil {
			t.Fatalf("GenerateAccessToken(%q, %q, %q): %v", userID, email, role, err)
		}
		c, err := svc.ValidateAccessToken(tok)
		if err != nil {
			t.Fatalf("ValidateAccessToken of a fresh token for (%q, %q, %q): %v", userID, email, role, err)
		}
		if c.UserID != userID || c.Email != email || c.Role != role || c.Type != AccessToken {
			t.Errorf("round trip of (%q, %q, %q) gave %+v", userID, email, role, c)
		}
		ref, _, err := svc.GenerateRefreshToken(userID)
		if err != nil {
			t.Fatalf("GenerateRefreshToken(%q): %v", userID, err)
		}
		if _, err := svc.ValidateAccessToken(ref); err == nil {
			t.Errorf("refresh token for %q accepted as access token", userID)
		}
	})
}
