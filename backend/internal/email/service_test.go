package email

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func newTestService() *Service {
	config := &Config{
		SMTPHost:  "localhost",
		SMTPPort:  1025,
		FromEmail: "test@openpairing.com",
		FromName:  "OpenPairing",
	}
	logger := zerolog.Nop()
	return NewService(config, logger)
}

func TestNewService(t *testing.T) {
	s := newTestService()
	assert.NotNil(t, s)
	assert.NotNil(t, s.config)
	assert.NotNil(t, s.templates)
}

func TestBuildMessage_PlainText(t *testing.T) {
	s := newTestService()

	email := &Email{
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "This is the message body.",
	}

	msg := s.buildMessage(email)
	msgStr := string(msg)

	assert.Contains(t, msgStr, "From: OpenPairing <test@openpairing.com>")
	assert.Contains(t, msgStr, "To: recipient@example.com")
	assert.Contains(t, msgStr, "Subject: Test Subject")
	assert.Contains(t, msgStr, "Content-Type: text/plain")
	assert.Contains(t, msgStr, "This is the message body.")
}

func TestBuildMessage_HTMLOnly(t *testing.T) {
	s := newTestService()

	email := &Email{
		To:       []string{"recipient@example.com"},
		Subject:  "Test Subject",
		HTMLBody: "<html><body><h1>Hello</h1></body></html>",
	}

	msg := s.buildMessage(email)
	msgStr := string(msg)

	assert.Contains(t, msgStr, "Content-Type: text/html")
	assert.Contains(t, msgStr, "<html><body><h1>Hello</h1></body></html>")
}

func TestBuildMessage_MultipartAlternative(t *testing.T) {
	s := newTestService()

	email := &Email{
		To:       []string{"recipient@example.com"},
		Subject:  "Test Subject",
		Body:     "Plain text version",
		HTMLBody: "<html><body><h1>HTML version</h1></body></html>",
	}

	msg := s.buildMessage(email)
	msgStr := string(msg)

	assert.Contains(t, msgStr, "Content-Type: multipart/alternative")
	assert.Contains(t, msgStr, "Plain text version")
	assert.Contains(t, msgStr, "<html><body><h1>HTML version</h1></body></html>")
}

func TestBuildMessage_MultipleRecipients(t *testing.T) {
	s := newTestService()

	email := &Email{
		To:      []string{"a@example.com", "b@example.com"},
		CC:      []string{"c@example.com"},
		Subject: "Test",
		Body:    "Test body",
	}

	msg := s.buildMessage(email)
	msgStr := string(msg)

	assert.Contains(t, msgStr, "To: a@example.com, b@example.com")
	assert.Contains(t, msgStr, "Cc: c@example.com")
}

func TestTemplates_Registered(t *testing.T) {
	s := newTestService()

	expectedTemplates := []string{
		"password_reset",
		"pairing_notification",
		"game_starting",
		"registration_confirmation",
		"lineup_deadline",
	}

	for _, name := range expectedTemplates {
		_, ok := s.templates[name]
		assert.True(t, ok, "Template %s should be registered", name)
	}
}

func TestTemplates_Execute(t *testing.T) {
	s := newTestService()

	tests := []struct {
		name     string
		template string
		data     *TemplateData
		contains []string
	}{
		{
			name:     "password_reset",
			template: "password_reset",
			data: &TemplateData{
				RecipientName: "John",
				Data: map[string]interface{}{
					"ResetLink": "https://example.com/reset/abc123",
				},
			},
			contains: []string{"John", "https://example.com/reset/abc123", "Reset Password"},
		},
		{
			name:     "pairing_notification",
			template: "pairing_notification",
			data: &TemplateData{
				RecipientName: "Alice",
				Data: map[string]interface{}{
					"TournamentName": "City Championship",
					"Round":          3,
					"WhitePlayer":    "Alice Smith",
					"WhiteRating":    1850,
					"BlackPlayer":    "Bob Jones",
					"BlackRating":    1920,
					"Board":          1,
					"StartTime":      "14:00",
					"OnlineGame":     true,
					"TournamentLink": "https://example.com/tournament/123",
				},
			},
			contains: []string{"Alice", "City Championship", "Round 3", "Board: 1", "online game"},
		},
		{
			name:     "game_starting",
			template: "game_starting",
			data: &TemplateData{
				RecipientName: "Bob",
				Data: map[string]interface{}{
					"TournamentName": "Open Tournament",
					"Round":          1,
					"OpponentName":   "Charlie",
					"Color":          "White",
					"GameLink":       "https://example.com/game/456",
				},
			},
			contains: []string{"Bob", "Open Tournament", "Charlie", "White"},
		},
		{
			name:     "registration_confirmation",
			template: "registration_confirmation",
			data: &TemplateData{
				RecipientName: "David",
				Data: map[string]interface{}{
					"TournamentName": "Summer Open",
					"StartDate":      "2025-07-01",
					"EndDate":        "2025-07-05",
					"Location":       "Chess Club Berlin",
					"TimeControl":    "90+30",
					"StartRank":      15,
					"TournamentLink": "https://example.com/tournament/789",
				},
			},
			// Note: "+" becomes "&#43;" in HTML escaping
			contains: []string{"David", "Summer Open", "2025-07-01", "Chess Club Berlin", "90"},
		},
		{
			name:     "lineup_deadline",
			template: "lineup_deadline",
			data: &TemplateData{
				RecipientName: "Eve",
				Data: map[string]interface{}{
					"TournamentName": "Bundesliga Match",
					"TeamName":       "Chess Club Munich",
					"Round":          5,
					"Deadline":       "Saturday 18:00",
					"LineupLink":     "https://example.com/lineup/submit",
				},
			},
			contains: []string{"Eve", "Bundesliga Match", "Chess Club Munich", "Saturday 18:00"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, ok := s.templates[tt.template]
			assert.True(t, ok, "Template %s not found", tt.template)

			var buf bytes.Buffer
			err := tmpl.Execute(&buf, tt.data)
			assert.NoError(t, err)

			result := buf.String()
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected, "Template output should contain %s", expected)
			}
		})
	}
}

func TestSend_NoRecipients(t *testing.T) {
	s := newTestService()

	email := &Email{
		Subject: "Test",
		Body:    "Test body",
	}

	err := s.Send(nil, email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no recipients")
}

func TestSendTemplate_NotFound(t *testing.T) {
	s := newTestService()

	err := s.SendTemplate(nil, []string{"test@example.com"}, "nonexistent_template", "Subject", &TemplateData{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template nonexistent_template not found")
}
