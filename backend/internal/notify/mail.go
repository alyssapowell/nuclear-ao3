package notify

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"nuclear-ao3/internal/jobs"
	"nuclear-ao3/shared/email"
	"nuclear-ao3/shared/models"
)

// mailSender emails notifications to the people who get them by email,
// queueing one "email" job each. The emails are plain text, because
// notifications quote titles and names that anyone can write.
type mailSender struct {
	db  *sql.DB
	cfg *email.Config // nil: emails are off
}

func (m *mailSender) SendMessage(ctx context.Context, msg *models.Message) error {
	if m.cfg == nil {
		return nil
	}
	for _, r := range msg.Recipients {
		if !slices.Contains(r.Channels, models.ChannelEmail) {
			continue
		}
		var addr string
		err := m.db.QueryRowContext(ctx, `SELECT email FROM users WHERE id = $1 AND is_active`, r.UserID).Scan(&addr)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return fmt.Errorf("look up email for %s: %w", r.UserID, err)
		}
		if err := jobs.Enqueue(ctx, m.db, "email", m.render(addr, msg.Content)); err != nil {
			return err
		}
	}
	return nil
}

func (m *mailSender) render(to string, c models.MessageContent) email.Message {
	var text strings.Builder
	text.WriteString(c.PlainText)
	if text.Len() == 0 {
		text.WriteString(c.Subject)
	}
	if link := c.ActionURL; link != "" {
		if strings.HasPrefix(link, "/") {
			link = m.cfg.SiteURL + link
		}
		text.WriteString("\n\n" + link)
	}
	text.WriteString("\n\n--\nYou're getting this email because of your subscriptions on " + m.cfg.FromName +
		". Manage them at " + m.cfg.SiteURL + "/subscriptions\n")
	return email.Message{To: to, Subject: c.Subject, Text: text.String()}
}
