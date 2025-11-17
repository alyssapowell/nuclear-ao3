package email

import (
	"os"
	"strings"
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	// Set test environment variables
	os.Setenv("SMTP_HOST", "test-smtp.local")
	os.Setenv("SMTP_PORT", "2525")
	os.Setenv("SMTP_FROM", "test@example.com")
	os.Setenv("SMTP_FROM_NAME", "Test Sender")
	defer func() {
		os.Unsetenv("SMTP_HOST")
		os.Unsetenv("SMTP_PORT")
		os.Unsetenv("SMTP_FROM")
		os.Unsetenv("SMTP_FROM_NAME")
	}()

	config := LoadConfigFromEnv()

	if config.Host != "test-smtp.local" {
		t.Errorf("Expected Host to be 'test-smtp.local', got '%s'", config.Host)
	}
	if config.Port != "2525" {
		t.Errorf("Expected Port to be '2525', got '%s'", config.Port)
	}
	if config.From != "test@example.com" {
		t.Errorf("Expected From to be 'test@example.com', got '%s'", config.From)
	}
	if config.FromName != "Test Sender" {
		t.Errorf("Expected FromName to be 'Test Sender', got '%s'", config.FromName)
	}
}

func TestLoadConfigFromEnv_Defaults(t *testing.T) {
	// Clear environment variables to test defaults
	os.Unsetenv("SMTP_HOST")
	os.Unsetenv("SMTP_PORT")
	os.Unsetenv("SMTP_FROM")
	os.Unsetenv("SMTP_FROM_NAME")

	config := LoadConfigFromEnv()

	if config.Host != "mailhog" {
		t.Errorf("Expected default Host to be 'mailhog', got '%s'", config.Host)
	}
	if config.Port != "1025" {
		t.Errorf("Expected default Port to be '1025', got '%s'", config.Port)
	}
	if config.From != "noreply@nuclear-ao3.local" {
		t.Errorf("Expected default From to be 'noreply@nuclear-ao3.local', got '%s'", config.From)
	}
	if config.FromName != "Nuclear AO3" {
		t.Errorf("Expected default FromName to be 'Nuclear AO3', got '%s'", config.FromName)
	}
}

func TestWelcomeEmailTemplate(t *testing.T) {
	username := "testuser"
	subject, body := WelcomeEmailTemplate(username)

	if subject != "Welcome to Nuclear AO3!" {
		t.Errorf("Expected subject 'Welcome to Nuclear AO3!', got '%s'", subject)
	}

	if !strings.Contains(body, username) {
		t.Errorf("Expected body to contain username '%s'", username)
	}

	if !strings.Contains(body, "Nuclear AO3") {
		t.Error("Expected body to contain 'Nuclear AO3'")
	}

	if !strings.Contains(body, "Browse and search") {
		t.Error("Expected body to contain feature highlights")
	}
}

func TestWelcomeEmailHTMLTemplate(t *testing.T) {
	username := "testuser"
	subject, htmlBody := WelcomeEmailHTMLTemplate(username)

	if subject != "Welcome to Nuclear AO3!" {
		t.Errorf("Expected subject 'Welcome to Nuclear AO3!', got '%s'", subject)
	}

	if !strings.Contains(htmlBody, username) {
		t.Errorf("Expected HTML body to contain username '%s'", username)
	}

	if !strings.Contains(htmlBody, "<!DOCTYPE html>") {
		t.Error("Expected HTML body to be valid HTML")
	}

	if !strings.Contains(htmlBody, "<style>") {
		t.Error("Expected HTML body to contain styles")
	}

	if !strings.Contains(htmlBody, "Nuclear AO3") {
		t.Error("Expected HTML body to contain 'Nuclear AO3'")
	}

	// Check for feature highlights
	expectedFeatures := []string{
		"Browse and search",
		"Post your own",
		"Bookmark",
		"Subscribe",
		"Engage",
	}

	for _, feature := range expectedFeatures {
		if !strings.Contains(htmlBody, feature) {
			t.Errorf("Expected HTML body to contain feature '%s'", feature)
		}
	}
}

func TestWelcomeEmailHTMLTemplate_Escaping(t *testing.T) {
	// Test with special characters that should be handled properly
	username := "test<user>"
	_, htmlBody := WelcomeEmailHTMLTemplate(username)

	// The username should be in the email
	if !strings.Contains(htmlBody, username) {
		t.Errorf("Expected HTML body to contain username '%s'", username)
	}
}

func TestGetEnv(t *testing.T) {
	// Test with set environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	value := getEnv("TEST_VAR", "default_value")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}

	// Test with unset environment variable (should return default)
	value = getEnv("NONEXISTENT_VAR", "default_value")
	if value != "default_value" {
		t.Errorf("Expected 'default_value', got '%s'", value)
	}
}

// Note: We don't test actual email sending here because it requires an SMTP server
// Those tests should be integration tests, not unit tests
