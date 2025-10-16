# File-Based Email Templates

This directory contains file-based email templates for the Nuclear AO3 messaging system.

## Directory Structure

```
files/
├── email/
│   ├── generic/              # Fallback template for all message types
│   │   ├── subject.txt       # Subject line template
│   │   ├── body.txt          # Plain text body template
│   │   └── body.html         # HTML body template (optional)
│   ├── subscription_update/  # Work subscription update notifications
│   │   ├── subject.txt
│   │   ├── body.txt
│   │   └── body.html
│   ├── comment_notification/ # Comment notifications
│   │   ├── subject.txt
│   │   ├── body.txt
│   │   └── body.html
│   └── password_reset/       # Password reset emails
│       ├── subject.txt
│       ├── body.txt
│       └── body.html
```

## Template Requirements

Each template directory must contain:
- `subject.txt` - The email subject line template
- `body.txt` - Plain text email body template

Optional files:
- `body.html` - HTML email body template

## Template Syntax

Templates use Go's template syntax with variables available from the message content:

### Common Variables
- `{{.subject}}` - Message subject from content
- `{{.plain_text}}` - Message plain text from content  
- `{{.html}}` - Message HTML from content
- `{{.action_url}}` - Action URL from content
- `{{.site_name}}` - Site name (default: "Nuclear AO3")
- `{{.site_url}}` - Site URL (default: "https://nuclear-ao3.local")

### Message-Specific Variables

**Subscription Updates:**
- `{{.work_title}}` - Title of the work
- `{{.author_name}}` - Name of the author
- `{{.chapter_title}}` - Title of the new chapter

**Comment Notifications:**
- `{{.work_title}}` - Title of the work
- `{{.commenter_name}}` - Name of the commenter

**Password Reset:**
- `{{.expiry_hours}}` - Hours until reset link expires (default: "24")

## Features

### ✅ Git Version Control
- Templates are stored in Git alongside code
- Changes tracked automatically
- Easy rollback and versioning
- Collaborative editing via pull requests

### ✅ Hot Reload (Development)
- Templates automatically reload when files change
- No server restart required for template updates
- Enabled by setting `hotReload: true` in renderer config

### ✅ Multi-format Support
- Plain text for all email clients
- Rich HTML for modern clients
- Automatic fallback to plain text

### ✅ Template Validation
- Syntax validation on load
- Missing template detection
- Graceful fallback to generic template

## Usage

```go
// Create file-based template renderer
templatesDir := "./shared/messaging/templates/files"
renderer, err := templates.NewFileBasedTemplateRenderer(templatesDir, true) // Hot reload enabled
if err != nil {
    log.Fatal(err)
}

// Use with email provider
emailProvider := email.NewEmailChannelProvider(config, telemetry, renderer, classifier)
```

## Template Management

### Adding New Templates
1. Create directory: `files/email/new_template_name/`
2. Add required files: `subject.txt`, `body.txt`
3. Optionally add: `body.html`
4. Commit to Git

### Updating Templates
1. Edit template files directly
2. In development: Changes reload automatically
3. In production: Deploy via normal deployment process
4. Commit changes to Git

### Testing Templates
Templates can be tested by:
1. Creating test message content
2. Calling `RenderEmailTemplate()` 
3. Inspecting rendered output
4. Sending test emails

## Migration from Hardcoded Templates

To migrate from hardcoded templates to file-based templates:

### 1. Update Template Renderer Initialization

**Before (Hardcoded):**
```go
templateRenderer := templates.NewEmailTemplateRenderer()
```

**After (File-based):**
```go
templatesDir := "./shared/messaging/templates/files"
templateRenderer, err := templates.NewFileBasedTemplateRenderer(templatesDir, hotReload)
if err != nil {
    log.Fatalf("Failed to initialize template renderer: %v", err)
}
```

### 2. Update Email Provider Initialization

**Before:**
```go
emailProvider := email.NewEmailChannelProvider(config, telemetry, templateRenderer, classifier)
```

**After (Same):**
```go
emailProvider := email.NewEmailChannelProvider(config, telemetry, templateRenderer, classifier)
```

The interface is the same, so no changes needed to the email provider.

### 3. Configuration Options

```go
// For development - enable hot reload
renderer, err := templates.NewFileBasedTemplateRenderer(templatesDir, true)

// For production - disable hot reload for performance
renderer, err := templates.NewFileBasedTemplateRenderer(templatesDir, false)
```

### 4. Template Validation

The file-based renderer includes automatic validation:
- Syntax errors detected at load time
- Missing templates fall back to generic template
- Validation errors logged with helpful details

### 5. Benefits of Migration

- **Designer-friendly**: Non-programmers can edit templates
- **Version control**: All changes tracked in Git
- **Hot reload**: Instant template updates in development
- **Better organization**: Templates organized in clear file structure
- **Validation**: Automatic syntax checking and error reporting

## Best Practices

1. **Keep templates simple** - Avoid complex logic in templates
2. **Test across email clients** - HTML renders differently everywhere
3. **Provide meaningful plain text** - Not everyone uses HTML email
4. **Use semantic HTML** - Tables for layout, proper headers
5. **Include unsubscribe info** - Required for bulk email
6. **Version control everything** - Templates are code!