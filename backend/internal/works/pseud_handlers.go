package works

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/html"
	"nuclear-ao3/shared/models"
)

// A user's pseuds, following AO3 (app/models/pseud.rb, pseuds_controller.rb).
// These endpoints only ever show a user their own pseuds. Everywhere else the
// API names a pseud by {id, name}.

// pseudRequest is what creating or changing a pseud accepts. A field that
// isn't sent is left alone when changing a pseud.
type pseudRequest struct {
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	IsDefault       *bool   `json:"is_default"`
	IconURL         *string `json:"icon_url"` // where the icon is; uploading one is not built yet
	IconAltText     *string `json:"icon_alt_text"`
	IconCommentText *string `json:"icon_comment_text"`
}

const pseudColumns = `id, user_id, name, coalesce(description, ''), is_default, coalesce(icon_url, ''),
	icon_alt_text, icon_comment_text, created_at, updated_at`

type scanner interface{ Scan(dest ...any) error }

func scanPseud(row scanner) (models.Pseud, error) {
	var p models.Pseud
	err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.IsDefault, &p.IconURL,
		&p.IconAltText, &p.IconCommentText, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

// isUniqueViolation reports whether err is Postgres refusing a duplicate.
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

// CreatePseud adds a pseud to the signed-in user's account. Making it the
// default demotes the old default.
func (ws *WorkService) CreatePseud(c *gin.Context) {
	userID := ws.getUserIDFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	var req pseudRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}
	name := strings.TrimSpace(*req.Name)
	if err := models.ValidatePseudName(name); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	desc, alt, comment, icon := deref(req.Description), deref(req.IconAltText), deref(req.IconCommentText), deref(req.IconURL)
	if err := models.ValidatePseudText(desc, alt, comment, icon); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	tx, err := ws.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pseud"})
		return
	}
	defer tx.Rollback()
	makeDefault := req.IsDefault != nil && *req.IsDefault
	if makeDefault {
		if _, err := tx.Exec(`UPDATE pseuds SET is_default = false, updated_at = now()
			WHERE user_id = $1 AND is_default`, *userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pseud"})
			return
		}
	}
	p, err := scanPseud(tx.QueryRow(`
		INSERT INTO pseuds (user_id, name, description, is_default, icon_url, icon_alt_text, icon_comment_text)
		VALUES ($1, $2, $3, $4, nullif($5, ''), $6, $7)
		RETURNING `+pseudColumns, *userID, name, html.Clean(desc), makeDefault, icon, alt, comment))
	if isUniqueViolation(err) {
		c.JSON(http.StatusConflict, gin.H{"error": "You already have a pseud with that name"})
		return
	}
	if err != nil {
		log.Printf("create pseud: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pseud"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pseud"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"pseud": p})
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// GetUserPseuds lists the signed-in user's pseuds, default first.
func (ws *WorkService) GetUserPseuds(c *gin.Context) {
	userID := ws.getUserIDFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	rows, err := ws.db.Query(`SELECT `+pseudColumns+` FROM pseuds WHERE user_id = $1
		ORDER BY is_default DESC, lower(name)`, *userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pseuds"})
		return
	}
	defer rows.Close()
	pseuds := []models.Pseud{}
	for rows.Next() {
		p, err := scanPseud(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan pseud"})
			return
		}
		pseuds = append(pseuds, p)
	}
	c.JSON(http.StatusOK, gin.H{"pseuds": pseuds})
}

// UpdatePseud renames or describes one of the signed-in user's pseuds, or
// makes it the default. As in AO3, the old default is demoted when another
// pseud becomes the default; asking for the default to stop being the
// default hands the role to the pseud named after the username.
func (ws *WorkService) UpdatePseud(c *gin.Context) {
	userID := ws.getUserIDFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	id, err := uuid.Parse(c.Param("pseud_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pseud ID"})
		return
	}
	var req pseudRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	tx, err := ws.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pseud"})
		return
	}
	defer tx.Rollback()

	// Lock the user's pseuds so two changes of default can't cross.
	cur, err := scanPseud(tx.QueryRow(`SELECT `+pseudColumns+` FROM pseuds
		WHERE id = $1 AND user_id = $2 FOR UPDATE`, id, *userID))
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pseud not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pseud"})
		return
	}

	name, desc, alt, comment, icon := cur.Name, cur.Description, cur.IconAltText, cur.IconCommentText, cur.IconURL
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		if err := models.ValidatePseudName(name); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
	}
	if req.Description != nil {
		desc = *req.Description
	}
	if req.IconAltText != nil {
		alt = *req.IconAltText
	}
	if req.IconCommentText != nil {
		comment = *req.IconCommentText
	}
	if req.IconURL != nil {
		icon = *req.IconURL
	}
	if err := models.ValidatePseudText(desc, alt, comment, icon); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	if req.Description != nil {
		desc = html.Clean(desc)
	}

	isDefault := cur.IsDefault
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}
	switch {
	case isDefault && !cur.IsDefault:
		if _, err := tx.Exec(`UPDATE pseuds SET is_default = false, updated_at = now()
			WHERE user_id = $1 AND is_default`, *userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pseud"})
			return
		}
	case !isDefault && cur.IsDefault:
		// The pseud named after the username takes over; if this is it, or
		// there isn't one, the default stays where it is.
		var heir uuid.UUID
		err := tx.QueryRow(`SELECT p.id FROM pseuds p JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $1 AND p.id <> $2 AND lower(p.name) = lower(u.username::text)`,
			*userID, id).Scan(&heir)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pseud"})
			return
		}
		if err == sql.ErrNoRows {
			isDefault = true
			break
		}
		// One default at a time: this pseud steps down before the heir steps up.
		if _, err := tx.Exec(`UPDATE pseuds SET is_default = false, updated_at = now() WHERE id = $1`, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pseud"})
			return
		}
		if _, err := tx.Exec(`UPDATE pseuds SET is_default = true, updated_at = now() WHERE id = $1`, heir); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pseud"})
			return
		}
	}

	p, err := scanPseud(tx.QueryRow(`
		UPDATE pseuds SET name = $3, description = $4, is_default = $5, icon_url = nullif($6, ''),
			icon_alt_text = $7, icon_comment_text = $8, updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING `+pseudColumns, id, *userID, name, desc, isDefault, icon, alt, comment))
	if isUniqueViolation(err) {
		c.JSON(http.StatusConflict, gin.H{"error": "You already have a pseud with that name"})
		return
	}
	if err != nil {
		log.Printf("update pseud: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pseud"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pseud"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pseud": p})
}

// DeletePseud deletes one of the signed-in user's pseuds. AO3 refuses to
// delete the default pseud or the one named after the username. Everything
// the pseud was credited on moves to the default pseud first (Pseud#
// replace_me_with_default): works and series it created, comments it wrote,
// and gifts it received.
func (ws *WorkService) DeletePseud(c *gin.Context) {
	userID := ws.getUserIDFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	id, err := uuid.Parse(c.Param("pseud_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pseud ID"})
		return
	}
	tx, err := ws.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pseud"})
		return
	}
	defer tx.Rollback()

	var isDefault, matchesUsername bool
	err = tx.QueryRow(`SELECT p.is_default, lower(p.name) = lower(u.username::text)
		FROM pseuds p JOIN users u ON u.id = p.user_id
		WHERE p.id = $1 AND p.user_id = $2 FOR UPDATE OF p`, id, *userID).Scan(&isDefault, &matchesUsername)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pseud not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pseud"})
		return
	}
	if isDefault {
		c.JSON(http.StatusConflict, gin.H{"error": "You cannot delete your default pseudonym, sorry!"})
		return
	}
	if matchesUsername {
		c.JSON(http.StatusConflict, gin.H{"error": "You cannot delete the pseud that matches your username, sorry!"})
		return
	}
	var def uuid.UUID
	if err := tx.QueryRow(`SELECT id FROM pseuds WHERE user_id = $1 AND is_default`, *userID).Scan(&def); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pseud"})
		return
	}

	steps := []string{
		// Credit on works, chapters and series. If the default pseud already
		// has a credit on the same thing, the two merge: the default is
		// approved if either was, and this pseud's credit goes.
		`UPDATE creatorships d SET approved = d.approved OR o.approved, updated_at = now()
			FROM creatorships o
			WHERE o.pseud_id = $1 AND d.pseud_id = $2
			  AND d.creation_id = o.creation_id AND d.creation_type = o.creation_type`,
		`DELETE FROM creatorships o USING creatorships d
			WHERE o.pseud_id = $1 AND d.pseud_id = $2
			  AND d.creation_id = o.creation_id AND d.creation_type = o.creation_type`,
		`UPDATE creatorships SET pseud_id = $2, updated_at = now() WHERE pseud_id = $1`,
		// Comments keep their text; the timestamp changes so caches refresh.
		`UPDATE comments SET pseudonym_id = $2, updated_at = now() WHERE pseudonym_id = $1`,
		`UPDATE comment_kudos SET pseudonym_id = $2 WHERE pseudonym_id = $1`,
		`UPDATE gifts SET pseud_id = $2, updated_at = now() WHERE pseud_id = $1`,
	}
	for _, q := range steps {
		if _, err := tx.Exec(q, id, def); err != nil {
			log.Printf("delete pseud %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pseud"})
			return
		}
	}
	if _, err := tx.Exec(`DELETE FROM pseuds WHERE id = $1`, id); err != nil {
		log.Printf("delete pseud %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pseud"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pseud"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pseud deleted. Its works and comments are now under your default pseud.", "moved_to": def})
}
