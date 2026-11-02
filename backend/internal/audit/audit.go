// Package audit writes and reads the record of administrator actions.
//
// Every action that a moderator or administrator takes (hiding a work,
// suspending an account, granting a role) is recorded in the admin_actions
// table in the same database transaction as the change itself, so there is
// never a change without a row, or a row without a change. The table is
// append-only: see migration 034.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/user"
	"time"

	"github.com/google/uuid"
)

// Actor is who did something.
type Actor struct {
	UserID *uuid.UUID // nil for actions taken on the command line
	Label  string     // the account's email address, or "cli:<operating system user>"
}

// CLIActor is the actor for commands run with the nuclear program on the
// server. There is no site account behind them, so the operating system user
// who ran the command is recorded instead.
func CLIActor() Actor {
	name := "unknown"
	if u, err := user.Current(); err == nil && u.Username != "" {
		name = u.Username
	}
	return Actor{Label: "cli:" + name}
}

// UserActor is the actor for a logged-in administrator.
func UserActor(id uuid.UUID, email string) Actor {
	return Actor{UserID: &id, Label: email}
}

// Execer is what Record writes through. Pass the *sql.Tx of the change
// being recorded.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Record adds one row to the log. Call it inside the transaction that makes
// the change, and let that transaction's commit or rollback decide whether
// the row exists. details may be nil.
func Record(ctx context.Context, tx Execer, actor Actor, action, targetType, targetID, reason string, details any) error {
	if actor.Label == "" {
		return fmt.Errorf("audit: an action needs an actor")
	}
	var detailsJSON any
	if details != nil {
		b, err := json.Marshal(details)
		if err != nil {
			return fmt.Errorf("audit: details: %w", err)
		}
		detailsJSON = string(b)
	}
	var reasonArg any
	if reason != "" {
		reasonArg = reason
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO admin_actions (actor_user_id, actor_label, action, target_type, target_id, reason, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		actor.UserID, actor.Label, action, targetType, targetID, reasonArg, detailsJSON)
	if err != nil {
		return fmt.Errorf("audit: record %s %s %s: %w", action, targetType, targetID, err)
	}
	return nil
}

// Entry is one row of the log.
type Entry struct {
	ID          int64           `json:"id"`
	At          time.Time       `json:"at"`
	ActorUserID *uuid.UUID      `json:"actor_user_id,omitempty"`
	Actor       string          `json:"actor"`
	Action      string          `json:"action"`
	TargetType  string          `json:"target_type"`
	TargetID    string          `json:"target_id"`
	Reason      string          `json:"reason,omitempty"`
	Details     json.RawMessage `json:"details,omitempty"`
}

// List returns the newest rows first: those at or after since (the zero time
// means from the beginning), at most limit of them.
func List(ctx context.Context, db *sql.DB, since time.Time, limit int) ([]Entry, error) {
	if limit < 1 {
		limit = 50
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, at, actor_user_id, actor_label, action, target_type, target_id,
		       COALESCE(reason, ''), details
		FROM admin_actions
		WHERE at >= $1
		ORDER BY at DESC, id DESC
		LIMIT $2`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var e Entry
		var actorID sql.NullString
		var details []byte
		if err := rows.Scan(&e.ID, &e.At, &actorID, &e.Actor, &e.Action, &e.TargetType, &e.TargetID, &e.Reason, &details); err != nil {
			return nil, err
		}
		if actorID.Valid {
			if id, err := uuid.Parse(actorID.String); err == nil {
				e.ActorUserID = &id
			}
		}
		e.Details = details
		out = append(out, e)
	}
	return out, rows.Err()
}
