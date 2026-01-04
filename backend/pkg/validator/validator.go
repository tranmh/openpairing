package validator

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// ValidationErrors holds a map of field to error message
type ValidationErrors map[string]string

// Add adds an error for a field
func (v ValidationErrors) Add(field, message string) {
	v[field] = message
}

// HasErrors returns true if there are validation errors
func (v ValidationErrors) HasErrors() bool {
	return len(v) > 0
}

// ValidateEmail validates an email address
func ValidateEmail(email string) bool {
	if len(email) > 255 {
		return false
	}
	return emailRegex.MatchString(email)
}

// ValidatePassword validates a password
// Requirements: min 8 characters, at least one uppercase, one lowercase, one number
func ValidatePassword(password string) (bool, string) {
	if len(password) < 8 {
		return false, "Password must be at least 8 characters long"
	}

	var hasUpper, hasLower, hasNumber bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}

	if !hasUpper {
		return false, "Password must contain at least one uppercase letter"
	}
	if !hasLower {
		return false, "Password must contain at least one lowercase letter"
	}
	if !hasNumber {
		return false, "Password must contain at least one number"
	}

	return true, ""
}

// ValidateName validates a name (first or last name)
func ValidateName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 100 {
		return false
	}
	return true
}

// ValidateRole validates a user role
func ValidateRole(role string) bool {
	validRoles := map[string]bool{
		"player":    true,
		"arbiter":   true,
		"organizer": true,
		"admin":     true,
	}
	return validRoles[role]
}

// ValidateLanguage validates a language code
func ValidateLanguage(lang string) bool {
	validLanguages := map[string]bool{
		"de": true,
		"en": true,
	}
	return validLanguages[lang]
}

// SanitizeString trims whitespace and removes control characters
func SanitizeString(s string) string {
	s = strings.TrimSpace(s)
	// Remove control characters
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}
