package email

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

// EmailConfig holds SMTP configuration
type EmailConfig struct {
	Host     string
	Port     string
	From     string
	FromName string
	Username string
	Password string
}

// LoadConfigFromEnv loads email configuration from environment variables
func LoadConfigFromEnv() *EmailConfig {
	return &EmailConfig{
		Host:     getEnv("SMTP_HOST", "mailhog"),
		Port:     getEnv("SMTP_PORT", "1025"),
		From:     getEnv("SMTP_FROM", "noreply@nuclear-ao3.local"),
		FromName: getEnv("SMTP_FROM_NAME", "Nuclear AO3"),
		Username: getEnv("SMTP_USERNAME", ""),
		Password: getEnv("SMTP_PASSWORD", ""),
	}
}

// SendEmail sends a simple text email
func SendEmail(to, subject, body string) error {
	config := LoadConfigFromEnv()
	return SendEmailWithConfig(config, to, subject, body)
}

// SendEmailWithConfig sends an email using the provided configuration
func SendEmailWithConfig(config *EmailConfig, to, subject, body string) error {
	// Build the email message
	from := config.From
	if config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", config.FromName, config.From)
	}

	message := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"\r\n"+
			"%s",
		from, to, subject, body,
	))

	// Setup SMTP server address
	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)

	// For MailHog (development), no auth needed
	// For production SMTP, use authentication
	var auth smtp.Auth
	if config.Username != "" && config.Password != "" {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	}

	// Send the email
	return smtp.SendMail(addr, auth, config.From, []string{to}, message)
}

// SendHTMLEmail sends an HTML email
func SendHTMLEmail(to, subject, htmlBody string) error {
	config := LoadConfigFromEnv()
	return SendHTMLEmailWithConfig(config, to, subject, htmlBody)
}

// SendHTMLEmailWithConfig sends an HTML email using the provided configuration
func SendHTMLEmailWithConfig(config *EmailConfig, to, subject, htmlBody string) error {
	from := config.From
	if config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", config.FromName, config.From)
	}

	message := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s",
		from, to, subject, htmlBody,
	))

	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)

	var auth smtp.Auth
	if config.Username != "" && config.Password != "" {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	}

	return smtp.SendMail(addr, auth, config.From, []string{to}, message)
}

// Helper function to get environment variable with default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// WelcomeEmailTemplate generates a welcome email
func WelcomeEmailTemplate(username string) (subject, body string) {
	subject = "Welcome to Nuclear AO3!"
	body = fmt.Sprintf(`Hello %s!

Welcome to Nuclear AO3, the fast and modern fanfiction archive!

Your account has been successfully created. You can now:
- Browse and search thousands of works
- Post your own fanfiction
- Bookmark your favorite stories
- Subscribe to authors and series
- Engage with the community through comments and kudos

Get started by visiting your dashboard and exploring what our community has to offer!

Happy reading and writing!

- The Nuclear AO3 Team

---
This is an automated message. Please do not reply to this email.`, username)
	return
}

// WelcomeEmailHTMLTemplate generates a welcome email in HTML format
func WelcomeEmailHTMLTemplate(username string) (subject, htmlBody string) {
	subject = "Welcome to Nuclear AO3!"
	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #ea580c 0%%, #fb923c 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #ffffff; padding: 30px; border: 1px solid #e5e7eb; border-radius: 0 0 8px 8px; }
        .button { display: inline-block; padding: 12px 24px; background: #ea580c; color: white; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 30px; color: #6b7280; font-size: 12px; }
        ul { padding-left: 20px; }
        li { margin: 8px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1 style="margin: 0; font-size: 28px;">🚀 Welcome to Nuclear AO3!</h1>
        </div>
        <div class="content">
            <h2>Hello %s!</h2>
            <p>We're excited to have you join our community! Your account has been successfully created.</p>
            
            <h3>What you can do now:</h3>
            <ul>
                <li>📚 <strong>Browse and search</strong> thousands of fanfiction works</li>
                <li>✍️ <strong>Post your own</strong> stories and share your creativity</li>
                <li>🔖 <strong>Bookmark</strong> your favorite works for easy access</li>
                <li>🔔 <strong>Subscribe</strong> to authors and series you love</li>
                <li>💬 <strong>Engage</strong> with the community through comments and kudos</li>
            </ul>
            
            <p style="margin-top: 30px;">Ready to get started?</p>
            <center>
                <a href="http://localhost:3000/dashboard" class="button">Visit Your Dashboard</a>
            </center>
        </div>
        <div class="footer">
            <p>This is an automated message from Nuclear AO3. Please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>`, strings.TrimSpace(username))
	return
}
