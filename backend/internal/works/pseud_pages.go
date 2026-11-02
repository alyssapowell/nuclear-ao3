package works

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/platform"
	"nuclear-ao3/shared/models"
)

// A pseud's page, as in AO3's /users/:user_id/pseuds/:id/works, /series and
// /bookmarks. The lists are keyed by pseud and say nothing about the account
// behind it. The pseud's other pseuds are not listed, so no page ties two
// pseuds of one person together.

// pseudPage is the pseud a page is about and whether the viewer owns it.
type pseudPage struct {
	ID      uuid.UUID
	Name    string
	OwnerID uuid.UUID
	Default bool
	Owned   bool // the viewer is the person behind this pseud
}

func (p pseudPage) summary() models.PseudSummary {
	return models.PseudSummary{ID: p.ID.String(), Name: p.Name}
}

// loadPseudPage reads :pseud_id and answers 400 or 404 itself when it can't.
func (ws *WorkService) loadPseudPage(c *gin.Context) (pseudPage, *uuid.UUID, bool) {
	var p pseudPage
	id, err := uuid.Parse(c.Param("pseud_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pseud ID"})
		return p, nil, false
	}
	err = ws.db.QueryRowContext(c.Request.Context(),
		`SELECT id, name, user_id, is_default FROM pseuds WHERE id = $1`, id).
		Scan(&p.ID, &p.Name, &p.OwnerID, &p.Default)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pseud not found"})
		return p, nil, false
	}
	if err != nil {
		log.Printf("pseud page: load pseud %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load pseud"})
		return p, nil, false
	}
	viewer := ws.getUserIDFromContext(c)
	p.Owned = viewer != nil && *viewer == p.OwnerID
	// A page shows its owner extra (drafts, works that hide their creator).
	// An app gets the page as any other reader would see it, or the
	// difference would tell the app which pseuds belong to the person.
	if p.Owned && platform.IsApp(c) {
		viewer, p.Owned = nil, false
	}
	return p, viewer, true
}

// pageParams reads page and limit (default 20, at most 100).
func pageParams(c *gin.Context) (page, limit, offset int) {
	page, limit = 1, 20
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	return page, limit, (page - 1) * limit
}

func pagination(page, limit, total int) gin.H {
	return gin.H{"page": page, "limit": limit, "total": total, "total_pages": (total + limit - 1) / limit}
}

// GetPseudWorks lists the works credited to a pseud.
func (ws *WorkService) GetPseudWorks(c *gin.Context) {
	pseud, viewer, ok := ws.loadPseudPage(c)
	if !ok {
		return
	}
	page, limit, offset := pageParams(c)

	// Its owner sees everything under it, drafts included. Anyone else sees
	// published works they may read, and as in AO3 never a work that hides
	// its creator: listing it here would say who wrote it.
	where := ` WHERE cr.pseud_id = $1 AND cr.approved = true`
	args := []interface{}{pseud.ID}
	if !pseud.Owned {
		where += " AND w.status = 'published' AND can_user_view_work(w.id, $2::uuid)" + hidesNoCreator
		args = append(args, viewer)
	}
	from := ` FROM works w JOIN creatorships cr ON w.id = cr.creation_id AND cr.creation_type = 'Work'`

	var total int
	if err := ws.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*)`+from+where, args...).Scan(&total); err != nil {
		log.Printf("pseud works: count for pseud %s: %v", pseud.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pseud works"})
		return
	}

	n := len(args)
	rows, err := ws.db.QueryContext(c.Request.Context(), `
		SELECT w.id, w.title, w.summary, w.language, w.rating,
			w.category, w.warnings, w.fandoms, w.characters, w.relationships, w.freeform_tags,
			w.word_count, w.chapter_count, w.max_chapters, w.is_complete, w.status,
			w.published_at, w.updated_at, w.created_at,
			COALESCE(w.hit_count, 0),
			COALESCE(w.kudos_count, 0) + own_shadowed_kudos(w.id, $`+strconv.Itoa(n+3)+`::uuid),
			COALESCE(w.comment_count, 0) + own_shadowed_comments(w.id, $`+strconv.Itoa(n+3)+`::uuid),
			COALESCE(w.bookmark_count, 0)`+from+where+
		` ORDER BY w.updated_at DESC LIMIT $`+strconv.Itoa(n+1)+` OFFSET $`+strconv.Itoa(n+2),
		append(args, limit, offset, viewer)...)
	if err != nil {
		log.Printf("pseud works: list for pseud %s: %v", pseud.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pseud works"})
		return
	}
	defer rows.Close()

	works := []models.Work{}
	for rows.Next() {
		var work models.Work
		var categoryStr, warningsStr, summary, status sql.NullString
		var fandoms, characters, relationships, freeformTags pq.StringArray
		var publishedAt sql.NullTime
		var maxChapters sql.NullInt64

		if err := rows.Scan(
			&work.ID, &work.Title, &summary,
			&work.Language, &work.Rating, &categoryStr, &warningsStr,
			&fandoms, &characters, &relationships, &freeformTags,
			&work.WordCount, &work.ChapterCount, &maxChapters,
			&work.IsComplete, &status, &publishedAt, &work.UpdatedAt, &work.CreatedAt,
			&work.Hits, &work.Kudos, &work.Comments, &work.Bookmarks); err != nil {
			log.Printf("pseud works: scan for pseud %s: %v", pseud.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan work"})
			return
		}
		work.Summary = summary.String
		work.Status = status.String
		if publishedAt.Valid {
			work.PublishedAt = &publishedAt.Time
		}
		if maxChapters.Valid {
			m := int(maxChapters.Int64)
			work.MaxChapters = &m
		}
		if categoryStr.Valid && categoryStr.String != "" {
			work.Category = []string{categoryStr.String}
		}
		if warningsStr.Valid && warningsStr.String != "" {
			work.Warnings = []string{warningsStr.String}
		}
		work.Fandoms = []string(fandoms)
		work.Characters = []string(characters)
		work.Relationships = []string(relationships)
		work.FreeformTags = []string(freeformTags)
		works = append(works, work)
	}

	c.JSON(http.StatusOK, gin.H{
		"pseud":      pseud.summary(),
		"works":      works,
		"pagination": pagination(page, limit, total),
	})
}

// pseudSeriesWhere picks the series a pseud belongs to: those with a work
// the pseud is credited on (and may be shown on), and an empty series made by
// the person behind the pseud, which goes under their default pseud. $1 is the
// pseud, $2 the viewer, $3 the owner's account, $4 whether the pseud is the
// owner's default and $5 whether the viewer owns the pseud.
const pseudSeriesWhere = `
	WHERE can_user_view_series(s.id, $2::uuid)
	  AND series_shows_creator(s.id, $2::uuid)
	  AND (EXISTS (SELECT 1 FROM series_works sw
	               JOIN creatorships cr ON cr.creation_id = sw.work_id AND cr.creation_type = 'Work' AND cr.approved
	               WHERE sw.series_id = s.id AND cr.pseud_id = $1
	                 AND can_user_view_work(sw.work_id, $2::uuid)
	                 AND (NOT work_hides_creator(sw.work_id) OR $5::bool))
	       OR ($4::bool AND s.user_id = $3 AND NOT EXISTS (SELECT 1 FROM series_works sw WHERE sw.series_id = s.id)))`

// GetPseudSeries lists the series a pseud has works in.
func (ws *WorkService) GetPseudSeries(c *gin.Context) {
	pseud, viewer, ok := ws.loadPseudPage(c)
	if !ok {
		return
	}
	page, limit, offset := pageParams(c)
	ctx := c.Request.Context()
	args := []interface{}{pseud.ID, viewer, pseud.OwnerID, pseud.Default, pseud.Owned}

	var total int
	if err := ws.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM series s`+pseudSeriesWhere, args...).Scan(&total); err != nil {
		log.Printf("pseud series: count for pseud %s: %v", pseud.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pseud series"})
		return
	}
	rows, err := ws.db.QueryContext(ctx, `
		SELECT s.id, s.title, COALESCE(s.description, ''), COALESCE(s.notes, ''), s.user_id, s.is_complete,
			(SELECT COUNT(*) FROM series_works sw JOIN works vw ON vw.id = sw.work_id
			 WHERE sw.series_id = s.id AND can_user_view_work(vw.id, $2::uuid)
			   AND (NOT work_hides_creator(vw.id) OR is_work_creator(vw.id, $2::uuid))),
			s.created_at, s.updated_at
		FROM series s`+pseudSeriesWhere+` ORDER BY s.updated_at DESC LIMIT $6 OFFSET $7`,
		append(args, limit, offset)...)
	if err != nil {
		log.Printf("pseud series: list for pseud %s: %v", pseud.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pseud series"})
		return
	}
	defer rows.Close()

	series := []models.Series{}
	for rows.Next() {
		var s models.Series
		if err := rows.Scan(&s.ID, &s.Title, &s.Summary, &s.Notes, &s.UserID, &s.IsComplete,
			&s.WorkCount, &s.CreatedAt, &s.UpdatedAt); err != nil {
			log.Printf("pseud series: scan for pseud %s: %v", pseud.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan series"})
			return
		}
		series = append(series, s)
	}
	rows.Close()
	ptrs := make([]*models.Series, len(series))
	for i := range series {
		ptrs[i] = &series[i]
	}
	ws.fillSeriesWordCounts(ctx, ptrs, viewer)
	ws.attachSeriesAuthors(ctx, ptrs, viewer)

	c.JSON(http.StatusOK, gin.H{
		"pseud":      pseud.summary(),
		"series":     series,
		"pagination": pagination(page, limit, total),
	})
}

// GetPseudBookmarks lists the bookmarks made under a pseud that the viewer
// may see.
func (ws *WorkService) GetPseudBookmarks(c *gin.Context) {
	pseud, viewer, ok := ws.loadPseudPage(c)
	if !ok {
		return
	}
	page, limit, offset := pageParams(c)
	ctx := c.Request.Context()

	// Private and hidden bookmarks, and those of works the viewer can't read,
	// go through can_user_view_bookmark. A creator's bookmark of their own work
	// that hides its creator would tie the pseud to the work, so strangers
	// don't see it.
	const where = ` WHERE b.pseud_id = $1 AND can_user_view_bookmark(b.id, $2::uuid)
		AND (b.user_id = $2::uuid OR NOT work_hides_creator(b.work_id) OR NOT is_work_creator(b.work_id, b.user_id))`

	var total int
	if err := ws.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bookmarks b`+where, pseud.ID, viewer).Scan(&total); err != nil {
		log.Printf("pseud bookmarks: count for pseud %s: %v", pseud.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookmarks"})
		return
	}
	rows, err := ws.db.QueryContext(ctx, `
		SELECT b.id, b.work_id, COALESCE(b.notes, ''), COALESCE(b.tags, '{}'), b.is_private, b.created_at, b.updated_at,
			w.title, COALESCE(w.summary, ''), w.rating, w.fandoms, w.characters, w.relationships,
			w.freeform_tags, w.word_count, w.chapter_count, w.is_complete, w.status,
			w.published_at, w.updated_at,
			COALESCE(w.hit_count, 0),
			COALESCE(w.kudos_count, 0) + own_shadowed_kudos(w.id, $2::uuid),
			COALESCE(w.comment_count, 0) + own_shadowed_comments(w.id, $2::uuid),
			COALESCE(w.bookmark_count, 0)
		FROM bookmarks b JOIN works w ON w.id = b.work_id`+where+
		` ORDER BY b.created_at DESC LIMIT $3 OFFSET $4`, pseud.ID, viewer, limit, offset)
	if err != nil {
		log.Printf("pseud bookmarks: list for pseud %s: %v", pseud.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookmarks"})
		return
	}
	defer rows.Close()

	bookmarks := []gin.H{}
	for rows.Next() {
		var b models.Bookmark
		var w models.Work
		var fandoms, characters, relationships, freeformTags, tags pq.StringArray
		var status sql.NullString
		if err := rows.Scan(
			&b.ID, &b.WorkID, &b.Notes, &tags, &b.IsPrivate, &b.CreatedAt, &b.UpdatedAt,
			&w.Title, &w.Summary, &w.Rating, &fandoms, &characters, &relationships, &freeformTags,
			&w.WordCount, &w.ChapterCount, &w.IsComplete, &status, &w.PublishedAt, &w.UpdatedAt,
			&w.Hits, &w.Kudos, &w.Comments, &w.Bookmarks); err != nil {
			log.Printf("pseud bookmarks: scan for pseud %s: %v", pseud.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan bookmark"})
			return
		}
		bookmarks = append(bookmarks, gin.H{
			"id":         b.ID,
			"work_id":    b.WorkID,
			"notes":      b.Notes,
			"tags":       []string(tags),
			"is_private": b.IsPrivate,
			"created_at": b.CreatedAt,
			"updated_at": b.UpdatedAt,
			"pseud":      pseud.summary(),
			"work": gin.H{
				"id":            b.WorkID,
				"title":         w.Title,
				"summary":       w.Summary,
				"rating":        w.Rating,
				"fandoms":       []string(fandoms),
				"characters":    []string(characters),
				"relationships": []string(relationships),
				"freeform_tags": []string(freeformTags),
				"word_count":    w.WordCount,
				"chapter_count": w.ChapterCount,
				"is_complete":   w.IsComplete,
				"status":        status.String,
				"published_at":  w.PublishedAt,
				"updated_at":    w.UpdatedAt,
				"hits":          w.Hits,
				"kudos":         w.Kudos,
				"comments":      w.Comments,
				"bookmarks":     w.Bookmarks,
			},
		})
	}
	rows.Close()
	// Authors come after the list is read: get_work_authors names nobody for
	// a work that hides its creator.
	for _, b := range bookmarks {
		workID := b["work_id"].(uuid.UUID)
		authors, err := ws.fetchWorkAuthors(ctx, workID, viewer)
		if err != nil {
			authors = []models.WorkAuthor{}
		}
		b["work"].(gin.H)["authors"] = authors
	}

	c.JSON(http.StatusOK, gin.H{
		"pseud":      pseud.summary(),
		"bookmarks":  bookmarks,
		"pagination": pagination(page, limit, total),
	})
}

// fillSeriesWordCounts sets each series' word count from the works the viewer
// may see, leaving out a hidden creator's work unless the viewer wrote it.
func (ws *WorkService) fillSeriesWordCounts(ctx context.Context, series []*models.Series, viewer *uuid.UUID) {
	for _, s := range series {
		if err := ws.db.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(w.word_count), 0)
			FROM works w JOIN series_works sw ON w.id = sw.work_id
			WHERE sw.series_id = $1 AND w.status != 'draft' AND can_user_view_work(w.id, $2::uuid)
			  AND (NOT work_hides_creator(w.id) OR is_work_creator(w.id, $2::uuid))`, s.ID, viewer).Scan(&s.WordCount); err != nil {
			s.WordCount = 0
		}
	}
}

// attachSeriesAuthors sets the creators of each series as pseuds, the way AO3
// does (Series#allpseuds: the pseuds of its works' creators, in name order).
// A pseud is included only for works the viewer may see, and a work that hides
// its creator adds nobody unless the pseud is the viewer's own. A series with
// no works yet is credited to its owner's default pseud.
func (ws *WorkService) attachSeriesAuthors(ctx context.Context, series []*models.Series, viewer *uuid.UUID) {
	if len(series) == 0 {
		return
	}
	ids := make([]string, len(series))
	byID := make(map[uuid.UUID]*models.Series, len(series))
	for i, s := range series {
		ids[i] = s.ID.String()
		byID[s.ID] = s
		s.Authors = []models.WorkAuthor{}
	}
	rows, err := ws.db.QueryContext(ctx, `
		SELECT sw.series_id, p.id, p.name
		FROM series_works sw
		JOIN creatorships cr ON cr.creation_id = sw.work_id AND cr.creation_type = 'Work' AND cr.approved
		JOIN pseuds p ON p.id = cr.pseud_id
		WHERE sw.series_id = ANY($1::uuid[])
		  AND can_user_view_work(sw.work_id, $2::uuid)
		  AND (NOT work_hides_creator(sw.work_id) OR p.user_id = $2::uuid)
		GROUP BY sw.series_id, p.id, p.name
		UNION ALL
		SELECT s.id, p.id, p.name
		FROM series s JOIN pseuds p ON p.user_id = s.user_id AND p.is_default
		WHERE s.id = ANY($1::uuid[]) AND NOT EXISTS (SELECT 1 FROM series_works sw WHERE sw.series_id = s.id)
		ORDER BY 3`, pq.Array(ids), viewer)
	if err != nil {
		log.Printf("series authors: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var seriesID, pseudID uuid.UUID
		var name string
		if err := rows.Scan(&seriesID, &pseudID, &name); err != nil {
			log.Printf("series authors: scan: %v", err)
			return
		}
		if s := byID[seriesID]; s != nil {
			id := pseudID
			s.Authors = append(s.Authors, models.WorkAuthor{PseudID: &id, PseudName: name})
		}
	}
}
