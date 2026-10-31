// Package moderation holds what administrators and moderators can do to
// content and accounts: hide and unhide, mark spam, approve comments,
// suspend and ban users, grant and revoke roles.
//
// The rules come from AO3. Hiding is Admin::UserCreationsController#hide
// (app/controllers/admin/user_creations_controller.rb:19) and for comments
// CommentsController#hide (comments_controller.rb:511). Marking a work as spam
// hides it too (user_creations_controller.rb:36-52). Suspending and banning
// follow UserManager (app/models/user_manager.rb:83-110).
//
// Every function here makes its change and writes its audit row in one
// transaction, and is used by both the admin HTTP endpoints and the nuclear
// command line, so the two cannot disagree.
package moderation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/audit"
)

// What callers can get back besides a database error.
var (
	ErrNotFound     = errors.New("not found")
	ErrNoChange     = errors.New("already in that state")
	ErrIsSpam       = errors.New("the work is marked as spam; unmark it as spam instead")
	ErrBadInput     = errors.New("bad input")
	ErrProtectedAcc = errors.New("that account cannot be suspended or banned")
	ErrForbidden    = errors.New("not allowed")
)

// holdsRole says whether the account has an active role.
func holdsRole(ctx context.Context, tx *sql.Tx, id uuid.UUID, roles ...string) (bool, error) {
	var ok bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM user_roles
		WHERE user_id = $1 AND role = ANY($2) AND revoked_at IS NULL)`, id, pq.Array(roles)).Scan(&ok)
	return ok, err
}

// checkSanction enforces who may act on whom, for suspensions, bans and
// role changes. Nobody may act on their own account (no lifting your own ban,
// no granting yourself a role). Only an admin may act on someone who holds the
// admin or moderator role, so a moderator cannot discipline staff. Command
// line actors have no account and count as admins: they already have access
// to the server. Call it inside the transaction, before changing anything.
func checkSanction(ctx context.Context, tx *sql.Tx, actor audit.Actor, target uuid.UUID) error {
	if actor.UserID == nil {
		return nil
	}
	if *actor.UserID == target {
		return fmt.Errorf("%w: you can't do that to your own account", ErrForbidden)
	}
	isAdmin, err := holdsRole(ctx, tx, *actor.UserID, "admin")
	if err != nil || isAdmin {
		return err
	}
	staff, err := holdsRole(ctx, tx, target, "admin", "moderator")
	if err != nil {
		return err
	}
	if staff {
		return fmt.Errorf("%w: only an admin can do that to an admin or moderator", ErrForbidden)
	}
	return nil
}

// checkRoleChange: roles are changed by admins only (and the command line),
// never on yourself.
func checkRoleChange(ctx context.Context, tx *sql.Tx, actor audit.Actor, target uuid.UUID) error {
	if err := checkSanction(ctx, tx, actor, target); err != nil {
		return err
	}
	if actor.UserID == nil {
		return nil
	}
	isAdmin, err := holdsRole(ctx, tx, *actor.UserID, "admin")
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("%w: only an admin can change roles", ErrForbidden)
	}
	return nil
}

// Kind is a type of content that can be hidden.
type Kind string

const (
	Work     Kind = "work"
	Chapter  Kind = "chapter"
	Series   Kind = "series"
	Bookmark Kind = "bookmark"
	Comment  Kind = "comment"
)

// hideSQL gives, for each kind, the table and a short label that is saved in
// the audit row so a reader can tell what was hidden without a lookup.
var hideSQL = map[Kind]struct{ table, label string }{
	Work:     {"works", "title"},
	Chapter:  {"chapters", "work_id::text"},
	Series:   {"series", "title"},
	Bookmark: {"bookmarks", "work_id::text"},
	Comment:  {"comments", "COALESCE(work_id, (SELECT ch.work_id FROM chapters ch WHERE ch.id = chapter_id))::text"},
}

// Service does the moderation actions.
type Service struct {
	db *sql.DB
}

// New returns a Service on db.
func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// exists tells "no such row" from "row is already in that state".
func exists(ctx context.Context, tx *sql.Tx, table string, id uuid.UUID) (bool, error) {
	var ok bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM `+table+` WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

// SetHidden hides or unhides a work, chapter, series, bookmark or comment.
func (s *Service) SetHidden(ctx context.Context, actor audit.Actor, kind Kind, id uuid.UUID, hidden bool, reason string) error {
	h, ok := hideSQL[kind]
	if !ok {
		return ErrBadInput
	}
	if hidden && reason == "" {
		return fmt.Errorf("%w: a reason is required to hide", ErrBadInput)
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var label sql.NullString
		err := tx.QueryRowContext(ctx, `UPDATE `+h.table+` SET hidden_by_admin = $2
			WHERE id = $1 AND hidden_by_admin <> $2 RETURNING `+h.label, id, hidden).Scan(&label)
		if errors.Is(err, sql.ErrNoRows) {
			found, ferr := exists(ctx, tx, h.table, id)
			if ferr != nil {
				return ferr
			}
			if !found {
				return ErrNotFound
			}
			return ErrNoChange
		}
		if err != nil {
			return err
		}
		if kind == Work && !hidden {
			// A spam work stays out of sight until it is unmarked as spam.
			var spam bool
			if err := tx.QueryRowContext(ctx, `SELECT spam FROM works WHERE id = $1`, id).Scan(&spam); err != nil {
				return err
			}
			if spam {
				return ErrIsSpam
			}
		}
		action := "hide"
		if !hidden {
			action = "unhide"
		}
		return audit.Record(ctx, tx, actor, action, string(kind), id.String(), reason, map[string]any{"label": label.String})
	})
}

// SetWorkSpam marks a work as spam, or clears the mark. Like AO3
// (user_creations_controller.rb:36-52), marking hides the work and
// unmarking unhides it.
func (s *Service) SetWorkSpam(ctx context.Context, actor audit.Actor, id uuid.UUID, spam bool, reason string) error {
	if spam && reason == "" {
		return fmt.Errorf("%w: a reason is required to mark spam", ErrBadInput)
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var title string
		var wasHidden bool
		err := tx.QueryRowContext(ctx, `SELECT title, hidden_by_admin FROM works WHERE id = $1 AND spam <> $2 FOR UPDATE`,
			id, spam).Scan(&title, &wasHidden)
		if errors.Is(err, sql.ErrNoRows) {
			found, ferr := exists(ctx, tx, "works", id)
			if ferr != nil {
				return ferr
			}
			if !found {
				return ErrNotFound
			}
			return ErrNoChange
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE works SET spam = $2, hidden_by_admin = $2 WHERE id = $1`, id, spam); err != nil {
			return err
		}
		action := "spam"
		if !spam {
			action = "unspam"
		}
		return audit.Record(ctx, tx, actor, action, "work", id.String(), reason,
			map[string]any{"label": title, "was_hidden": wasHidden})
	})
}

// SetCommentSpam marks a comment as spam (AO3's "reject": the comment is
// shown to admins only) or clears the mark.
func (s *Service) SetCommentSpam(ctx context.Context, actor audit.Actor, id uuid.UUID, spam bool, reason string) error {
	if spam && reason == "" {
		return fmt.Errorf("%w: a reason is required to mark spam", ErrBadInput)
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE comments SET is_spam = $2 WHERE id = $1 AND is_spam <> $2`, id, spam)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return notFoundOrNoChange(ctx, tx, "comments", id)
		}
		action := "spam"
		if !spam {
			action = "unspam"
		}
		return audit.Record(ctx, tx, actor, action, "comment", id.String(), reason, nil)
	})
}

// ApproveComment takes a comment out of review and clears a spam mark, so it
// shows to everyone (AO3's "approve" and "review", comments_controller.rb:430-490).
func (s *Service) ApproveComment(ctx context.Context, actor audit.Actor, id uuid.UUID, reason string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE comments SET unreviewed = false, is_spam = false
			WHERE id = $1 AND (unreviewed OR is_spam)`, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return notFoundOrNoChange(ctx, tx, "comments", id)
		}
		return audit.Record(ctx, tx, actor, "approve", "comment", id.String(), reason, nil)
	})
}

// SetCommentFrozen freezes (AO3 "iced") or unfreezes a comment and every
// reply under it, so that nobody can reply.
func (s *Service) SetCommentFrozen(ctx context.Context, actor audit.Actor, id uuid.UUID, frozen bool, reason string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		n, err := FreezeThread(ctx, tx, id, frozen)
		if err != nil {
			return err
		}
		if n == 0 {
			return notFoundOrNoChange(ctx, tx, "comments", id)
		}
		action := "freeze"
		if !frozen {
			action = "unfreeze"
		}
		return audit.Record(ctx, tx, actor, action, "comment", id.String(), reason, map[string]any{"comments": n})
	})
}

// FreezeThread sets iced on a comment and all of its replies, and returns how
// many comments changed. The work's creator can do the same through the
// comment API (AO3: comments_controller.rb:217-231).
func FreezeThread(ctx context.Context, tx *sql.Tx, id uuid.UUID, frozen bool) (int64, error) {
	res, err := tx.ExecContext(ctx, `
		WITH RECURSIVE thread AS (
			SELECT id FROM comments WHERE id = $1
			UNION ALL
			SELECT c.id FROM comments c JOIN thread t ON c.parent_comment_id = t.id)
		UPDATE comments SET iced = $2 WHERE id IN (SELECT id FROM thread) AND iced <> $2`, id, frozen)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func notFoundOrNoChange(ctx context.Context, tx *sql.Tx, table string, id uuid.UUID) error {
	found, err := exists(ctx, tx, table, id)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	return ErrNoChange
}

// UserByName finds an account by its username.
func (s *Service) UserByName(ctx context.Context, username string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE username = $1`, username).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("no account named %q: %w", username, ErrNotFound)
	}
	return id, err
}

// lockUser reads and locks the account being changed. orphan_account holds
// the works of people who left, and AO3 refuses to suspend or ban it
// (user_manager.rb:52-58).
func lockUser(ctx context.Context, tx *sql.Tx, id uuid.UUID) (username string, banned, suspended bool, err error) {
	err = tx.QueryRowContext(ctx, `SELECT username, banned, suspended AND (suspended_until IS NULL OR suspended_until > now())
		FROM users WHERE id = $1 FOR UPDATE`, id).Scan(&username, &banned, &suspended)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, false, ErrNotFound
	}
	return
}

func refuseProtected(username string) error {
	if username == "orphan_account" {
		return ErrProtectedAcc
	}
	return nil
}

// SuspendUser suspends an account until the given time and ends its login
// sessions. A suspended user can read but not post (see platform.ActiveAccount).
// AO3's suspension takes a number of days; the end time is stored the same way.
func (s *Service) SuspendUser(ctx context.Context, actor audit.Actor, id uuid.UUID, until time.Time, reason string) error {
	if reason == "" {
		return fmt.Errorf("%w: a reason is required to suspend", ErrBadInput)
	}
	if !until.After(time.Now()) {
		return fmt.Errorf("%w: the suspension must end in the future", ErrBadInput)
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		username, _, _, err := lockUser(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := checkSanction(ctx, tx, actor, id); err != nil {
			return err
		}
		if err := refuseProtected(username); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE users SET suspended = true, suspended_until = $2 WHERE id = $1`, id, until); err != nil {
			return err
		}
		revoked, err := RevokeSessions(ctx, tx, username)
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, actor, "suspend", "user", id.String(), reason,
			map[string]any{"username": username, "until": until.UTC().Format(time.RFC3339), "sessions_ended": revoked.Sessions})
	})
}

// LiftSuspension ends a suspension early.
func (s *Service) LiftSuspension(ctx context.Context, actor audit.Actor, id uuid.UUID, reason string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		username, _, suspended, err := lockUser(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := checkSanction(ctx, tx, actor, id); err != nil {
			return err
		}
		if !suspended {
			return ErrNoChange
		}
		if _, err := tx.ExecContext(ctx, `UPDATE users SET suspended = false, suspended_until = NULL WHERE id = $1`, id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, actor, "unsuspend", "user", id.String(), reason, map[string]any{"username": username})
	})
}

// BanUser bans an account and ends its login sessions. Like a suspension it
// leaves the account able to read and to log in.
func (s *Service) BanUser(ctx context.Context, actor audit.Actor, id uuid.UUID, reason string) error {
	if reason == "" {
		return fmt.Errorf("%w: a reason is required to ban", ErrBadInput)
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		username, banned, _, err := lockUser(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := checkSanction(ctx, tx, actor, id); err != nil {
			return err
		}
		if err := refuseProtected(username); err != nil {
			return err
		}
		if banned {
			return ErrNoChange
		}
		if _, err := tx.ExecContext(ctx, `UPDATE users SET banned = true WHERE id = $1`, id); err != nil {
			return err
		}
		revoked, err := RevokeSessions(ctx, tx, username)
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, actor, "ban", "user", id.String(), reason,
			map[string]any{"username": username, "sessions_ended": revoked.Sessions})
	})
}

// UnbanUser lifts a ban.
func (s *Service) UnbanUser(ctx context.Context, actor audit.Actor, id uuid.UUID, reason string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		username, banned, _, err := lockUser(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := checkSanction(ctx, tx, actor, id); err != nil {
			return err
		}
		if !banned {
			return ErrNoChange
		}
		if _, err := tx.ExecContext(ctx, `UPDATE users SET banned = false WHERE id = $1`, id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, actor, "unban", "user", id.String(), reason, map[string]any{"username": username})
	})
}

// Roles that can be granted and revoked. Every account also has the plain
// "user" role, which is not managed here.
var Roles = []string{"admin", "moderator", "tag_wrangler"}

func validRole(role string) bool {
	for _, r := range Roles {
		if r == role {
			return true
		}
	}
	return false
}

// GrantRole gives an account a role, or restores it if it was revoked.
// A role that is already held is ErrNoChange.
func (s *Service) GrantRole(ctx context.Context, actor audit.Actor, userID uuid.UUID, role string) error {
	if !validRole(role) {
		return fmt.Errorf("%w: unknown role %q (use admin, moderator or tag_wrangler)", ErrBadInput, role)
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		username, _, _, err := lockUser(ctx, tx, userID)
		if err != nil {
			return err
		}
		if err := checkRoleChange(ctx, tx, actor, userID); err != nil {
			return err
		}
		var held bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM user_roles
			WHERE user_id = $1 AND role = $2 AND revoked_at IS NULL)`, userID, role).Scan(&held); err != nil {
			return err
		}
		if held {
			return ErrNoChange
		}
		var by any
		if actor.UserID != nil {
			by = *actor.UserID
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_roles (user_id, role, granted_by) VALUES ($1, $2, $3)
			ON CONFLICT (user_id, role) DO UPDATE
			SET revoked_at = NULL, revoked_by = NULL, granted_at = NOW(), granted_by = EXCLUDED.granted_by`,
			userID, role, by); err != nil {
			return err
		}
		return audit.Record(ctx, tx, actor, "grant_role", "user", userID.String(), "",
			map[string]any{"username": username, "role": role})
	})
}

// RevokeRole takes a role away. A role the account does not hold is ErrNoChange.
func (s *Service) RevokeRole(ctx context.Context, actor audit.Actor, userID uuid.UUID, role string) error {
	if !validRole(role) {
		return fmt.Errorf("%w: unknown role %q (use admin, moderator or tag_wrangler)", ErrBadInput, role)
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		username, _, _, err := lockUser(ctx, tx, userID)
		if err != nil {
			return err
		}
		if err := checkRoleChange(ctx, tx, actor, userID); err != nil {
			return err
		}
		var by any
		if actor.UserID != nil {
			by = *actor.UserID
		}
		res, err := tx.ExecContext(ctx, `UPDATE user_roles SET revoked_at = NOW(), revoked_by = $3
			WHERE user_id = $1 AND role = $2 AND revoked_at IS NULL`, userID, role, by)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNoChange
		}
		return audit.Record(ctx, tx, actor, "revoke_role", "user", userID.String(), "",
			map[string]any{"username": username, "role": role})
	})
}

// RoleHolder is one account with a role.
type RoleHolder struct {
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	GrantedAt time.Time `json:"granted_at"`
}

// RoleHolders lists everyone who currently holds a role other than "user",
// admins first.
func (s *Service) RoleHolders(ctx context.Context) ([]RoleHolder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.username, r.role, COALESCE(r.granted_at, u.created_at)
		FROM user_roles r JOIN users u ON u.id = r.user_id
		WHERE r.role <> 'user' AND r.revoked_at IS NULL
		ORDER BY CASE r.role WHEN 'admin' THEN 0 WHEN 'moderator' THEN 1 ELSE 2 END, u.username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RoleHolder
	for rows.Next() {
		var h RoleHolder
		if err := rows.Scan(&h.Username, &h.Role, &h.GrantedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// Revoked says how many of each thing RevokeSessions ended.
type Revoked struct{ Sessions, AccessTokens, RefreshTokens, Codes, Consents int64 }

// RevokeSessions ends login sessions and revokes OAuth tokens inside tx: for
// every account when username is empty, otherwise for that one account. Only
// sessions and tokens that were still live are counted.
func RevokeSessions(ctx context.Context, tx *sql.Tx, username string) (Revoked, error) {
	var r Revoked
	// With no username, "$1 = ''" is true for every row.
	if username != "" {
		var id string
		if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE username = $1`, username).Scan(&id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return r, fmt.Errorf("no account named %q", username)
			}
			return r, err
		}
	}
	for _, q := range []struct {
		dst *int64
		sql string
	}{
		{&r.Sessions, `UPDATE user_sessions SET is_active = false
			WHERE is_active AND ($1 = '' OR user_id = (SELECT id FROM users WHERE username = $1))`},
		{&r.AccessTokens, `UPDATE oauth_access_tokens SET is_revoked = true, revoked_at = now()
			WHERE NOT is_revoked AND ($1 = '' OR user_id = (SELECT id FROM users WHERE username = $1))`},
		{&r.RefreshTokens, `UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = now()
			WHERE NOT is_revoked AND ($1 = '' OR user_id = (SELECT id FROM users WHERE username = $1))`},
		// An unused code could still be swapped for new tokens.
		{&r.Codes, `UPDATE authorization_codes SET used_at = now()
			WHERE used_at IS NULL AND ($1 = '' OR user_id = (SELECT id FROM users WHERE username = $1))`},
		// Apps must ask the user for permission again.
		{&r.Consents, `UPDATE user_consents SET is_revoked = true, revoked_at = now()
			WHERE NOT is_revoked AND ($1 = '' OR user_id = (SELECT id FROM users WHERE username = $1))`},
	} {
		res, err := tx.ExecContext(ctx, q.sql, username)
		if err != nil {
			return r, err
		}
		*q.dst, _ = res.RowsAffected()
	}
	return r, nil
}
