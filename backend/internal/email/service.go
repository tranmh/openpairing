package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Config holds email service configuration
type Config struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	UseTLS       bool
	SkipVerify   bool // For development only
}

// Service handles email sending
type Service struct {
	config    *Config
	logger    zerolog.Logger
	templates map[string]*template.Template
}

// NewService creates a new email service
func NewService(config *Config, logger zerolog.Logger) *Service {
	s := &Service{
		config:    config,
		logger:    logger.With().Str("component", "email_service").Logger(),
		templates: make(map[string]*template.Template),
	}

	// Register default templates
	s.registerDefaultTemplates()

	return s
}

// Email represents an email message
type Email struct {
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	Body        string
	HTMLBody    string
	Attachments []Attachment
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Send sends an email
func (s *Service) Send(ctx context.Context, email *Email) error {
	if len(email.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	// Build message
	msg := s.buildMessage(email)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	var client *smtp.Client
	var err error

	if s.config.UseTLS {
		// Use TLS connection
		tlsConfig := &tls.Config{
			ServerName:         s.config.SMTPHost,
			InsecureSkipVerify: s.config.SkipVerify,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, s.config.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		client, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
	}
	defer client.Close()

	// Authenticate if credentials provided
	if s.config.SMTPUsername != "" && s.config.SMTPPassword != "" {
		auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	allRecipients := append(email.To, email.CC...)
	allRecipients = append(allRecipients, email.BCC...)

	for _, recipient := range allRecipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to add recipient %s: %w", recipient, err)
		}
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start message: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close message: %w", err)
	}

	if err := client.Quit(); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to quit SMTP connection cleanly")
	}

	s.logger.Info().
		Strs("to", email.To).
		Str("subject", email.Subject).
		Msg("Email sent successfully")

	return nil
}

func (s *Service) buildMessage(email *Email) []byte {
	var buf bytes.Buffer

	// Headers
	buf.WriteString(fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.FromEmail))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))
	if len(email.CC) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(email.CC, ", ")))
	}
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	buf.WriteString("MIME-Version: 1.0\r\n")

	// Determine content type
	if email.HTMLBody != "" && email.Body != "" {
		// Multipart message
		boundary := "----=_Part_0_" + fmt.Sprintf("%d", time.Now().UnixNano())
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		buf.WriteString("\r\n")

		// Plain text part
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(email.Body)
		buf.WriteString("\r\n")

		// HTML part
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(email.HTMLBody)
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if email.HTMLBody != "" {
		// HTML only
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(email.HTMLBody)
	} else {
		// Plain text only
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(email.Body)
	}

	return buf.Bytes()
}

// Template-based email sending

// TemplateData holds data for email templates
type TemplateData struct {
	RecipientName string
	Data          map[string]interface{}
}

// SendTemplate sends an email using a template
func (s *Service) SendTemplate(ctx context.Context, to []string, templateName string, subject string, data *TemplateData) error {
	tmpl, ok := s.templates[templateName]
	if !ok {
		return fmt.Errorf("template %s not found", templateName)
	}

	var htmlBuf bytes.Buffer
	if err := tmpl.Execute(&htmlBuf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	email := &Email{
		To:       to,
		Subject:  subject,
		HTMLBody: htmlBuf.String(),
	}

	return s.Send(ctx, email)
}

// RegisterTemplate registers a custom email template
func (s *Service) RegisterTemplate(name string, tmpl *template.Template) {
	s.templates[name] = tmpl
}

func (s *Service) registerDefaultTemplates() {
	// Password reset template
	s.templates["password_reset"] = template.Must(template.New("password_reset").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #2563eb; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9fafb; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>OpenPairing</h1>
        </div>
        <div class="content">
            <h2>Password Reset Request</h2>
            <p>Hello {{.RecipientName}},</p>
            <p>We received a request to reset your password. Click the button below to create a new password:</p>
            <p><a href="{{.Data.ResetLink}}" class="button">Reset Password</a></p>
            <p>If you didn't request this, you can safely ignore this email.</p>
            <p>This link will expire in 24 hours.</p>
        </div>
        <div class="footer">
            <p>© OpenPairing - Chess Tournament Management</p>
        </div>
    </div>
</body>
</html>
`))

	// Pairing notification template
	s.templates["pairing_notification"] = template.Must(template.New("pairing_notification").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #2563eb; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9fafb; }
        .pairing-box { background: white; border: 1px solid #e5e7eb; border-radius: 8px; padding: 16px; margin: 16px 0; }
        .vs { text-align: center; font-weight: bold; color: #666; margin: 8px 0; }
        .player { font-size: 18px; }
        .details { color: #666; font-size: 14px; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Data.TournamentName}}</h1>
        </div>
        <div class="content">
            <h2>Round {{.Data.Round}} Pairing</h2>
            <p>Hello {{.RecipientName}},</p>
            <p>Your pairing for round {{.Data.Round}} has been published:</p>
            <div class="pairing-box">
                <div class="player">{{.Data.WhitePlayer}} ({{.Data.WhiteRating}})</div>
                <div class="vs">vs</div>
                <div class="player">{{.Data.BlackPlayer}} ({{.Data.BlackRating}})</div>
                <div class="details">
                    <p>Board: {{.Data.Board}}</p>
                    <p>Start Time: {{.Data.StartTime}}</p>
                    {{if .Data.OnlineGame}}<p>This is an online game</p>{{end}}
                </div>
            </div>
            <p><a href="{{.Data.TournamentLink}}" class="button">View Tournament</a></p>
        </div>
        <div class="footer">
            <p>© OpenPairing - Chess Tournament Management</p>
        </div>
    </div>
</body>
</html>
`))

	// Game starting notification template
	s.templates["game_starting"] = template.Must(template.New("game_starting").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #dc2626; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9fafb; }
        .button { display: inline-block; background: #dc2626; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Game Starting!</h1>
        </div>
        <div class="content">
            <h2>Your game is about to start</h2>
            <p>Hello {{.RecipientName}},</p>
            <p>Your game in <strong>{{.Data.TournamentName}}</strong> (Round {{.Data.Round}}) is starting now!</p>
            <p>Your opponent: <strong>{{.Data.OpponentName}}</strong></p>
            <p>You are playing as: <strong>{{.Data.Color}}</strong></p>
            <p><a href="{{.Data.GameLink}}" class="button">Join Game</a></p>
        </div>
        <div class="footer">
            <p>© OpenPairing - Chess Tournament Management</p>
        </div>
    </div>
</body>
</html>
`))

	// Tournament registration confirmation template
	s.templates["registration_confirmation"] = template.Must(template.New("registration_confirmation").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #16a34a; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9fafb; }
        .info-box { background: white; border: 1px solid #e5e7eb; border-radius: 8px; padding: 16px; margin: 16px 0; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Registration Confirmed</h1>
        </div>
        <div class="content">
            <h2>You're registered!</h2>
            <p>Hello {{.RecipientName}},</p>
            <p>Your registration for <strong>{{.Data.TournamentName}}</strong> has been confirmed.</p>
            <div class="info-box">
                <p><strong>Tournament:</strong> {{.Data.TournamentName}}</p>
                <p><strong>Dates:</strong> {{.Data.StartDate}} - {{.Data.EndDate}}</p>
                <p><strong>Location:</strong> {{.Data.Location}}</p>
                <p><strong>Time Control:</strong> {{.Data.TimeControl}}</p>
                <p><strong>Your Starting Rank:</strong> {{.Data.StartRank}}</p>
            </div>
            <p><a href="{{.Data.TournamentLink}}" class="button">View Tournament</a></p>
        </div>
        <div class="footer">
            <p>© OpenPairing - Chess Tournament Management</p>
        </div>
    </div>
</body>
</html>
`))

	// Team lineup deadline template
	s.templates["lineup_deadline"] = template.Must(template.New("lineup_deadline").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #f59e0b; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9fafb; }
        .warning { background: #fef3c7; border: 1px solid #f59e0b; border-radius: 8px; padding: 16px; margin: 16px 0; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Lineup Deadline Reminder</h1>
        </div>
        <div class="content">
            <h2>Submit your lineup soon!</h2>
            <p>Hello {{.RecipientName}},</p>
            <div class="warning">
                <p><strong>Deadline:</strong> {{.Data.Deadline}}</p>
                <p>The lineup submission deadline for Round {{.Data.Round}} of <strong>{{.Data.TournamentName}}</strong> is approaching.</p>
            </div>
            <p>Team: <strong>{{.Data.TeamName}}</strong></p>
            <p>Please submit your lineup before the deadline to avoid forfeits.</p>
            <p><a href="{{.Data.LineupLink}}" class="button">Submit Lineup</a></p>
        </div>
        <div class="footer">
            <p>© OpenPairing - Chess Tournament Management</p>
        </div>
    </div>
</body>
</html>
`))
}

// SendPasswordReset sends a password reset email
func (s *Service) SendPasswordReset(ctx context.Context, to, recipientName, resetLink string) error {
	return s.SendTemplate(ctx, []string{to}, "password_reset", "Reset Your Password - OpenPairing", &TemplateData{
		RecipientName: recipientName,
		Data: map[string]interface{}{
			"ResetLink": resetLink,
		},
	})
}

// SendPairingNotification sends a pairing notification email
func (s *Service) SendPairingNotification(ctx context.Context, to, recipientName string, data map[string]interface{}) error {
	return s.SendTemplate(ctx, []string{to}, "pairing_notification", fmt.Sprintf("Round %v Pairing - %v", data["Round"], data["TournamentName"]), &TemplateData{
		RecipientName: recipientName,
		Data:          data,
	})
}

// SendGameStarting sends a game starting notification email
func (s *Service) SendGameStarting(ctx context.Context, to, recipientName string, data map[string]interface{}) error {
	return s.SendTemplate(ctx, []string{to}, "game_starting", "Your Game is Starting!", &TemplateData{
		RecipientName: recipientName,
		Data:          data,
	})
}

// SendRegistrationConfirmation sends a registration confirmation email
func (s *Service) SendRegistrationConfirmation(ctx context.Context, to, recipientName string, data map[string]interface{}) error {
	return s.SendTemplate(ctx, []string{to}, "registration_confirmation", fmt.Sprintf("Registration Confirmed - %v", data["TournamentName"]), &TemplateData{
		RecipientName: recipientName,
		Data:          data,
	})
}

// SendLineupDeadlineReminder sends a lineup deadline reminder email
func (s *Service) SendLineupDeadlineReminder(ctx context.Context, to, recipientName string, data map[string]interface{}) error {
	return s.SendTemplate(ctx, []string{to}, "lineup_deadline", fmt.Sprintf("Lineup Deadline - Round %v", data["Round"]), &TemplateData{
		RecipientName: recipientName,
		Data:          data,
	})
}
