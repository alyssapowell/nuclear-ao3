package support

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/auth"
	"nuclear-ao3/internal/moderation"
	"nuclear-ao3/internal/platform"
)

// Sending is limited per address (or per account, when logged in) so the form
// cannot be used to fill the support inbox or to send mail to other people.
const (
	DefaultSendLimit  = 5
	DefaultSendWindow = time.Hour
)

// Option changes how Mount behaves.
type Option func(*handlers)

// WithSendLimit sets how many tickets one sender may send per window.
func WithSendLimit(n int, window time.Duration) Option {
	return func(h *handlers) { h.limit, h.window = n, window }
}

type handlers struct {
	svc     *Service
	deps    *platform.Deps
	limiter *auth.Limiter
	limit   int
	window  time.Duration
}

// Mount registers POST /api/v1/support (guests and logged-in people) and the
// admin routes under /api/v1/admin/support (moderators and administrators).
//
// AO3 gives its policy_and_abuse admins the support tickets; the closest role
// here is moderator, as for the other admin routes. Apps (OAuth clients) can
// use neither: see platform/appaccess.go.
func Mount(r *gin.Engine, s *Service, d *platform.Deps, opts ...Option) {
	h := &handlers{svc: s, deps: d, limiter: auth.NewLimiter(), limit: DefaultSendLimit, window: DefaultSendWindow}
	for _, o := range opts {
		o(h)
	}
	r.POST("/api/v1/support", d.OptionalUser, h.submit)

	adm := r.Group("/api/v1/admin/support", d.RequireUser, d.ActiveAccount, d.RequireRole("moderator"))
	adm.GET("", h.list)
	adm.GET("/:ticket_id", h.get)
	adm.POST("/:ticket_id/notes", h.note)
	adm.POST("/:ticket_id/close", h.close(true))
	adm.POST("/:ticket_id/reopen", h.close(false))
}

func (h *handlers) submit(c *gin.Context) {
	var user *uuid.UUID
	if v, ok := c.Get("user_uuid"); ok {
		if id, ok := v.(uuid.UUID); ok {
			user = &id
		}
	}
	// A request body is read only up to a little more than the longest
	// message, so a huge one is refused without being held in memory.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
	var sub Submission
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "The message could not be read."})
		return
	}
	// Count the attempt before doing any work. Both the address and the
	// account count, so neither can be used to get round the other.
	keys := []string{"support:ip:" + c.ClientIP()}
	if user != nil {
		keys = append(keys, "support:user:"+user.String())
	}
	for _, k := range keys {
		if ok, wait := h.limiter.Allow(k, h.limit, h.window); !ok {
			secs := int(wait.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(secs))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited",
				"message": "You have sent several messages recently. Please try again later."})
			return
		}
	}
	if _, err := h.svc.Submit(c.Request.Context(), user, sub); err != nil {
		if errors.Is(err, ErrBadInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": messageOf(err)})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	// Nothing about the ticket or the sender comes back.
	c.JSON(http.StatusCreated, gin.H{"received": true})
}

func messageOf(err error) string {
	s := err.Error()
	if p := ErrBadInput.Error() + ": "; len(s) > len(p) && s[:len(p)] == p {
		return s[len(p):]
	}
	return s
}

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrBadInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": messageOf(err)})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	case errors.Is(err, ErrNoChange):
		c.JSON(http.StatusConflict, gin.H{"error": "no_change", "message": "It is already in that state."})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
	}
}

func (h *handlers) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	tickets, total, err := h.svc.List(c.Request.Context(), ListFilter{
		Status: c.Query("status"), Kind: c.Query("kind"), Limit: limit, Offset: offset,
	})
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"tickets": tickets, "total": total})
}

func (h *handlers) ticketID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("ticket_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid id"})
		return uuid.Nil, false
	}
	return id, true
}

func (h *handlers) get(c *gin.Context) {
	id, ok := h.ticketID(c)
	if !ok {
		return
	}
	t, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *handlers) note(c *gin.Context) {
	id, ok := h.ticketID(c)
	if !ok {
		return
	}
	actor, ok := moderation.Actor(c, h.deps.DB)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	var body struct {
		Body  string `json:"body"`
		Reply bool   `json:"reply"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid JSON"})
		return
	}
	if err := h.svc.AddNote(c.Request.Context(), actor, id, body.Body, body.Reply); err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"added": true, "reply": body.Reply})
}

func (h *handlers) close(closed bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := h.ticketID(c)
		if !ok {
			return
		}
		actor, ok := moderation.Actor(c, h.deps.DB)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
			return
		}
		var body struct {
			Reason string `json:"reason"`
		}
		if c.Request.ContentLength != 0 {
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid JSON"})
				return
			}
		}
		if err := h.svc.SetClosed(c.Request.Context(), actor, id, closed, body.Reason); err != nil {
			fail(c, err)
			return
		}
		status := "open"
		if closed {
			status = "closed"
		}
		c.JSON(http.StatusOK, gin.H{"status": status})
	}
}
