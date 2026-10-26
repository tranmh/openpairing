package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/tranmh/openpairing/internal/config"
)

var (
	benchTokenSink  string
	benchClaimsSink *Claims
	benchErrSink    error
)

func benchJWTService() *JWTService {
	return NewJWTService(&config.JWTConfig{
		Secret:        "benchmark-access-secret-0123456789abcdef",
		RefreshSecret: "benchmark-refresh-secret-0123456789abcdef",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	})
}

func BenchmarkJWTService_GenerateAccessToken(b *testing.B) {
	s := benchJWTService()
	// Email length grows the claims payload that is encoded and signed.
	for _, n := range []int{16, 256, 4096} {
		email := strings.Repeat("a", n) + "@example.com"
		b.Run("email="+itoaBench(n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				tok, _, err := s.GenerateAccessToken("0b4c3a2e-1111-4222-8333-444455556666", email, "admin")
				if err != nil {
					b.Fatal(err)
				}
				benchTokenSink = tok
			}
		})
	}
}

func BenchmarkJWTService_ValidateAccessToken(b *testing.B) {
	s := benchJWTService()
	for _, n := range []int{16, 256, 4096} {
		tok, _, err := s.GenerateAccessToken("0b4c3a2e-1111-4222-8333-444455556666", strings.Repeat("a", n)+"@example.com", "user")
		if err != nil {
			b.Fatal(err)
		}
		b.Run("email="+itoaBench(n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchClaimsSink, benchErrSink = s.ValidateAccessToken(tok)
			}
		})
	}
}

func BenchmarkJWTService_ValidateRefreshToken(b *testing.B) {
	s := benchJWTService()
	refresh, _, err := s.GenerateRefreshToken("0b4c3a2e-1111-4222-8333-444455556666")
	if err != nil {
		b.Fatal(err)
	}
	access, _, err := s.GenerateAccessToken("0b4c3a2e-1111-4222-8333-444455556666", "a@example.com", "user")
	if err != nil {
		b.Fatal(err)
	}
	cases := []struct{ name, tok string }{
		{"valid", refresh},
		{"access_token", access},
		{"malformed", "not.a.jwt"},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchClaimsSink, benchErrSink = s.ValidateRefreshToken(c.tok)
			}
		})
	}
}

func itoaBench(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
