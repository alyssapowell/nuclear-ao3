package moderation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/audit"
)

// This file holds what the admin screens need beyond the actions
// themselves: finding an account, ending its sessions, and the queue of
// abuse reports. Everything here is behind the moderator role. Account
// emails and ids appear only in these admin responses.

// Account is what an admin sees about one account.
type Account struct {
	ID             uuid.UUID  `json:"id"`
	Email          string     `json:"email"`
	CreatedAt      time.Time  `json:"created_at"`
	Verified       bool       `json:"is_verified"`
	Banned         bool       `json:"banned"`
	Suspended      bool       `json:"suspended"` // right now: a lapsed suspension is false
	SuspendedUntil *time.Time `json:"suspended_until,omitempty"`
	Roles          []string   `json:"roles"`
}

// FindAccount looks an account up by what an admin types: an email address
// (any letter case) or an account id.
func (s *Service) FindAccount(ctx context.Context, ref string) (*Account, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("%w: type an email address or an account id", ErrBadInput)
	}
	var a Account
	var until sql.NullTime
	var flag bool
	var roles pq.StringArray
	const cols = `SELECT u.id, u.email, u.created_at, u.is_verified, u.banned, u.suspended, u.suspended_until,
		COALESCE((SELECT array_agg(r.role ORDER BY r.role) FROM user_roles r
			WHERE r.user_id = u.id AND r.revoked_at IS NULL AND r.role <> 'user'), '{}')
		FROM users u WHERE `
	var row *sql.Row
	if id, err := uuid.Parse(ref); err == nil {
		row = s.db.QueryRowContext(ctx, cols+`u.id = $1`, id)
	} else {
		row = s.db.QueryRowContext(ctx, cols+`lower(u.email) = lower($1)`, ref)
	}
	err := row.Scan(&a.ID, &a.Email, &a.CreatedAt, &a.Verified, &a.Banned, &flag, &until, &roles)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.Roles = []string(roles)
	if a.Roles == nil {
		a.Roles = []string{}
	}
	if flag && (!until.Valid || until.Time.After(time.Now())) {
		a.Suspended = true
		if until.Valid {
			a.SuspendedUntil = &until.Time
		}
	}
	return &a, nil
}

// LogoutEverywhere ends every login session and revokes every app token of
// one account. The same rules as a suspension decide who may do it to whom.
func (s *Service) LogoutEverywhere(ctx context.Context, actor audit.Actor, id uuid.UUID, reason string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		email, _, _, err := lockUser(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := checkSanction(ctx, tx, actor, id); err != nil {
			return err
		}
		r, err := RevokeSessions(ctx, tx, &id)
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, actor, "revoke_sessions", "user", id.String(), reason,
			map[string]any{"email": email, "sessions_ended": r.Sessions, "app_tokens_revoked": r.AccessTokens})
	})
}

// AbuseReport is one report in the queue.
type AbuseReport struct {
	Kind        string     `json:"kind"` // "work" or "comment"
	ID          uuid.UUID  `json:"id"`
	TargetID    uuid.UUID  `json:"target_id"`
	WorkID      *uuid.UUID `json:"work_id,omitempty"`
	ChapterID   *uuid.UUID `json:"chapter_id,omitempty"`
	TargetTitle string     `json:"target_title,omitempty"` // a work's title, or the start of a comment
	TargetGone  bool       `json:"target_hidden"`
	Reason      string     `json:"reason"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	Resolution  string     `json:"resolution,omitempty"`
}

var reportStatuses = map[string]bool{"pending": true, "resolved": true, "dismissed": true, "escalated": true}

// ListAbuseReports returns reports about works and comments, newest first.
// kind is "", "work" or "comment"; status "" means every status.
func (s *Service) ListAbuseReports(ctx context.Context, kind, status string, limit, offset int) ([]AbuseReport, error) {
	if kind != "" && kind != "work" && kind != "comment" {
		return nil, fmt.Errorf("%w: kind must be work or comment", ErrBadInput)
	}
	if status != "" && !reportStatuses[status] {
		return nil, fmt.Errorf("%w: status must be pending, resolved, dismissed or escalated", ErrBadInput)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT kind, id, target_id, work_id, chapter_id, title, hidden, reason, description, status, created_at, reviewed_at, resolution FROM (
			SELECT 'work' AS kind, r.id, r.work_id AS target_id, r.work_id AS work_id, NULL::uuid AS chapter_id,
			       w.title AS title, (w.hidden_by_admin OR w.spam) AS hidden,
			       r.reason, COALESCE(r.description, '') AS description, COALESCE(r.status, 'pending') AS status,
			       r.created_at, r.reviewed_at, COALESCE(r.resolution, '') AS resolution
			FROM work_reports r JOIN works w ON w.id = r.work_id
			UNION ALL
			SELECT 'comment', r.id, r.comment_id,
			       COALESCE(c.work_id, ch.work_id), c.chapter_id,
			       left(c.content, 140), (c.hidden_by_admin OR c.is_spam),
			       r.reason, COALESCE(r.description, ''), COALESCE(r.status, 'pending'),
			       r.created_at, r.reviewed_at, COALESCE(r.resolution, '')
			FROM comment_reports r JOIN comments c ON c.id = r.comment_id
			LEFT JOIN chapters ch ON ch.id = c.chapter_id
		) q
		WHERE ($1 = '' OR kind = $1) AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC, id
		LIMIT $3 OFFSET $4`, kind, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AbuseReport{}
	for rows.Next() {
		var r AbuseReport
		var work, chapter uuid.NullUUID
		var reviewed sql.NullTime
		if err := rows.Scan(&r.Kind, &r.ID, &r.TargetID, &work, &chapter, &r.TargetTitle, &r.TargetGone,
			&r.Reason, &r.Description, &r.Status, &r.CreatedAt, &reviewed, &r.Resolution); err != nil {
			return nil, err
		}
		if work.Valid {
			r.WorkID = &work.UUID
		}
		if chapter.Valid {
			r.ChapterID = &chapter.UUID
		}
		if reviewed.Valid {
			r.ReviewedAt = &reviewed.Time
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CloseReport marks a report resolved or dismissed and records who did it.
// A report that already has that status is ErrNoChange.
func (s *Service) CloseReport(ctx context.Context, actor audit.Actor, kind string, id uuid.UUID, status, note string) error {
	table := map[string]string{"work": "work_reports", "comment": "comment_reports"}[kind]
	if table == "" {
		return fmt.Errorf("%w: kind must be work or comment", ErrBadInput)
	}
	if status != "resolved" && status != "dismissed" {
		return ErrBadInput
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var by any
		if actor.UserID != nil {
			by = *actor.UserID
		}
		res, err := tx.ExecContext(ctx, `UPDATE `+table+` SET status = $2, reviewed_by = $3, reviewed_at = now(), resolution = NULLIF($4, '')
			WHERE id = $1 AND COALESCE(status, 'pending') <> $2`, id, status, by, note)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return notFoundOrNoChange(ctx, tx, table, id)
		}
		action := "resolve_report"
		if status == "dismissed" {
			action = "dismiss_report"
		}
		return audit.Record(ctx, tx, actor, action, kind+"_report", id.String(), note, nil)
	})
}

func (h *handlers) findAccount(c *gin.Context) {
	a, err := h.svc.FindAccount(c.Request.Context(), c.Query("q"))
	if err != nil {
		Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"account": a})
}

func (h *handlers) logoutEverywhere(c *gin.Context) {
	actor, id, reason, ok := h.begin(c, "user_id")
	if !ok {
		return
	}
	if err := h.svc.LogoutEverywhere(c.Request.Context(), actor, id, reason); err != nil {
		Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *handlers) listAbuseReports(c *gin.Context) {
	limit, offset := 25, 0
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "limit must be 1 to 100"})
			return
		}
		limit = n
	}
	if v := c.Query("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "offset must be 0 or more"})
			return
		}
		offset = n
	}
	status := c.DefaultQuery("status", "pending")
	if status == "all" {
		status = ""
	}
	reports, err := h.svc.ListAbuseReports(c.Request.Context(), c.Query("kind"), status, limit+1, offset)
	if err != nil {
		Fail(c, err)
		return
	}
	more := len(reports) > limit
	if more {
		reports = reports[:limit]
	}
	c.JSON(http.StatusOK, gin.H{"reports": reports, "has_more": more})
}

func (h *handlers) closeReport(status string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, id, note, ok := h.begin(c, "report_id")
		if !ok {
			return
		}
		if err := h.svc.CloseReport(c.Request.Context(), actor, c.Param("kind"), id, status, note); err != nil {
			Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": status})
	}
}
