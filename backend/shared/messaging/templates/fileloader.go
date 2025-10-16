package templates

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	texttemplate "text/template"
	"time"

	"nuclear-ao3/shared/models"
)

// ValidationError represents a template validation error
type ValidationError struct {
	TemplateName string
	File         string
	Err          error
}

func (v ValidationError) Error() string {
	return fmt.Sprintf("template %s, file %s: %v", v.TemplateName, v.File, v.Err)
}

// FileBasedTemplateRenderer loads templates from the filesystem
type FileBasedTemplateRenderer struct {
	mu           sync.RWMutex
	templatesDir string
	templates    map[string]*EmailTemplate
	lastModified map[string]time.Time
	hotReload    bool
}

// NewFileBasedTemplateRenderer creates a new file-based template renderer
func NewFileBasedTemplateRenderer(templatesDir string, hotReload bool) (*FileBasedTemplateRenderer, error) {
	renderer := &FileBasedTemplateRenderer{
		templatesDir: templatesDir,
		templates:    make(map[string]*EmailTemplate),
		lastModified: make(map[string]time.Time),
		hotReload:    hotReload,
	}

	// Load all templates on startup
	if err := renderer.LoadAllTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Validate all loaded templates
	if validationErrors := renderer.ValidateTemplates(); len(validationErrors) > 0 {
		log.Printf("Warning: Found %d template validation errors:", len(validationErrors))
		for _, err := range validationErrors {
			log.Printf("  - %s", err.Error())
		}
	}

	return renderer, nil
}

// LoadAllTemplates loads all templates from the filesystem
func (r *FileBasedTemplateRenderer) LoadAllTemplates() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	templatesPath := filepath.Join(r.templatesDir, "email")
	if _, err := os.Stat(templatesPath); os.IsNotExist(err) {
		return fmt.Errorf("templates directory does not exist: %s", templatesPath)
	}

	// Walk through template directories
	err := filepath.WalkDir(templatesPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root email directory itself
		if path == templatesPath {
			return nil
		}

		// Only process directories that are direct children of the email templates directory
		if d.IsDir() && filepath.Dir(path) == templatesPath {
			templateName := d.Name()
			templateDir := path

			// Load the template from this directory
			emailTemplate, err := r.loadTemplateFromDirectory(templateName, templateDir)
			if err != nil {
				log.Printf("Warning: Failed to load template %s: %v", templateName, err)
				return nil // Continue with other templates
			}

			if emailTemplate != nil {
				r.templates[templateName] = emailTemplate
				log.Printf("Loaded template: %s", templateName)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk templates directory: %w", err)
	}

	log.Printf("Loaded %d email templates from %s", len(r.templates), templatesPath)
	return nil
}

// loadTemplateFromDirectory loads a template from a directory containing template files
func (r *FileBasedTemplateRenderer) loadTemplateFromDirectory(name, dir string) (*EmailTemplate, error) {
	emailTemplate := &EmailTemplate{
		Name:        name,
		DefaultVars: make(map[string]interface{}),
	}

	// Map template name to message type
	emailTemplate.MessageType = r.mapNameToMessageType(name)

	// Track latest modification time for hot reloading
	var latestMod time.Time

	// Load subject template
	subjectPath := filepath.Join(dir, "subject.txt")
	if subjectContent, modTime, err := r.loadFileWithModTime(subjectPath); err == nil {
		subjectTmpl, err := template.New("subject").Parse(subjectContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse subject template: %w", err)
		}
		emailTemplate.Subject = subjectTmpl
		if modTime.After(latestMod) {
			latestMod = modTime
		}
	}

	// Load plain text body template
	bodyTextPath := filepath.Join(dir, "body.txt")
	if bodyContent, modTime, err := r.loadFileWithModTime(bodyTextPath); err == nil {
		bodyTmpl, err := texttemplate.New("body").Parse(bodyContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse body text template: %w", err)
		}
		emailTemplate.PlainText = bodyTmpl
		if modTime.After(latestMod) {
			latestMod = modTime
		}
	}

	// Load HTML body template
	bodyHTMLPath := filepath.Join(dir, "body.html")
	if htmlContent, modTime, err := r.loadFileWithModTime(bodyHTMLPath); err == nil {
		htmlTmpl, err := template.New("html").Parse(htmlContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse HTML template: %w", err)
		}
		emailTemplate.HTML = htmlTmpl
		if modTime.After(latestMod) {
			latestMod = modTime
		}
	}

	// Check if we have at least subject and plain text
	if emailTemplate.Subject == nil || emailTemplate.PlainText == nil {
		return nil, fmt.Errorf("template %s must have at least subject.txt and body.txt", name)
	}

	// Set default variables based on template type
	r.setDefaultVariables(emailTemplate)

	// Store modification time for hot reloading
	r.lastModified[name] = latestMod

	return emailTemplate, nil
}

// loadFileWithModTime loads a file and returns its content and modification time
func (r *FileBasedTemplateRenderer) loadFileWithModTime(path string) (string, time.Time, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", time.Time{}, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", time.Time{}, err
	}

	return string(content), stat.ModTime(), nil
}

// mapNameToMessageType maps template directory names to message types
func (r *FileBasedTemplateRenderer) mapNameToMessageType(name string) models.MessageType {
	switch name {
	case "subscription_update":
		return models.MessageSubscriptionUpdate
	case "comment_notification":
		return models.MessageCommentNotify
	case "kudos_notification":
		return models.MessageKudosNotify
	case "system_alert":
		return models.MessageSystemAlert
	case "password_reset":
		return models.MessagePasswordReset
	case "account_security":
		return models.MessageAccountSecurity
	case "collection_update":
		return models.MessageCollectionUpdate
	case "series_update":
		return models.MessageSeriesUpdate
	case "invitation":
		return models.MessageInvitation
	default:
		return "" // Generic template
	}
}

// setDefaultVariables sets default variables for templates based on their type
func (r *FileBasedTemplateRenderer) setDefaultVariables(template *EmailTemplate) {
	// Common defaults for all templates
	template.DefaultVars["site_name"] = "Nuclear AO3"
	template.DefaultVars["site_url"] = "https://nuclear-ao3.local"

	// Template-specific defaults
	switch template.MessageType {
	case models.MessagePasswordReset:
		template.DefaultVars["expiry_hours"] = "24"
	case models.MessageSystemAlert:
		template.DefaultVars["alert_type"] = "Notice"
	}
}

// RenderEmailTemplate renders an email template for a specific message type
func (r *FileBasedTemplateRenderer) RenderEmailTemplate(messageType models.MessageType, content *models.MessageContent) (*RenderedEmail, error) {
	// Check for hot reload if enabled
	if r.hotReload {
		r.checkAndReloadIfNeeded()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Find template by message type first, then by name
	var emailTemplate *EmailTemplate
	templateName := string(messageType)

	// Try to find by message type
	for _, tmpl := range r.templates {
		if tmpl.MessageType == messageType {
			emailTemplate = tmpl
			break
		}
	}

	// Fall back to name-based lookup
	if emailTemplate == nil {
		if tmpl, exists := r.templates[templateName]; exists {
			emailTemplate = tmpl
		}
	}

	// Fall back to generic template
	if emailTemplate == nil {
		if tmpl, exists := r.templates["generic"]; exists {
			emailTemplate = tmpl
		} else {
			return nil, fmt.Errorf("no template found for message type %s and no generic fallback", messageType)
		}
	}

	// Merge content variables with template defaults
	variables := make(map[string]interface{})
	for k, v := range emailTemplate.DefaultVars {
		variables[k] = v
	}
	for k, v := range content.Variables {
		variables[k] = v
	}

	// Add content fields to variables
	variables["subject"] = content.Subject
	variables["plain_text"] = content.PlainText
	variables["html"] = content.HTML
	variables["action_url"] = content.ActionURL

	// Render subject
	subject, err := r.renderHTMLTemplate(emailTemplate.Subject, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render subject: %w", err)
	}

	// Render plain text
	plainText, err := r.renderTextTemplate(emailTemplate.PlainText, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render plain text: %w", err)
	}

	rendered := &RenderedEmail{
		Subject:   strings.TrimSpace(subject),
		PlainText: plainText,
		Headers:   make(map[string]string),
	}

	// Render HTML if template exists
	if emailTemplate.HTML != nil {
		html, err := r.renderHTMLTemplate(emailTemplate.HTML, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to render HTML: %w", err)
		}
		rendered.HTML = html
	}

	// Add default headers
	rendered.Headers["X-Nuclear-AO3-Message-Type"] = string(messageType)
	rendered.Headers["X-Mailer"] = "Nuclear AO3 Messaging Service v1.0"
	rendered.Headers["X-Template-Name"] = emailTemplate.Name

	return rendered, nil
}

// renderHTMLTemplate renders an HTML template
func (r *FileBasedTemplateRenderer) renderHTMLTemplate(tmpl *template.Template, variables map[string]interface{}) (string, error) {
	if tmpl == nil {
		return "", nil
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderTextTemplate renders a text template
func (r *FileBasedTemplateRenderer) renderTextTemplate(tmpl *texttemplate.Template, variables map[string]interface{}) (string, error) {
	if tmpl == nil {
		return "", nil
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// checkAndReloadIfNeeded checks if templates have been modified and reloads them
func (r *FileBasedTemplateRenderer) checkAndReloadIfNeeded() {
	// This is a simplified hot reload - in production you might want to use file system watchers
	templatesPath := filepath.Join(r.templatesDir, "email")

	// Walk through template directories and check modification times
	filepath.WalkDir(templatesPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}

		// Skip root directory
		if path == templatesPath {
			return nil
		}

		templateName := d.Name()

		// Check if any template files in this directory have been modified
		templateDir := path
		files := []string{"subject.txt", "body.txt", "body.html"}

		for _, file := range files {
			filePath := filepath.Join(templateDir, file)
			if stat, err := os.Stat(filePath); err == nil {
				if lastMod, exists := r.lastModified[templateName]; exists && stat.ModTime().After(lastMod) {
					log.Printf("Template %s has been modified, reloading...", templateName)

					// Reload this specific template
					r.mu.Lock()
					if newTemplate, err := r.loadTemplateFromDirectory(templateName, templateDir); err == nil {
						r.templates[templateName] = newTemplate
						log.Printf("Reloaded template: %s", templateName)
					} else {
						log.Printf("Failed to reload template %s: %v", templateName, err)
					}
					r.mu.Unlock()
					break
				}
			}
		}

		return nil
	})
}

// GetTemplate retrieves a template by name
func (r *FileBasedTemplateRenderer) GetTemplate(name string) (*EmailTemplate, bool) {
	if r.hotReload {
		r.checkAndReloadIfNeeded()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	template, exists := r.templates[name]
	return template, exists
}

// ListTemplates returns all loaded template names
func (r *FileBasedTemplateRenderer) ListTemplates() []string {
	if r.hotReload {
		r.checkAndReloadIfNeeded()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}

// ReloadTemplates manually reloads all templates from disk
func (r *FileBasedTemplateRenderer) ReloadTemplates() error {
	log.Println("Manually reloading all templates...")
	return r.LoadAllTemplates()
}

// ValidateTemplates validates all loaded templates for syntax errors
func (r *FileBasedTemplateRenderer) ValidateTemplates() []ValidationError {
	var errors []ValidationError

	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, template := range r.templates {
		// Validate each template component
		errors = append(errors, r.validateTemplate(name, template)...)
	}

	return errors
}

// validateTemplate validates a single template's syntax
func (r *FileBasedTemplateRenderer) validateTemplate(name string, template *EmailTemplate) []ValidationError {
	var errors []ValidationError

	// Create test data for validation
	testData := map[string]interface{}{
		"site_name":      "Test Site",
		"work_title":     "Test Work",
		"author_name":    "Test Author",
		"commenter_name": "Test Commenter",
		"action_url":     "https://example.com",
		"plain_text":     "Test plain text",
		"html":           "<p>Test HTML</p>",
		"subject":        "Test Subject",
		"expiry_hours":   "24",
		"alert_type":     "Notice",
	}

	// Validate subject template
	if template.Subject != nil {
		if err := r.testTemplateExecution("subject", template.Subject, testData); err != nil {
			errors = append(errors, ValidationError{
				TemplateName: name,
				File:         "subject.txt",
				Err:          err,
			})
		}
	}

	// Validate plain text template
	if template.PlainText != nil {
		if err := r.testTextTemplateExecution("plain text", template.PlainText, testData); err != nil {
			errors = append(errors, ValidationError{
				TemplateName: name,
				File:         "body.txt",
				Err:          err,
			})
		}
	}

	// Validate HTML template
	if template.HTML != nil {
		if err := r.testTemplateExecution("HTML", template.HTML, testData); err != nil {
			errors = append(errors, ValidationError{
				TemplateName: name,
				File:         "body.html",
				Err:          err,
			})
		}
	}

	return errors
}

// testTemplateExecution tests HTML template execution with test data
func (r *FileBasedTemplateRenderer) testTemplateExecution(templateType string, tmpl *template.Template, testData map[string]interface{}) error {
	var buf strings.Builder
	if err := tmpl.Execute(&buf, testData); err != nil {
		return fmt.Errorf("failed to execute %s template: %w", templateType, err)
	}
	return nil
}

// testTextTemplateExecution tests text template execution with test data
func (r *FileBasedTemplateRenderer) testTextTemplateExecution(templateType string, tmpl *texttemplate.Template, testData map[string]interface{}) error {
	var buf strings.Builder
	if err := tmpl.Execute(&buf, testData); err != nil {
		return fmt.Errorf("failed to execute %s template: %w", templateType, err)
	}
	return nil
}

// ValidateTemplateFile validates a single template file for syntax errors
func (r *FileBasedTemplateRenderer) ValidateTemplateFile(templateName, fileName string) error {
	templatesPath := filepath.Join(r.templatesDir, "email")
	templateDir := filepath.Join(templatesPath, templateName)
	filePath := filepath.Join(templateDir, fileName)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("template file does not exist: %s", filePath)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Parse template based on file type
	switch filepath.Ext(fileName) {
	case ".html":
		_, err = template.New("test").Parse(string(content))
	case ".txt":
		_, err = texttemplate.New("test").Parse(string(content))
	default:
		return fmt.Errorf("unsupported template file type: %s", filepath.Ext(fileName))
	}

	if err != nil {
		return fmt.Errorf("template syntax error: %w", err)
	}

	return nil
}
