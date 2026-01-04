package validator

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"valid email", "test@example.com", true},
		{"valid email with subdomain", "test@mail.example.com", true},
		{"valid email with plus", "test+tag@example.com", true},
		{"valid email with dots", "test.name@example.com", true},
		{"empty string", "", false},
		{"missing @", "testexample.com", false},
		{"missing domain", "test@", false},
		{"missing local part", "@example.com", false},
		{"missing TLD", "test@example", false},
		{"spaces", "test @example.com", false},
		{"double @", "test@@example.com", false},
		{"too long", string(make([]byte, 256)) + "@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEmail(tt.email)
			if got != tt.want {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantOk   bool
	}{
		{"valid password", "Password123", true},
		{"valid with special chars", "P@ssword1!", true},
		{"too short", "Pass1", false},
		{"no uppercase", "password123", false},
		{"no lowercase", "PASSWORD123", false},
		{"no number", "PasswordABC", false},
		{"only lowercase", "password", false},
		{"only uppercase", "PASSWORD", false},
		{"only numbers", "12345678", false},
		{"minimum length with requirements", "Passwor1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, msg := ValidatePassword(tt.password)
			if gotOk != tt.wantOk {
				t.Errorf("ValidatePassword(%q) = (%v, %q), want ok=%v", tt.password, gotOk, msg, tt.wantOk)
			}
			if !gotOk && msg == "" {
				t.Error("Expected error message when validation fails")
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name string
		n    string
		want bool
	}{
		{"valid name", "John", true},
		{"valid name with space", " John ", true}, // Should trim
		{"valid long name", "Maximilian-Friedrich", true},
		{"empty", "", false},
		{"only spaces", "   ", false},
		{"too long", string(make([]byte, 101)), false},
		{"single char", "J", true},
		{"german characters", "Müller", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateName(tt.n)
			if got != tt.want {
				t.Errorf("ValidateName(%q) = %v, want %v", tt.n, got, tt.want)
			}
		})
	}
}

func TestValidateRole(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"player", "player", true},
		{"arbiter", "arbiter", true},
		{"organizer", "organizer", true},
		{"admin", "admin", true},
		{"invalid", "superuser", false},
		{"empty", "", false},
		{"uppercase", "PLAYER", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateRole(tt.role)
			if got != tt.want {
				t.Errorf("ValidateRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestValidateLanguage(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want bool
	}{
		{"german", "de", true},
		{"english", "en", true},
		{"french", "fr", false},
		{"empty", "", false},
		{"uppercase", "DE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateLanguage(tt.lang)
			if got != tt.want {
				t.Errorf("ValidateLanguage(%q) = %v, want %v", tt.lang, got, tt.want)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal string", "hello", "hello"},
		{"leading spaces", "  hello", "hello"},
		{"trailing spaces", "hello  ", "hello"},
		{"both sides", "  hello  ", "hello"},
		{"control characters", "hello\x00world", "helloworld"},
		{"newline", "hello\nworld", "helloworld"},
		{"tab", "hello\tworld", "helloworld"},
		{"mixed", "  hello\x00\nworld  ", "helloworld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeString(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidationErrors(t *testing.T) {
	t.Run("Add and HasErrors", func(t *testing.T) {
		errs := ValidationErrors{}

		if errs.HasErrors() {
			t.Error("Expected no errors initially")
		}

		errs.Add("email", "Invalid email")

		if !errs.HasErrors() {
			t.Error("Expected errors after Add")
		}

		if errs["email"] != "Invalid email" {
			t.Errorf("Expected 'Invalid email', got %q", errs["email"])
		}
	})

	t.Run("Multiple errors", func(t *testing.T) {
		errs := ValidationErrors{}
		errs.Add("email", "Invalid email")
		errs.Add("password", "Too short")

		if len(errs) != 2 {
			t.Errorf("Expected 2 errors, got %d", len(errs))
		}
	})
}
