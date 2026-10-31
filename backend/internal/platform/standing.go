package platform

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Standing says whether an account may write to the archive.
type Standing struct {
	Banned         bool
	Suspended      bool       // a suspension that has not ended yet
	SuspendedUntil *time.Time // when it ends; nil for a ban or a suspension with no end
}

// AccountStanding reads a user's ban and suspension state. A suspension
// whose end time has passed does not count (the same rule as the
// user_standing() SQL function), so suspensions end on time with no job
// running.
func AccountStanding(ctx context.Context, db *sql.DB, id uuid.UUID) (Standing, error) {
	var banned, suspended bool
	var until sql.NullTime
	err := db.QueryRowContext(ctx,
		`SELECT banned, suspended, suspended_until FROM users WHERE id = $1`, id).Scan(&banned, &suspended, &until)
	if err == sql.ErrNoRows {
		return Standing{}, nil
	}
	if err != nil {
		return Standing{}, err
	}
	s := Standing{Banned: banned}
	if suspended && (!until.Valid || until.Time.After(time.Now())) {
		s.Suspended = true
		if until.Valid {
			s.SuspendedUntil = &until.Time
		}
	}
	return s, nil
}

// ActiveAccount refuses banned and suspended users. AO3 does the same for
// creating and changing works, chapters, comments, bookmarks, series,
// collections, pseuds, profiles and kudos (check_user_status in
// app/controllers/application_controller.rb:431). Requests with no
// logged-in user pass through, so it is safe on routes that allow guests.
func (d *Deps) ActiveAccount(c *gin.Context) {
	d.refuse(c, true)
}

// NotSuspended refuses suspended users but lets banned users through. AO3
// uses it for deleting and orphaning works and chapters
// (check_user_not_suspended, application_controller.rb:452), so that someone
// cannot delete their work to avoid a suspension.
func (d *Deps) NotSuspended(c *gin.Context) {
	d.refuse(c, false)
}

func (d *Deps) refuse(c *gin.Context, alsoBanned bool) {
	v, ok := c.Get("user_uuid")
	if !ok {
		c.Next()
		return
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		c.Next()
		return
	}
	st, err := AccountStanding(c.Request.Context(), d.DB, id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not check the account"})
		return
	}
	switch {
	case st.Suspended:
		body := gin.H{"error": "account_suspended",
			"message": "This account is suspended. You can still read, but you can't post or change anything until the suspension ends."}
		if st.SuspendedUntil != nil {
			body["suspended_until"] = st.SuspendedUntil.UTC().Format(time.RFC3339)
		}
		c.AbortWithStatusJSON(http.StatusForbidden, body)
	case st.Banned && alsoBanned:
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "account_banned",
			"message": "This account has been banned. You can still read, but you can't post or change anything."})
	default:
		c.Next()
	}
}
