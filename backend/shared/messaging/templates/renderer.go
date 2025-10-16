package templates

import (
	"bytes"
	"fmt"
	"html/template"
	texttemplate "text/template"

	"nuclear-ao3/shared/models"
)

// TemplateRenderer defines the interface for email template rendering
type TemplateRenderer interface {
	RenderEmailTemplate(messageType models.MessageType, content *models.MessageContent) (*RenderedEmail, error)
	GetTemplate(name string) (*EmailTemplate, bool)
	ListTemplates() []string
}

// EmailTemplateRenderer implements template rendering for email messages
type EmailTemplateRenderer struct {
	templates map[string]*EmailTemplate
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	Name        string
	MessageType models.MessageType
	Subject     *template.Template
	PlainText   *texttemplate.Template
	HTML        *template.Template
	DefaultVars map[string]interface{}
}

// RenderedEmail represents a rendered email
type RenderedEmail struct {
	Subject   string            `json:"subject"`
	PlainText string            `json:"plain_text"`
	HTML      string            `json:"html,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// NewEmailTemplateRenderer creates a new email template renderer
func NewEmailTemplateRenderer() *EmailTemplateRenderer {
	renderer := &EmailTemplateRenderer{
		templates: make(map[string]*EmailTemplate),
	}

	// Register default templates
	renderer.registerDefaultTemplates()

	return renderer
}

// RenderEmailTemplate renders an email template for a specific message type
func (r *EmailTemplateRenderer) RenderEmailTemplate(messageType models.MessageType, content *models.MessageContent) (*RenderedEmail, error) {
	templateName := string(messageType)
	emailTemplate, exists := r.templates[templateName]
	if !exists {
		// Use generic template as fallback
		emailTemplate = r.templates["generic"]
		if emailTemplate == nil {
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
	subject, err := r.renderTemplate(emailTemplate.Subject, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render subject: %w", err)
	}

	// Render plain text
	plainText, err := r.renderTextTemplate(emailTemplate.PlainText, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render plain text: %w", err)
	}

	rendered := &RenderedEmail{
		Subject:   subject,
		PlainText: plainText,
		Headers:   make(map[string]string),
	}

	// Render HTML if template exists
	if emailTemplate.HTML != nil {
		html, err := r.renderTemplate(emailTemplate.HTML, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to render HTML: %w", err)
		}
		rendered.HTML = html
	}

	// Add default headers
	rendered.Headers["X-Nuclear-AO3-Message-Type"] = string(messageType)
	rendered.Headers["X-Mailer"] = "Nuclear AO3 Messaging Service v1.0"

	return rendered, nil
}

// renderTemplate renders an HTML template
func (r *EmailTemplateRenderer) renderTemplate(tmpl *template.Template, variables map[string]interface{}) (string, error) {
	if tmpl == nil {
		return "", nil
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderTextTemplate renders a text template
func (r *EmailTemplateRenderer) renderTextTemplate(tmpl *texttemplate.Template, variables map[string]interface{}) (string, error) {
	if tmpl == nil {
		return "", nil
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// registerDefaultTemplates registers built-in email templates
func (r *EmailTemplateRenderer) registerDefaultTemplates() {
	// Generic template
	r.RegisterTemplate(&EmailTemplate{
		Name:        "generic",
		MessageType: "", // Used as fallback for any message type
		Subject:     template.Must(template.New("subject").Parse("{{.subject}}")),
		PlainText:   texttemplate.Must(texttemplate.New("plain").Parse("{{.plain_text}}")),
		HTML:        template.Must(template.New("html").Parse("{{.html}}")),
	})

	// Subscription update template
	r.RegisterTemplate(&EmailTemplate{
		Name:        "subscription_update",
		MessageType: models.MessageSubscriptionUpdate,
		Subject:     template.Must(template.New("subject").Parse("[AO3] New update: {{.work_title}}")),
		PlainText: texttemplate.Must(texttemplate.New("plain").Parse(`{{.author_name}} has updated "{{.work_title}}".

{{.plain_text}}

View the work: {{.action_url}}

---
You are receiving this because you subscribed to this work.
To manage your subscription preferences, visit your account settings.`)),
		HTML: template.Must(template.New("html").Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.work_title}} - New Update</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #990000;">{{.author_name}} has updated "{{.work_title}}"</h2>
        
        <div style="background: #f9f9f9; padding: 15px; border-radius: 5px; margin: 20px 0;">
            {{.html}}
        </div>
        
        <p><a href="{{.action_url}}" style="background: #990000; color: white; padding: 10px 20px; text-decoration: none; border-radius: 3px;">Read Update</a></p>
        
        <hr style="margin: 30px 0; border: none; border-top: 1px solid #ddd;">
        <p style="font-size: 12px; color: #666;">
            You are receiving this because you subscribed to this work.<br>
            To manage your subscription preferences, visit your account settings.
        </p>
    </div>
</body>
</html>`)),
		DefaultVars: map[string]interface{}{
			"site_name": "Archive of Our Own",
		},
	})

	// Comment notification template
	r.RegisterTemplate(&EmailTemplate{
		Name:        "comment_notification",
		MessageType: models.MessageCommentNotify,
		Subject:     template.Must(template.New("subject").Parse("[AO3] New comment on {{.work_title}}")),
		PlainText: texttemplate.Must(texttemplate.New("plain").Parse(`{{.commenter_name}} left a comment on "{{.work_title}}".

{{.plain_text}}

Reply to comment: {{.action_url}}

---
You are receiving this because you have comment notifications enabled.
To manage your notification preferences, visit your account settings.`)),
		HTML: template.Must(template.New("html").Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>New Comment - {{.work_title}}</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #990000;">New comment on "{{.work_title}}"</h2>
        
        <p><strong>{{.commenter_name}}</strong> said:</p>
        
        <div style="background: #f9f9f9; padding: 15px; border-radius: 5px; margin: 20px 0; border-left: 3px solid #990000;">
            {{.html}}
        </div>
        
        <p><a href="{{.action_url}}" style="background: #990000; color: white; padding: 10px 20px; text-decoration: none; border-radius: 3px;">Reply to Comment</a></p>
        
        <hr style="margin: 30px 0; border: none; border-top: 1px solid #ddd;">
        <p style="font-size: 12px; color: #666;">
            You are receiving this because you have comment notifications enabled.<br>
            To manage your notification preferences, visit your account settings.
        </p>
    </div>
</body>
</html>`)),
	})

	// Kudos notification template
	r.RegisterTemplate(&EmailTemplate{
		Name:        "kudos_notification",
		MessageType: models.MessageKudosNotify,
		Subject:     template.Must(template.New("subject").Parse("[AO3] {{.kudos_giver_name}} left kudos on {{.work_title}}")),
		PlainText: texttemplate.Must(texttemplate.New("plain").Parse(`{{.kudos_giver_name}} left kudos on "{{.work_title}}".

{{if .kudos_message}}{{.kudos_message}}{{end}}

View your work: {{.action_url}}

---
You are receiving this because you have kudos notifications enabled.
To manage your notification preferences, visit your account settings.`)),
		HTML: template.Must(template.New("html").Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>New Kudos - {{.work_title}}</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #990000;">{{.kudos_giver_name}} left kudos!</h2>
        
        <p>Someone appreciated your work "<strong>{{.work_title}}</strong>"</p>
        
        {{if .kudos_message}}
        <div style="background: #f9f9f9; padding: 15px; border-radius: 5px; margin: 20px 0;">
            {{.kudos_message}}
        </div>
        {{end}}
        
        <p><a href="{{.action_url}}" style="background: #990000; color: white; padding: 10px 20px; text-decoration: none; border-radius: 3px;">View Work</a></p>
        
        <hr style="margin: 30px 0; border: none; border-top: 1px solid #ddd;">
        <p style="font-size: 12px; color: #666;">
            You are receiving this because you have kudos notifications enabled.<br>
            To manage your notification preferences, visit your account settings.
        </p>
    </div>
</body>
</html>`)),
	})

	// Password reset template
	r.RegisterTemplate(&EmailTemplate{
		Name:        "password_reset",
		MessageType: models.MessagePasswordReset,
		Subject:     template.Must(template.New("subject").Parse("[AO3] Password Reset Request")),
		PlainText: texttemplate.Must(texttemplate.New("plain").Parse(`You have requested a password reset for your Archive of Our Own account.

To reset your password, click the following link or copy it into your browser:
{{.action_url}}

This link will expire in {{.expiry_hours}} hours.

If you did not request this password reset, please ignore this email. Your password will not be changed.

For security reasons, this email was sent from an automated system. Please do not reply to this email.`)),
		HTML: template.Must(template.New("html").Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset Request</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #990000;">Password Reset Request</h2>
        
        <p>You have requested a password reset for your Archive of Our Own account.</p>
        
        <p>To reset your password, click the button below:</p>
        
        <p style="text-align: center; margin: 30px 0;">
            <a href="{{.action_url}}" style="background: #990000; color: white; padding: 15px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Reset Password</a>
        </p>
        
        <p>This link will expire in <strong>{{.expiry_hours}} hours</strong>.</p>
        
        <p>If you did not request this password reset, please ignore this email. Your password will not be changed.</p>
        
        <hr style="margin: 30px 0; border: none; border-top: 1px solid #ddd;">
        <p style="font-size: 12px; color: #666;">
            For security reasons, this email was sent from an automated system. Please do not reply to this email.
        </p>
    </div>
</body>
</html>`)),
		DefaultVars: map[string]interface{}{
			"expiry_hours": "24",
		},
	})

	// System alert template
	r.RegisterTemplate(&EmailTemplate{
		Name:        "system_alert",
		MessageType: models.MessageSystemAlert,
		Subject:     template.Must(template.New("subject").Parse("[AO3] {{.alert_type}}: {{.subject}}")),
		PlainText: texttemplate.Must(texttemplate.New("plain").Parse(`{{.alert_type}} - {{.subject}}

{{.plain_text}}

{{if .action_url}}For more information, visit: {{.action_url}}{{end}}

---
This is an automated system notification from Archive of Our Own.`)),
		HTML: template.Must(template.New("html").Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.alert_type}} - {{.subject}}</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <div style="background: #fff3cd; border: 1px solid #ffeaa7; border-radius: 5px; padding: 15px; margin-bottom: 20px;">
            <h2 style="color: #856404; margin-top: 0;">{{.alert_type}}</h2>
            <h3 style="color: #856404;">{{.subject}}</h3>
        </div>
        
        <div style="margin: 20px 0;">
            {{.html}}
        </div>
        
        {{if .action_url}}
        <p><a href="{{.action_url}}" style="background: #990000; color: white; padding: 10px 20px; text-decoration: none; border-radius: 3px;">More Information</a></p>
        {{end}}
        
        <hr style="margin: 30px 0; border: none; border-top: 1px solid #ddd;">
        <p style="font-size: 12px; color: #666;">
            This is an automated system notification from Archive of Our Own.
        </p>
    </div>
</body>
</html>`)),
	})
}

// RegisterTemplate registers a new email template
func (r *EmailTemplateRenderer) RegisterTemplate(template *EmailTemplate) error {
	if template.Name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	r.templates[template.Name] = template
	return nil
}

// GetTemplate retrieves a template by name
func (r *EmailTemplateRenderer) GetTemplate(name string) (*EmailTemplate, bool) {
	template, exists := r.templates[name]
	return template, exists
}

// ListTemplates returns all registered template names
func (r *EmailTemplateRenderer) ListTemplates() []string {
	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}
