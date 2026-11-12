// Package support is the Contact and Feedback forms: AO3's "Support and
// Feedback" and "Policy and Abuse" requests.
//
// Anyone, logged in or not, can send one. It is stored as a ticket (see
// migration 052) and emailed to the archive's support address, and the sender
// gets a confirmation. Tickets are read only by administrators and moderators
// (which is what the /api/v1/admin/support routes check). Nothing about a
// ticket, and in particular nothing about who sent it, is ever sent back to
// anyone else.
//
// Everything is plain text. Text is stored as typed (with control characters
// removed) and is never turned into HTML, here or in the website.
package support

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"nuclear-ao3/internal/audit"
	"nuclear-ao3/internal/jobs"
	"nuclear-ao3/shared/email"
)

// Limits, in characters (not bytes).
const (
	MaxSubject = 200
	MaxBody    = 10000
	MaxURL     = 2000
	MaxEmail   = 254
)

// The two kinds of ticket.
const (
	KindSupport = "support"
	KindAbuse   = "abuse"
)

var (
	ErrBadInput = errors.New("bad input")
	ErrNotFound = errors.New("not found")
	ErrNoChange = errors.New("no change")
)

func bad(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrBadInput, fmt.Sprintf(format, a...))
}

// Service stores and reads tickets.
type Service struct {
	db *sql.DB
	// supportEmail is where new tickets are emailed. Empty means they are
	// only stored.
	supportEmail string
}

// New makes a Service. supportEmail is SUPPORT_EMAIL.
func New(db *sql.DB, supportEmail string) *Service {
	return &Service{db: db, supportEmail: strings.TrimSpace(supportEmail)}
}

// Submission is what the form sends.
type Submission struct {
	Kind    string `json:"kind"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	// Email is required for guests. For a logged-in person it is ignored:
	// the account's own address is used.
	Email string `json:"email"`
	// URL is the page the report is about (optional).
	URL string `json:"url"`
}

// cleanText removes control characters other than newline and tab, turns
// carriage returns into newlines, and trims the ends.
func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ToValidUTF8(s, "")
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) || r == ' ' || r == ' ' || (r >= '‪' && r <= '‮') || (r >= '⁦' && r <= '⁩') {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(cleanText(s)), " ")
}

// validEmail returns the address if it is a plain address (no display name,
// no comments), as a reply can safely be sent to it.
func validEmail(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", bad("an email address is required")
	}
	if utf8.RuneCountInString(s) > MaxEmail {
		return "", bad("that email address is too long")
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s || !strings.Contains(s[strings.LastIndexByte(s, '@'):], ".") {
		return "", bad("that does not look like an email address")
	}
	return s, nil
}

// validate cleans a submission and checks it. The email is checked
// separately, since a logged-in person doesn't give one.
func (s *Submission) validate() error {
	if s.Kind == "" {
		s.Kind = KindSupport
	}
	if s.Kind != KindSupport && s.Kind != KindAbuse {
		return bad("kind must be support or abuse")
	}
	s.Subject = oneLine(s.Subject)
	s.Body = cleanText(s.Body)
	s.URL = oneLine(s.URL)
	if s.Subject == "" {
		return bad("a subject is required")
	}
	if utf8.RuneCountInString(s.Subject) > MaxSubject {
		return bad("the subject can be at most %d characters", MaxSubject)
	}
	if s.Body == "" {
		return bad("a message is required")
	}
	if utf8.RuneCountInString(s.Body) > MaxBody {
		return bad("the message can be at most %d characters", MaxBody)
	}
	if s.URL != "" {
		if len(s.URL) > MaxURL {
			return bad("the page address is too long")
		}
		u, err := url.Parse(s.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return bad("the page address must start with http:// or https://")
		}
	}
	return nil
}

func kindLabel(kind string) string {
	if kind == KindAbuse {
		return "Policy & Abuse"
	}
	return "Support & Feedback"
}

// Submit stores a ticket and queues its emails in one transaction. user is
// the logged-in sender, or nil for a guest.
func (s *Service) Submit(ctx context.Context, user *uuid.UUID, sub Submission) (uuid.UUID, error) {
	if err := sub.validate(); err != nil {
		return uuid.Nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback()

	var to string
	if user != nil {
		err = tx.QueryRowContext(ctx, `SELECT email FROM users WHERE id = $1`, *user).Scan(&to)
		if err != nil {
			return uuid.Nil, fmt.Errorf("support: account email: %w", err)
		}
	} else if to, err = validEmail(sub.Email); err != nil {
		return uuid.Nil, err
	}

	var id uuid.UUID
	var urlArg any
	if sub.URL != "" {
		urlArg = sub.URL
	}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO support_tickets (kind, subject, body, user_id, reporter_email, url)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		sub.Kind, sub.Subject, sub.Body, user, to, urlArg).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("support: insert: %w", err)
	}

	label := kindLabel(sub.Kind)
	if s.supportEmail != "" {
		text := fmt.Sprintf("%s request\nFrom: %s\n", label, to)
		if user != nil {
			text += "Logged in: yes\n"
		}
		if sub.URL != "" {
			text += "Page: " + sub.URL + "\n"
		}
		text += "Ticket: " + id.String() + "\n\n" + sub.Body + "\n"
		err = jobs.Enqueue(ctx, tx, "email", email.Message{
			To: s.supportEmail, Subject: "[" + label + "] " + sub.Subject, Text: text,
		})
		if err != nil {
			return uuid.Nil, err
		}
	}
	// The confirmation repeats nothing the sender wrote. A guest's address
	// isn't checked, so anything copied in here could be sent to a stranger
	// in the archive's name. For the same reason one address gets at most
	// one confirmation an hour; the message itself is always kept.
	// The lock makes messages naming the same address (in any case) wait
	// for each other, so two sent at once can't both get a confirmation.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('support-confirm:' || lower($1)))`, to); err != nil {
		return uuid.Nil, err
	}
	var confirmedRecently bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM jobs
		WHERE kind = 'email' AND lower(payload->>'to') = lower($1) AND payload->>'subject' = $2
		  AND created_at > now() - interval '1 hour')`, to, confirmationSubject).Scan(&confirmedRecently); err != nil {
		return uuid.Nil, err
	}
	if confirmedRecently {
		return id, tx.Commit()
	}
	err = jobs.Enqueue(ctx, tx, "email", email.Message{
		To:      to,
		Subject: confirmationSubject,
		Text: "Thank you for writing to us. We have received your " + label + " request and a volunteer will read it.\n\n" +
			"If you didn't send a message to us, you can ignore this email.\n",
	})
	if err != nil {
		return uuid.Nil, err
	}
	return id, tx.Commit()
}

// confirmationSubject is the subject of the email that tells a sender their
// message arrived.
const confirmationSubject = "We received your message"

// Ticket is a ticket as administrators see it.
type Ticket struct {
	ID            uuid.UUID  `json:"id"`
	Kind          string     `json:"kind"`
	Subject       string     `json:"subject"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
	ReporterEmail string     `json:"reporter_email"`
	ReporterUser  *uuid.UUID `json:"reporter_user_id,omitempty"`
	URL           string     `json:"url,omitempty"`
	Body          string     `json:"body,omitempty"`
	Notes         []Note     `json:"notes,omitempty"`
}

// Note is something staff wrote on a ticket.
type Note struct {
	ID        int64     `json:"id"`
	Kind      string    `json:"kind"` // "note" or "reply"
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

// ListFilter narrows List. Empty fields match everything.
type ListFilter struct {
	Status string
	Kind   string
	Limit  int
	Offset int
}

// List returns tickets, newest first, without their bodies or notes, and the
// total number that match.
func (s *Service) List(ctx context.Context, f ListFilter) ([]Ticket, int, error) {
	if f.Status != "" && f.Status != "open" && f.Status != "closed" {
		return nil, 0, bad("status must be open or closed")
	}
	if f.Kind != "" && f.Kind != KindSupport && f.Kind != KindAbuse {
		return nil, 0, bad("kind must be support or abuse")
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT count(*) FROM support_tickets
		WHERE ($1 = '' OR status = $1) AND ($2 = '' OR kind = $2)`, f.Status, f.Kind).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, subject, status, created_at, closed_at, reporter_email, user_id, COALESCE(url, '')
		FROM support_tickets
		WHERE ($1 = '' OR status = $1) AND ($2 = '' OR kind = $2)
		ORDER BY created_at DESC, id
		LIMIT $3 OFFSET $4`, f.Status, f.Kind, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scanTicket(r scanner) (Ticket, error) {
	var t Ticket
	var closed sql.NullTime
	var user uuid.NullUUID
	if err := r.Scan(&t.ID, &t.Kind, &t.Subject, &t.Status, &t.CreatedAt, &closed, &t.ReporterEmail, &user, &t.URL); err != nil {
		return t, err
	}
	if closed.Valid {
		t.ClosedAt = &closed.Time
	}
	if user.Valid {
		t.ReporterUser = &user.UUID
	}
	return t, nil
}

// Get returns one ticket with its body and notes.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Ticket, error) {
	t, err := scanTicket(s.db.QueryRowContext(ctx, `
		SELECT id, kind, subject, status, created_at, closed_at, reporter_email, user_id, COALESCE(url, '')
		FROM support_tickets WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT body FROM support_tickets WHERE id = $1`, id).Scan(&t.Body); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, body, author_label, created_at FROM support_ticket_notes
		WHERE ticket_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	t.Notes = []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Kind, &n.Body, &n.Author, &n.CreatedAt); err != nil {
			return nil, err
		}
		t.Notes = append(t.Notes, n)
	}
	return &t, rows.Err()
}

// AddNote adds an internal note to a ticket. With reply true the text is
// also emailed to the reporter. The audit row is written in the same
// transaction. The text of a note is not copied into the audit log.
func (s *Service) AddNote(ctx context.Context, actor audit.Actor, id uuid.UUID, text string, reply bool) error {
	text = cleanText(text)
	if text == "" {
		return bad("a note needs some text")
	}
	if utf8.RuneCountInString(text) > MaxBody {
		return bad("a note can be at most %d characters", MaxBody)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var to, subject, kind string
	err = tx.QueryRowContext(ctx, `SELECT reporter_email, subject, kind FROM support_tickets WHERE id = $1 FOR UPDATE`, id).Scan(&to, &subject, &kind)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	noteKind, action := "note", "support_note"
	if reply {
		noteKind, action = "reply", "support_reply"
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO support_ticket_notes (ticket_id, kind, body, author_id, author_label)
		VALUES ($1, $2, $3, $4, $5)`, id, noteKind, text, actor.UserID, actor.Label); err != nil {
		return err
	}
	if reply {
		err = jobs.Enqueue(ctx, tx, "email", email.Message{
			To:      to,
			Subject: "Re: " + subject,
			Text:    text + "\n\n--\nThis is a reply to your " + kindLabel(kind) + " request \"" + subject + "\".\n",
		})
		if err != nil {
			return err
		}
	}
	if err := audit.Record(ctx, tx, actor, action, "support_ticket", id.String(), "", nil); err != nil {
		return err
	}
	return tx.Commit()
}

// SetClosed closes or reopens a ticket.
func (s *Service) SetClosed(ctx context.Context, actor audit.Actor, id uuid.UUID, closed bool, reason string) error {
	reason = oneLine(reason)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	from, to, action := "closed", "open", "support_reopen"
	q := `UPDATE support_tickets SET status = 'open', closed_at = NULL, closed_by = NULL
	      WHERE id = $1 AND status = 'closed'`
	args := []any{id}
	if closed {
		from, to, action = "open", "closed", "support_close"
		q = `UPDATE support_tickets SET status = 'closed', closed_at = now(), closed_by = $2
		     WHERE id = $1 AND status = 'open'`
		args = append(args, actor.UserID)
	}
	res, err := tx.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM support_tickets WHERE id = $1)`, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		return ErrNoChange
	}
	if err := audit.Record(ctx, tx, actor, action, "support_ticket", id.String(), reason,
		map[string]string{"from": from, "to": to}); err != nil {
		return err
	}
	return tx.Commit()
}
