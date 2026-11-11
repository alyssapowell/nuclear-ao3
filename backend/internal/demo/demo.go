// Package demo fills a trial archive with made-up accounts and works, so the
// site can be tried without writing a story first.
//
// It is for trial archives only (TRIAL_NO_EMAIL=true) and refuses to run
// anywhere else, because every account it makes shares one password. The
// data is written with SQL in one transaction, through the same tables and
// triggers the app uses: counts, tag totals and search text come out exactly
// as if people had posted it. The text is original and is made from word
// lists; the fandoms are in the public domain. The random seed is fixed, so
// the same day gives the same archive.
package demo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	// Seed makes the archive the same every time it is loaded.
	Seed = 20250801
	// AccountCount is how many demo accounts there are.
	AccountCount = 25
	// Domain is the email domain of every demo account. Mail to example.org
	// is never delivered, so nothing here can reach a real person.
	Domain = "example.org"

	// MinPasswordLen and MaxPasswordLen bound DEMO_PASSWORD. The top is the
	// longest password sign-up accepts.
	MinPasswordLen = 12
	MaxPasswordLen = 40
)

// AccountEmail is the login of demo account n (1 to AccountCount):
// demo01@example.org and so on.
func AccountEmail(n int) string { return fmt.Sprintf("demo%02d@%s", n, Domain) }

// ErrNotTrial is returned when TRIAL_NO_EMAIL is not "true".
var ErrNotTrial = errors.New("demo data is only for trial archives: set TRIAL_NO_EMAIL=true to load it. It is refused anywhere else because every demo account shares one password")

// Password checks the settings and returns DEMO_PASSWORD. Demo data is only
// loaded when TRIAL_NO_EMAIL is "true" and the password is long enough; the
// password itself is never put in a message.
func Password(getenv func(string) string) (string, error) {
	if getenv("TRIAL_NO_EMAIL") != "true" {
		return "", ErrNotTrial
	}
	pw := getenv("DEMO_PASSWORD")
	if pw == "" {
		return "", fmt.Errorf("DEMO_PASSWORD is not set: choose a password of %d to %d characters for the demo accounts (anyone who knows it can log in as them)", MinPasswordLen, MaxPasswordLen)
	}
	if n := utf8.RuneCountInString(pw); n < MinPasswordLen || n > MaxPasswordLen || len(pw) > 72 {
		return "", fmt.Errorf("DEMO_PASSWORD must be %d to %d characters long", MinPasswordLen, MaxPasswordLen)
	}
	return pw, nil
}

// Summary says what is in the demo archive.
type Summary struct {
	// Skipped is true when the demo accounts were already there and nothing
	// was written.
	Skipped bool

	Accounts, Pseuds, Tags               int
	Works, Drafts, Anonymous, Restricted int
	Chapters                             int
	Comments, GuestComments, Replies     int
	Kudos, GuestKudos                    int
	Bookmarks, PrivateBookmarks          int
	Series, Collections, Subscriptions   int
	Blocks, Mutes                        int
	InAnonymousCollection, InUnrevealed  int
}

func (s Summary) String() string {
	if s.Skipped {
		return "demo accounts already exist; nothing loaded"
	}
	return fmt.Sprintf("%d accounts with %d pseuds, %d works (%d drafts, %d anonymous, %d members-only) in %d chapters, "+
		"%d comments (%d replies, %d from guests), %d kudos (%d from guests), %d bookmarks (%d private), "+
		"%d series, %d collections, %d subscriptions, %d blocks, %d mutes, %d tags",
		s.Accounts, s.Pseuds, s.Works, s.Drafts, s.Anonymous, s.Restricted, s.Chapters,
		s.Comments, s.Replies, s.GuestComments, s.Kudos, s.GuestKudos, s.Bookmarks, s.PrivateBookmarks,
		s.Series, s.Collections, s.Subscriptions, s.Blocks, s.Mutes, s.Tags)
}

// Load writes the demo archive to a migrated database, in one transaction,
// unless demo accounts are already there. Every account gets the password,
// and no account is given a role. now is the day the two years of dates end
// on.
func Load(ctx context.Context, db *sql.DB, password string, now time.Time) (*Summary, error) {
	exists := func(q interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}) (bool, error) {
		var found bool
		err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email ~ '^demo[0-9]{2}@example\.org$')`).Scan(&found)
		return found, err
	}
	if found, err := exists(db); err != nil || found {
		return &Summary{Skipped: found}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("demo data: hash password: %w", err)
	}
	p := newPlan(now.UTC().Truncate(24 * time.Hour))

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Two copies of the app starting together must not both load it.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('nuclear-demo-data'))`); err != nil {
		return nil, err
	}
	if found, err := exists(tx); err != nil || found {
		return &Summary{Skipped: found}, err
	}
	if err := p.write(ctx, tx, string(hash)); err != nil {
		return nil, fmt.Errorf("demo data: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return p.summary(), nil
}

func (p *plan) summary() *Summary {
	s := &Summary{Accounts: len(p.accounts), Pseuds: len(p.pseuds), Tags: len(p.tags), Works: len(p.works),
		Comments: len(p.comments), Kudos: len(p.kudos), Bookmarks: len(p.marks), Series: len(p.series),
		Collections: len(p.colls), Subscriptions: len(p.subs), Blocks: len(p.blocks), Mutes: len(p.mutes)}
	for _, w := range p.works {
		s.Chapters += len(w.chapters)
		for _, b := range []struct {
			on bool
			n  *int
		}{{w.draft, &s.Drafts}, {w.anon, &s.Anonymous}, {w.restricted, &s.Restricted}, {w.inAnon, &s.InAnonymousCollection}, {w.inUnrevealed, &s.InUnrevealed}} {
			if b.on {
				*b.n++
			}
		}
	}
	for _, c := range p.comments {
		if c.acct == nil {
			s.GuestComments++
		}
		if c.parent != nil {
			s.Replies++
		}
	}
	for _, k := range p.kudos {
		if k.acct == nil {
			s.GuestKudos++
		}
	}
	for _, b := range p.marks {
		if b.private {
			s.PrivateBookmarks++
		}
	}
	return s
}
