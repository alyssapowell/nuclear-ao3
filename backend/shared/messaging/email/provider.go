package email

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"nuclear-ao3/shared/messaging/errors"
	"nuclear-ao3/shared/messaging/telemetry"
	"nuclear-ao3/shared/messaging/templates"
	"nuclear-ao3/shared/models"
)

var (
	// emailRegex validates email addresses
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// EmailChannelProvider implements the ChannelProvider interface for email delivery
type EmailChannelProvider struct {
	config     *SMTPConfig
	telemetry  *telemetry.InMemoryTelemetryCollector
	templates  templates.TemplateRenderer
	classifier *errors.SMTPErrorClassifier
}

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Username     string        `json:"username,omitempty"`
	Password     string        `json:"password,omitempty"`
	UseTLS       bool          `json:"use_tls"`
	UseStartTLS  bool          `json:"use_starttls"`
	SkipVerify   bool          `json:"skip_verify"`
	Timeout      time.Duration `json:"timeout"`
	FromEmail    string        `json:"from_email"`
	FromName     string        `json:"from_name"`
	ReplyToEmail string        `json:"reply_to_email,omitempty"`
	ReturnPath   string        `json:"return_path,omitempty"`
	MaxRetries   int           `json:"max_retries"`
	RetryDelay   time.Duration `json:"retry_delay"`
}

// SMTPResponse contains detailed SMTP response information
type SMTPResponse struct {
	Code         int               `json:"code"`
	EnhancedCode string            `json:"enhanced_code,omitempty"`
	Message      string            `json:"message"`
	Headers      map[string]string `json:"headers,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
	ServerName   string            `json:"server_name,omitempty"`
	Duration     time.Duration     `json:"duration"`
}

// NewEmailChannelProvider creates a new email channel provider
func NewEmailChannelProvider(config *SMTPConfig, telemetry *telemetry.InMemoryTelemetryCollector, templates templates.TemplateRenderer, classifier *errors.SMTPErrorClassifier) *EmailChannelProvider {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Minute
	}

	return &EmailChannelProvider{
		config:     config,
		telemetry:  telemetry,
		templates:  templates,
		classifier: classifier,
	}
}

// GetChannelType returns the channel type
func (e *EmailChannelProvider) GetChannelType() models.DeliveryChannel {
	return models.ChannelEmail
}

// DeliverMessage delivers a message via email
func (e *EmailChannelProvider) DeliverMessage(ctx context.Context, msg *models.Message, recipient *models.Recipient) (*models.DeliveryAttempt, error) {
	startTime := time.Now()
	attemptID := uuid.New()

	// Create delivery attempt record
	attempt := &models.DeliveryAttempt{
		ID:          attemptID,
		MessageID:   msg.ID,
		UserID:      recipient.UserID,
		Channel:     models.ChannelEmail,
		Status:      models.DeliveryStatusPending,
		AttemptedAt: startTime,
		Metadata:    make(map[string]interface{}),
		RetryCount:  0,
	}

	// Get recipient email address
	channelConfig, exists := recipient.Preferences.Channels[models.ChannelEmail]
	if !exists || !channelConfig.Enabled {
		attempt.Status = models.DeliveryStatusFailed
		attempt.Error = &models.DeliveryError{
			Type:      "configuration_error",
			Message:   "Email channel not enabled for user",
			Retryable: false,
		}
		e.telemetry.RecordDeliveryAttempt(attempt)
		return attempt, fmt.Errorf("email channel not enabled for user")
	}

	emailAddress := channelConfig.Address
	if err := e.ValidateAddress(emailAddress); err != nil {
		attempt.Status = models.DeliveryStatusFailed
		attempt.Error = &models.DeliveryError{
			Type:      "invalid_address",
			Message:   fmt.Sprintf("Invalid email address: %v", err),
			Retryable: false,
		}
		e.telemetry.RecordDeliveryAttempt(attempt)
		return attempt, fmt.Errorf("invalid email address: %w", err)
	}

	// Render email template
	renderedEmail, err := e.templates.RenderEmailTemplate(msg.Type, &msg.Content)
	if err != nil {
		attempt.Status = models.DeliveryStatusFailed
		attempt.Error = &models.DeliveryError{
			Type:      "template_error",
			Message:   fmt.Sprintf("Failed to render email template: %v", err),
			Retryable: false,
		}
		e.telemetry.RecordDeliveryAttempt(attempt)
		return attempt, fmt.Errorf("failed to render email template: %w", err)
	}

	// Send email with full telemetry
	smtpResponse, err := e.sendEmailWithTelemetry(ctx, emailAddress, renderedEmail, attempt)

	// Calculate total duration
	duration := time.Since(startTime)
	e.telemetry.RecordLatency(models.ChannelEmail, duration)

	// Update attempt with results
	if err != nil {
		attempt.Status = models.DeliveryStatusFailed
		if smtpResponse != nil {
			attempt.Error = e.classifier.ClassifySMTPError(smtpResponse.Code, smtpResponse.Message)
		} else {
			attempt.Error = &models.DeliveryError{
				Type:      "network_error",
				Message:   err.Error(),
				Retryable: true,
			}
		}
		e.telemetry.RecordError(models.ChannelEmail, attempt.Error.Type, err)
	} else {
		attempt.Status = models.DeliveryStatusSent
		if smtpResponse != nil && smtpResponse.Code >= 200 && smtpResponse.Code < 300 {
			attempt.Status = models.DeliveryStatusDelivered
		}
	}

	// Store SMTP response in metadata
	if smtpResponse != nil {
		smtpData, _ := json.Marshal(smtpResponse)
		attempt.Metadata["smtp_response"] = string(smtpData)
	}

	attempt.Metadata["email_address"] = emailAddress
	attempt.Metadata["duration_ms"] = duration.Milliseconds()

	e.telemetry.RecordDeliveryAttempt(attempt)
	return attempt, err
}

// sendEmailWithTelemetry sends an email with comprehensive telemetry
func (e *EmailChannelProvider) sendEmailWithTelemetry(ctx context.Context, to string, email *templates.RenderedEmail, attempt *models.DeliveryAttempt) (*SMTPResponse, error) {
	startTime := time.Now()

	// Increment attempt counter
	e.telemetry.IncrementCounter("email_delivery_attempts", map[string]string{
		"smtp_host": e.config.Host,
	})

	// Build email message
	message, err := e.buildEmailMessage(to, email)
	if err != nil {
		return nil, fmt.Errorf("failed to build email message: %w", err)
	}

	// Connect to SMTP server with timeout
	conn, err := e.connectSMTP(ctx)
	if err != nil {
		e.telemetry.RecordError(models.ChannelEmail, "smtp_connection_error", err)
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	smtpClient, err := smtp.NewClient(conn, e.config.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer smtpClient.Quit()

	// Start TLS if configured
	if e.config.UseStartTLS {
		tlsConfig := &tls.Config{
			ServerName:         e.config.Host,
			InsecureSkipVerify: e.config.SkipVerify,
		}
		if err = smtpClient.StartTLS(tlsConfig); err != nil {
			return nil, fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate if credentials provided
	if e.config.Username != "" && e.config.Password != "" {
		auth := smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.Host)
		if err = smtpClient.Auth(auth); err != nil {
			e.telemetry.RecordError(models.ChannelEmail, "smtp_auth_error", err)
			return nil, fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	fromAddr := e.config.FromEmail
	if fromAddr == "" {
		fromAddr = e.config.Username
	}
	if err = smtpClient.Mail(fromAddr); err != nil {
		response := e.parseSMTPError(err)
		e.telemetry.RecordError(models.ChannelEmail, "smtp_mail_error", err)
		return response, fmt.Errorf("SMTP MAIL command failed: %w", err)
	}

	// Set recipient
	if err = smtpClient.Rcpt(to); err != nil {
		response := e.parseSMTPError(err)
		e.telemetry.RecordError(models.ChannelEmail, "smtp_rcpt_error", err)
		return response, fmt.Errorf("SMTP RCPT command failed: %w", err)
	}

	// Send message data
	writer, err := smtpClient.Data()
	if err != nil {
		response := e.parseSMTPError(err)
		e.telemetry.RecordError(models.ChannelEmail, "smtp_data_error", err)
		return response, fmt.Errorf("SMTP DATA command failed: %w", err)
	}

	_, err = writer.Write([]byte(message))
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to write email data: %w", err)
	}

	err = writer.Close()
	if err != nil {
		response := e.parseSMTPError(err)
		e.telemetry.RecordError(models.ChannelEmail, "smtp_send_error", err)
		return response, fmt.Errorf("failed to close email data: %w", err)
	}

	// Record successful send
	duration := time.Since(startTime)
	e.telemetry.IncrementCounter("email_delivery_success", map[string]string{
		"smtp_host": e.config.Host,
	})

	response := &SMTPResponse{
		Code:      250,
		Message:   "Message sent successfully",
		Timestamp: time.Now(),
		Duration:  duration,
	}

	return response, nil
}

// connectSMTP establishes connection to SMTP server
func (e *EmailChannelProvider) connectSMTP(ctx context.Context) (net.Conn, error) {
	address := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)

	dialer := &net.Dialer{
		Timeout: e.config.Timeout,
	}

	if e.config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName:         e.config.Host,
			InsecureSkipVerify: e.config.SkipVerify,
		}
		return tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	}

	return dialer.DialContext(ctx, "tcp", address)
}

// buildEmailMessage constructs the full email message
func (e *EmailChannelProvider) buildEmailMessage(to string, email *templates.RenderedEmail) (string, error) {
	var message strings.Builder

	// Headers
	message.WriteString(fmt.Sprintf("From: %s <%s>\r\n", e.config.FromName, e.config.FromEmail))
	message.WriteString(fmt.Sprintf("To: %s\r\n", to))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))

	if e.config.ReplyToEmail != "" {
		message.WriteString(fmt.Sprintf("Reply-To: %s\r\n", e.config.ReplyToEmail))
	}

	if e.config.ReturnPath != "" {
		message.WriteString(fmt.Sprintf("Return-Path: %s\r\n", e.config.ReturnPath))
	}

	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	message.WriteString(fmt.Sprintf("Message-ID: <%s@%s>\r\n", uuid.New().String(), e.config.Host))

	// Custom headers
	for key, value := range email.Headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// Content-Type
	if email.HTML != "" {
		// Multipart message with both plain text and HTML
		boundary := fmt.Sprintf("boundary_%s", uuid.New().String())
		message.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary))
		message.WriteString("\r\n")

		// Plain text part
		message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		message.WriteString("Content-Transfer-Encoding: 8bit\r\n")
		message.WriteString("\r\n")
		message.WriteString(email.PlainText)
		message.WriteString("\r\n\r\n")

		// HTML part
		message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		message.WriteString("Content-Transfer-Encoding: 8bit\r\n")
		message.WriteString("\r\n")
		message.WriteString(email.HTML)
		message.WriteString("\r\n\r\n")

		message.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		// Plain text only
		message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		message.WriteString("Content-Transfer-Encoding: 8bit\r\n")
		message.WriteString("\r\n")
		message.WriteString(email.PlainText)
	}

	return message.String(), nil
}

// parseSMTPError extracts information from SMTP errors
func (e *EmailChannelProvider) parseSMTPError(err error) *SMTPResponse {
	if err == nil {
		return nil
	}

	errorMsg := err.Error()
	response := &SMTPResponse{
		Message:   errorMsg,
		Timestamp: time.Now(),
	}

	// Try to extract SMTP code from error message
	if strings.Contains(errorMsg, "550") {
		response.Code = 550
	} else if strings.Contains(errorMsg, "551") {
		response.Code = 551
	} else if strings.Contains(errorMsg, "552") {
		response.Code = 552
	} else if strings.Contains(errorMsg, "553") {
		response.Code = 553
	} else if strings.Contains(errorMsg, "554") {
		response.Code = 554
	} else if strings.Contains(errorMsg, "450") {
		response.Code = 450
	} else if strings.Contains(errorMsg, "451") {
		response.Code = 451
	} else if strings.Contains(errorMsg, "452") {
		response.Code = 452
	} else if strings.Contains(errorMsg, "421") {
		response.Code = 421
	} else {
		// Default to generic error code
		response.Code = 500
	}

	// Extract enhanced status code if present
	enhancedCodeRegex := regexp.MustCompile(`\d\.\d+\.\d+`)
	if match := enhancedCodeRegex.FindString(errorMsg); match != "" {
		response.EnhancedCode = match
	}

	return response
}

// ValidateAddress validates an email address
func (e *EmailChannelProvider) ValidateAddress(address string) error {
	if address == "" {
		return fmt.Errorf("email address is empty")
	}

	if !emailRegex.MatchString(address) {
		return fmt.Errorf("invalid email format")
	}

	// Additional validation: check domain length
	parts := strings.Split(address, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format")
	}

	domain := parts[1]
	if len(domain) > 253 {
		return fmt.Errorf("domain name too long")
	}

	// Check for consecutive dots
	if strings.Contains(address, "..") {
		return fmt.Errorf("consecutive dots not allowed")
	}

	return nil
}

// SendVerification sends a verification email
func (e *EmailChannelProvider) SendVerification(ctx context.Context, address string, token string) error {
	if err := e.ValidateAddress(address); err != nil {
		return fmt.Errorf("invalid email address: %w", err)
	}

	// Create verification message
	content := &models.MessageContent{
		Subject:   "Verify your email address",
		PlainText: fmt.Sprintf("Please verify your email address by clicking this link or entering this code: %s", token),
		Variables: map[string]interface{}{
			"verification_token": token,
		},
	}

	// Create temporary message for verification
	msg := &models.Message{
		ID:      uuid.New(),
		Type:    models.MessageAccountSecurity,
		Content: *content,
	}

	// Create temporary recipient
	recipient := &models.Recipient{
		UserID: uuid.New(), // Temporary ID for verification
		Preferences: models.UserNotificationSettings{
			Channels: map[models.DeliveryChannel]models.ChannelConfig{
				models.ChannelEmail: {
					Enabled: true,
					Address: address,
				},
			},
		},
	}

	attempt, err := e.DeliverMessage(ctx, msg, recipient)
	if err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	if attempt.Status == models.DeliveryStatusFailed {
		return fmt.Errorf("verification email delivery failed: %s", attempt.Error.Message)
	}

	return nil
}

// GetDeliveryStatus retrieves delivery status (placeholder implementation)
func (e *EmailChannelProvider) GetDeliveryStatus(ctx context.Context, messageID string) (*models.DeliveryAttempt, error) {
	// In a real implementation, this would query a delivery tracking system
	// For now, return a basic response
	return &models.DeliveryAttempt{
		ID:        uuid.New(),
		MessageID: uuid.MustParse(messageID),
		Channel:   models.ChannelEmail,
		Status:    models.DeliveryStatusDelivered,
	}, nil
}

// GetMetrics returns channel metrics for a time period
func (e *EmailChannelProvider) GetMetrics(ctx context.Context, start, end time.Time) (*models.ChannelMetrics, error) {
	// In a real implementation, this would aggregate metrics from the telemetry system
	// For now, return dummy metrics
	return &models.ChannelMetrics{
		Sent:         100,
		Delivered:    95,
		Failed:       5,
		DeliveryRate: 0.95,
		AvgLatency:   2500, // 2.5 seconds
	}, nil
}

// IsAvailable checks if the email channel is available
func (e *EmailChannelProvider) IsAvailable(ctx context.Context) bool {
	// Test connection to SMTP server
	conn, err := e.connectSMTP(ctx)
	if err != nil {
		e.telemetry.RecordError(models.ChannelEmail, "availability_check_failed", err)
		return false
	}
	defer conn.Close()

	return true
}

// DefaultSMTPConfig returns a default SMTP configuration
func DefaultSMTPConfig() *SMTPConfig {
	return &SMTPConfig{
		Host:        "localhost",
		Port:        25,
		UseTLS:      false,
		UseStartTLS: false,
		SkipVerify:  false,
		Timeout:     30 * time.Second,
		FromEmail:   "noreply@nuclear-ao3.org",
		FromName:    "Nuclear AO3",
		MaxRetries:  3,
		RetryDelay:  time.Minute,
	}
}

// AO3CompatibleSMTPConfig returns an AO3-compatible SMTP configuration
func AO3CompatibleSMTPConfig() *SMTPConfig {
	return &SMTPConfig{
		Host:        "localhost",
		Port:        25,
		UseTLS:      false,
		UseStartTLS: false,
		SkipVerify:  true,
		Timeout:     30 * time.Second,
		FromEmail:   "noreply@archiveofourown.org",
		FromName:    "Archive of Our Own",
		MaxRetries:  3,
		RetryDelay:  time.Minute,
	}
}

// CloudSMTPConfig returns a configuration for cloud SMTP services
func CloudSMTPConfig(host string, port int, username, password string) *SMTPConfig {
	return &SMTPConfig{
		Host:        host,
		Port:        port,
		Username:    username,
		Password:    password,
		UseTLS:      false,
		UseStartTLS: true,
		SkipVerify:  false,
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		RetryDelay:  time.Minute,
	}
}
