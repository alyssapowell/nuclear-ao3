package works

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/shared/models"
)

// GetPseud is the head of a pseud's page: its name, description and icon.
// Nothing here names the account or the pseud's other pseuds, and nothing
// that is the same for every pseud of an account (join date, default flag) is
// shown. "owned" is true only for the person behind the pseud at their own
// browser, so the page can offer them an edit link.
func (ws *WorkService) GetPseud(c *gin.Context) {
	pseud, _, ok := ws.loadPseudPage(c)
	if !ok {
		return
	}
	var desc, icon, alt, comment string
	if err := ws.db.QueryRowContext(c.Request.Context(), `
		SELECT coalesce(description, ''), coalesce(icon_url, ''), coalesce(icon_alt_text, ''), coalesce(icon_comment_text, '')
		FROM pseuds WHERE id = $1`, pseud.ID).Scan(&desc, &icon, &alt, &comment); err != nil {
		log.Printf("pseud page: head of %s: %v", pseud.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load pseud"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pseud": gin.H{
		"id":                pseud.ID,
		"name":              pseud.Name,
		"description":       desc,
		"icon_url":          icon,
		"icon_alt_text":     alt,
		"icon_comment_text": comment,
		"owned":             pseud.Owned,
	}})
}

// GetPseudGifts lists the works given to a pseud (AO3: /users/:user_id/gifts).
// Only gifts the recipient has not refused, that no block turned away, on
// published works the viewer may read. A work the recipient's own account
// wrote (under another pseud) is left out: it would tie the two pseuds
// together. A work that hides its creator is listed with no author.
func (ws *WorkService) GetPseudGifts(c *gin.Context) {
	pseud, viewer, ok := ws.loadPseudPage(c)
	if !ok {
		return
	}
	page, limit, offset := pageParams(c)
	ctx := c.Request.Context()

	where := ` WHERE g.pseud_id = $1 AND NOT g.rejected AND NOT g.shadowed
		AND w.status = 'published' AND can_user_view_work(w.id, $2::uuid, $3::boolean)
		AND NOT EXISTS (SELECT 1 FROM creatorships cr JOIN pseuds cp ON cp.id = cr.pseud_id
			WHERE cr.creation_id = w.id AND cr.creation_type = 'Work' AND cp.user_id = $4)`
	from := ` FROM gifts g JOIN works w ON w.id = g.work_id`
	args := []interface{}{pseud.ID, viewer, pseud.AsReader, pseud.OwnerID}

	var total int
	if err := ws.db.QueryRowContext(ctx, `SELECT COUNT(*)`+from+where, args...).Scan(&total); err != nil {
		log.Printf("pseud gifts: count for pseud %s: %v", pseud.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch gifts"})
		return
	}
	rows, err := ws.db.QueryContext(ctx, `
		SELECT w.id, w.title, COALESCE(w.summary, ''), w.rating, w.fandoms, w.characters, w.relationships,
			w.freeform_tags, w.word_count, w.chapter_count, w.is_complete, w.published_at, w.updated_at, g.created_at`+
		from+where+` ORDER BY g.created_at DESC LIMIT $5 OFFSET $6`, append(args, limit, offset)...)
	if err != nil {
		log.Printf("pseud gifts: list for pseud %s: %v", pseud.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch gifts"})
		return
	}
	defer rows.Close()

	gifts := []gin.H{}
	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		var title, summary, rating string
		var fandoms, characters, relationships, freeform pq.StringArray
		var words, chapters int
		var complete bool
		var publishedAt sql.NullTime
		var updatedAt, givenAt time.Time
		if err := rows.Scan(&id, &title, &summary, &rating, &fandoms, &characters, &relationships,
			&freeform, &words, &chapters, &complete, &publishedAt, &updatedAt, &givenAt); err != nil {
			log.Printf("pseud gifts: scan for pseud %s: %v", pseud.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan gift"})
			return
		}
		ids = append(ids, id)
		gifts = append(gifts, gin.H{
			"work": gin.H{
				"id": id, "title": title, "summary": summary, "rating": rating,
				"fandoms": []string(fandoms), "characters": []string(characters),
				"relationships": []string(relationships), "freeform_tags": []string(freeform),
				"word_count": words, "chapter_count": chapters, "is_complete": complete,
				"published_at": publishedAt.Time, "updated_at": updatedAt,
			},
			"created_at": givenAt,
		})
	}
	rows.Close()
	for i, id := range ids {
		authors, err := ws.fetchWorkAuthors(ctx, id, viewer)
		if err != nil {
			authors = []models.WorkAuthor{}
		}
		gifts[i]["work"].(gin.H)["authors"] = authors
	}
	c.JSON(http.StatusOK, gin.H{
		"pseud":      pseud.summary(),
		"gifts":      gifts,
		"pagination": pagination(page, limit, total),
	})
}
