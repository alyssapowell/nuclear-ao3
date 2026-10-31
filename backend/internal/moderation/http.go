package moderation

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/audit"
	"nuclear-ao3/internal/platform"
)

// Mount registers the admin endpoints.
//
// Hiding, spam, comment review, suspensions and bans need the moderator role
// (administrators always pass). AO3 gives these to its policy_and_abuse
// admins (app/policies/user_creation_policy.rb, comment_policy.rb,
// user_manager_policy.rb); the closest role here is moderator. Reading the
// audit log is for administrators only.
func Mount(r *gin.Engine, s *Service, d *platform.Deps) {
	h := &handlers{svc: s, db: s.db}

	mod := r.Group("/api/v1/admin", d.RequireUser, d.ActiveAccount, d.RequireRole("moderator"))
	{
		mod.POST("/works/:work_id/hide", h.hide(Work, "work_id", true))
		mod.POST("/works/:work_id/unhide", h.hide(Work, "work_id", false))
		mod.POST("/works/:work_id/spam", h.workSpam(true))
		mod.POST("/works/:work_id/unspam", h.workSpam(false))
		mod.POST("/works/:work_id/chapters/:chapter_id/hide", h.hide(Chapter, "chapter_id", true))
		mod.POST("/works/:work_id/chapters/:chapter_id/unhide", h.hide(Chapter, "chapter_id", false))
		mod.POST("/series/:series_id/hide", h.hide(Series, "series_id", true))
		mod.POST("/series/:series_id/unhide", h.hide(Series, "series_id", false))
		mod.POST("/bookmarks/:bookmark_id/hide", h.hide(Bookmark, "bookmark_id", true))
		mod.POST("/bookmarks/:bookmark_id/unhide", h.hide(Bookmark, "bookmark_id", false))
		mod.POST("/comments/:comment_id/hide", h.hide(Comment, "comment_id", true))
		mod.POST("/comments/:comment_id/unhide", h.hide(Comment, "comment_id", false))
		mod.POST("/comments/:comment_id/spam", h.commentSpam(true))
		mod.POST("/comments/:comment_id/unspam", h.commentSpam(false))
		mod.POST("/comments/:comment_id/approve", h.approveComment)
		mod.POST("/comments/:comment_id/freeze", h.freezeComment(true))
		mod.POST("/comments/:comment_id/unfreeze", h.freezeComment(false))

		mod.POST("/users/:user_id/suspend", h.suspend)
		mod.POST("/users/:user_id/unsuspend", h.userAction("lift", func(c *gin.Context, a audit.Actor, id uuid.UUID, reason string) error {
			return s.LiftSuspension(c.Request.Context(), a, id, reason)
		}))
		mod.POST("/users/:user_id/ban", h.userAction("ban", func(c *gin.Context, a audit.Actor, id uuid.UUID, reason string) error {
			return s.BanUser(c.Request.Context(), a, id, reason)
		}))
		mod.POST("/users/:user_id/unban", h.userAction("unban", func(c *gin.Context, a audit.Actor, id uuid.UUID, reason string) error {
			return s.UnbanUser(c.Request.Context(), a, id, reason)
		}))
	}

	adm := r.Group("/api/v1/admin", d.RequireUser, d.ActiveAccount, d.RequireRole())
	adm.GET("/audit", h.auditLog)
}

type handlers struct {
	svc *Service
	db  *sql.DB
}

type reasonBody struct {
	Reason string `json:"reason"`
}

// Actor works out who is making the request. Use it after RequireUser.
func Actor(c *gin.Context, db *sql.DB) (audit.Actor, bool) {
	v, ok := c.Get("user_uuid")
	id, isID := v.(uuid.UUID)
	if !ok || !isID {
		return audit.Actor{}, false
	}
	var name string
	if err := db.QueryRowContext(c.Request.Context(), `SELECT username FROM users WHERE id = $1`, id).Scan(&name); err != nil {
		return audit.Actor{}, false
	}
	return audit.UserActor(id, name), true
}

// Fail turns an error from the Service into a response.
func Fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	case errors.Is(err, ErrNoChange):
		c.JSON(http.StatusConflict, gin.H{"error": "no_change", "message": "It is already in that state."})
	case errors.Is(err, ErrIsSpam):
		c.JSON(http.StatusConflict, gin.H{"error": "work_is_spam", "message": err.Error()})
	case errors.Is(err, ErrProtectedAcc):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "protected_account", "message": err.Error()})
	case errors.Is(err, ErrBadInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
	}
}

func (h *handlers) begin(c *gin.Context, param string) (audit.Actor, uuid.UUID, string, bool) {
	actor, ok := Actor(c, h.db)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return actor, uuid.Nil, "", false
	}
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid id"})
		return actor, uuid.Nil, "", false
	}
	var body reasonBody
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid JSON"})
			return actor, uuid.Nil, "", false
		}
	}
	return actor, id, body.Reason, true
}

func (h *handlers) hide(kind Kind, param string, hidden bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, id, reason, ok := h.begin(c, param)
		if !ok {
			return
		}
		if err := h.svc.SetHidden(c.Request.Context(), actor, kind, id, hidden, reason); err != nil {
			Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"hidden": hidden})
	}
}

func (h *handlers) workSpam(spam bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, id, reason, ok := h.begin(c, "work_id")
		if !ok {
			return
		}
		if err := h.svc.SetWorkSpam(c.Request.Context(), actor, id, spam, reason); err != nil {
			Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"spam": spam, "hidden": spam})
	}
}

func (h *handlers) commentSpam(spam bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, id, reason, ok := h.begin(c, "comment_id")
		if !ok {
			return
		}
		if err := h.svc.SetCommentSpam(c.Request.Context(), actor, id, spam, reason); err != nil {
			Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"spam": spam})
	}
}

func (h *handlers) approveComment(c *gin.Context) {
	actor, id, reason, ok := h.begin(c, "comment_id")
	if !ok {
		return
	}
	if err := h.svc.ApproveComment(c.Request.Context(), actor, id, reason); err != nil {
		Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"approved": true})
}

func (h *handlers) freezeComment(frozen bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, id, reason, ok := h.begin(c, "comment_id")
		if !ok {
			return
		}
		if err := h.svc.SetCommentFrozen(c.Request.Context(), actor, id, frozen, reason); err != nil {
			Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"frozen": frozen})
	}
}

func (h *handlers) suspend(c *gin.Context) {
	actor, ok := Actor(c, h.db)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	id, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid id"})
		return
	}
	var body struct {
		Until  string `json:"until"` // RFC 3339, or a date such as 2026-12-01
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid JSON"})
		return
	}
	until, err := ParseUntil(body.Until)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}
	if err := h.svc.SuspendUser(c.Request.Context(), actor, id, until, body.Reason); err != nil {
		Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"suspended_until": until.UTC().Format(time.RFC3339)})
}

// ParseUntil reads the end of a suspension: a full RFC 3339 time, or a date
// like 2026-12-01 meaning the end of that day in UTC.
func ParseUntil(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if d, err := time.Parse("2006-01-02", s); err == nil {
		return d.Add(24*time.Hour - time.Second), nil
	}
	return time.Time{}, errors.New("until must be a date like 2026-12-01 or a time like 2026-12-01T18:00:00Z")
}

func (h *handlers) userAction(_ string, do func(c *gin.Context, a audit.Actor, id uuid.UUID, reason string) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, id, reason, ok := h.begin(c, "user_id")
		if !ok {
			return
		}
		if err := do(c, actor, id, reason); err != nil {
			Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func (h *handlers) auditLog(c *gin.Context) {
	var since time.Time
	if v := c.Query("since"); v != "" {
		t, err := ParseSince(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
			return
		}
		since = t
	}
	limit := 50
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "limit must be 1 to 1000"})
			return
		}
		limit = n
	}
	entries, err := audit.List(c.Request.Context(), h.db, since, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"actions": entries})
}

// ParseSince reads the start of an audit listing: a date like 2026-10-01 or
// a full RFC 3339 time.
func ParseSince(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if d, err := time.Parse("2006-01-02", s); err == nil {
		return d, nil
	}
	return time.Time{}, errors.New("since must be a date like 2026-10-01 or a time like 2026-10-01T00:00:00Z")
}
