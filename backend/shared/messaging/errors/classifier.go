package errors

import (
	"strings"

	"nuclear-ao3/shared/models"
)

// SMTPErrorClassifier implements error classification for SMTP delivery errors
type SMTPErrorClassifier struct{}

// NewSMTPErrorClassifier creates a new SMTP error classifier
func NewSMTPErrorClassifier() *SMTPErrorClassifier {
	return &SMTPErrorClassifier{}
}

// ClassifySMTPError classifies SMTP errors into retryable categories
func (c *SMTPErrorClassifier) ClassifySMTPError(code int, message string) *models.DeliveryError {
	errorType, retryable := c.classifyByCode(code)

	// Further refine based on message content
	if errorType == "temporary_failure" {
		errorType, retryable = c.refineByMessage(message, retryable)
	}

	return &models.DeliveryError{
		Type:      errorType,
		Code:      string(rune(code)),
		Message:   message,
		Retryable: retryable,
		Details: map[string]interface{}{
			"smtp_code":    code,
			"smtp_message": message,
			"category":     c.getErrorCategory(errorType),
		},
	}
}

// classifyByCode provides initial classification based on SMTP response codes
func (c *SMTPErrorClassifier) classifyByCode(code int) (string, bool) {
	switch {
	// 2xx - Success (shouldn't be an error, but handle gracefully)
	case code >= 200 && code < 300:
		return "success", false

	// 4xx - Temporary failures (generally retryable)
	case code == 421:
		return "service_unavailable", true
	case code == 450:
		return "mailbox_busy", true
	case code == 451:
		return "temporary_failure", true
	case code == 452:
		return "insufficient_storage", true
	case code == 454:
		return "temporary_failure", true
	case code >= 400 && code < 500:
		return "temporary_failure", true

	// 5xx - Permanent failures (generally not retryable)
	case code == 500:
		return "syntax_error", false
	case code == 501:
		return "parameter_error", false
	case code == 502:
		return "command_not_implemented", false
	case code == 503:
		return "bad_sequence", false
	case code == 504:
		return "parameter_not_implemented", false
	case code == 550:
		return "mailbox_unavailable", false
	case code == 551:
		return "user_not_local", false
	case code == 552:
		return "storage_exceeded", false
	case code == 553:
		return "mailbox_name_invalid", false
	case code == 554:
		return "transaction_failed", false
	case code >= 500 && code < 600:
		return "permanent_failure", false

	// Unknown codes
	default:
		return "unknown_error", true // Default to retryable for unknown errors
	}
}

// refineByMessage refines error classification based on error message content
func (c *SMTPErrorClassifier) refineByMessage(message string, defaultRetryable bool) (string, bool) {
	lowerMessage := strings.ToLower(message)

	// Specific patterns that indicate different error types
	patterns := []struct {
		keywords  []string
		errorType string
		retryable bool
	}{
		// Authentication failures
		{[]string{"authentication", "auth", "login", "password"}, "auth_failed", false},
		{[]string{"invalid credentials", "bad username", "bad password"}, "auth_failed", false},

		// Rate limiting
		{[]string{"rate limit", "too many", "throttle", "quota"}, "rate_limited", true},
		{[]string{"hourly limit", "daily limit", "sending limit"}, "rate_limited", true},

		// Blacklisting/Reputation
		{[]string{"blacklist", "blocked", "reputation", "spam"}, "reputation_issue", false},
		{[]string{"rbl", "dnsbl", "spamhaus", "barracuda"}, "reputation_issue", false},

		// DNS/Network issues
		{[]string{"dns", "domain not found", "host not found"}, "dns_error", true},
		{[]string{"network", "connection", "timeout", "unreachable"}, "network_error", true},

		// Content filtering
		{[]string{"content", "filtered", "policy", "virus"}, "content_filtered", false},
		{[]string{"attachment", "file type", "malware"}, "content_filtered", false},

		// Mailbox issues
		{[]string{"mailbox full", "quota exceeded", "over quota"}, "mailbox_full", true},
		{[]string{"mailbox unavailable", "mailbox disabled"}, "mailbox_unavailable", false},
		{[]string{"user unknown", "no such user", "invalid recipient"}, "invalid_recipient", false},

		// Server issues
		{[]string{"server error", "internal error", "service unavailable"}, "server_error", true},
		{[]string{"maintenance", "temporarily unavailable"}, "server_maintenance", true},

		// SSL/TLS issues
		{[]string{"tls", "ssl", "certificate", "encryption"}, "tls_error", true},

		// Format/Protocol issues
		{[]string{"syntax error", "protocol error", "invalid format"}, "format_error", false},
		{[]string{"message too large", "size limit"}, "message_too_large", false},
	}

	for _, pattern := range patterns {
		for _, keyword := range pattern.keywords {
			if strings.Contains(lowerMessage, keyword) {
				return pattern.errorType, pattern.retryable
			}
		}
	}

	// If no specific pattern matches, return the original classification
	return "temporary_failure", defaultRetryable
}

// getErrorCategory returns a broad category for the error type
func (c *SMTPErrorClassifier) getErrorCategory(errorType string) string {
	switch errorType {
	case "auth_failed":
		return "authentication"
	case "rate_limited":
		return "rate_limiting"
	case "reputation_issue":
		return "reputation"
	case "dns_error", "network_error":
		return "connectivity"
	case "content_filtered":
		return "content"
	case "mailbox_full", "mailbox_unavailable", "invalid_recipient":
		return "recipient"
	case "server_error", "server_maintenance", "service_unavailable":
		return "server"
	case "tls_error":
		return "security"
	case "format_error", "syntax_error", "message_too_large":
		return "format"
	case "temporary_failure":
		return "temporary"
	case "permanent_failure":
		return "permanent"
	default:
		return "unknown"
	}
}

// IsRetryable determines if an error type should be retried
func (c *SMTPErrorClassifier) IsRetryable(errorType string) bool {
	retryableTypes := map[string]bool{
		"service_unavailable":  true,
		"mailbox_busy":         true,
		"temporary_failure":    true,
		"insufficient_storage": true,
		"rate_limited":         true,
		"dns_error":            true,
		"network_error":        true,
		"mailbox_full":         true,
		"server_error":         true,
		"server_maintenance":   true,
		"tls_error":            true,
		"unknown_error":        true,
	}

	return retryableTypes[errorType]
}

// GetRetryDelay returns suggested retry delay based on error type
func (c *SMTPErrorClassifier) GetRetryDelay(errorType string) int {
	// Return delay in seconds
	switch errorType {
	case "rate_limited":
		return 300 // 5 minutes for rate limiting
	case "server_maintenance", "service_unavailable":
		return 600 // 10 minutes for maintenance
	case "mailbox_busy", "insufficient_storage":
		return 180 // 3 minutes for busy/storage issues
	case "network_error", "dns_error":
		return 120 // 2 minutes for network issues
	case "server_error":
		return 240 // 4 minutes for server errors
	case "tls_error":
		return 60 // 1 minute for TLS issues
	default:
		return 60 // Default 1 minute delay
	}
}

// GetMaxRetries returns maximum retry attempts for error type
func (c *SMTPErrorClassifier) GetMaxRetries(errorType string) int {
	switch errorType {
	case "rate_limited":
		return 5 // Rate limits may resolve
	case "server_maintenance", "service_unavailable":
		return 3 // Maintenance is usually temporary
	case "mailbox_busy":
		return 4 // Busy mailboxes may become available
	case "network_error", "dns_error":
		return 3 // Network issues may resolve
	case "server_error":
		return 2 // Server errors may be temporary
	case "tls_error":
		return 2 // TLS issues may resolve
	case "insufficient_storage", "mailbox_full":
		return 2 // Storage issues may resolve
	default:
		return 1 // Conservative default
	}
}

// ClassifyError implements the ErrorClassifier interface
func (c *SMTPErrorClassifier) ClassifyError(err error, context map[string]interface{}) *models.DeliveryError {
	// Extract SMTP code and message from context if available
	var code int
	var message string

	if codeVal, exists := context["smtp_code"]; exists {
		if c, ok := codeVal.(int); ok {
			code = c
		}
	}

	if msgVal, exists := context["smtp_message"]; exists {
		if m, ok := msgVal.(string); ok {
			message = m
		}
	}

	// If we don't have SMTP-specific context, use the error message
	if code == 0 || message == "" {
		message = err.Error()
		// Try to extract code from error message
		if strings.Contains(message, "550") {
			code = 550
		} else if strings.Contains(message, "554") {
			code = 554
		} else if strings.Contains(message, "451") {
			code = 451
		} else {
			code = 500 // Default to generic server error
		}
	}

	return c.ClassifySMTPError(code, message)
}

// GetErrorCategory implements the ErrorClassifier interface
func (c *SMTPErrorClassifier) GetErrorCategory(errorType string) string {
	return c.getErrorCategory(errorType)
}
