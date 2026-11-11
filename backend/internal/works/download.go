package works

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/export"
	"nuclear-ao3/internal/platform"
)

// Downloads are made when asked for. Each is a few database reads and a
// little work on the text, so a visitor gets a generous number per minute
// and no more.
const (
	downloadsPerMinute = 30
	downloadLimitKeys  = 50000
)

// downloadLimiter is a fixed-window counter per client address. It is held
// in memory and capped, so a flood of addresses cannot grow it without end:
// when it is full, the oldest windows are dropped first.
type downloadLimiter struct {
	mu      sync.Mutex
	windows map[string]*downloadWindow
}

type downloadWindow struct {
	start time.Time
	n     int
}

func (l *downloadLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.windows == nil {
		l.windows = map[string]*downloadWindow{}
	}
	w := l.windows[key]
	if w == nil || now.Sub(w.start) >= time.Minute {
		if w == nil && len(l.windows) >= downloadLimitKeys {
			for k, old := range l.windows {
				if now.Sub(old.start) >= time.Minute {
					delete(l.windows, k)
				}
			}
			if len(l.windows) >= downloadLimitKeys { // all still current: refuse new addresses
				return false
			}
		}
		w = &downloadWindow{start: now}
		l.windows[key] = w
	}
	w.n++
	return w.n <= downloadsPerMinute
}

// DownloadWork answers GET /works/:work_id/download.:format (html, epub or
// txt). It follows the same rules as reading the work: the visibility
// functions decide who may see the work and which chapters, a work that hides
// its creators names nobody, and an app is treated as a plain reader. On top
// of reading, a draft, a work hidden by an admin or marked spam, and a work
// in a collection that has not revealed it are never downloadable, by anyone.
func (ws *WorkService) DownloadWork(c *gin.Context) {
	ctx := c.Request.Context()
	format, ok := export.Lookup(strings.ToLower(c.Param("format")))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unsupported format. Available: epub, html, txt"})
		return
	}

	workID, isRedirect, redirectURL, unknownLegacy := ws.parseWorkID(c.Param("work_id"))
	if unknownLegacy {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	if isRedirect {
		c.Redirect(http.StatusMovedPermanently, redirectURL+"/download."+format.Ext)
		return
	}
	if workID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID format"})
		return
	}

	if !ws.downloads.allow(c.ClientIP(), time.Now()) {
		c.Header("Retry-After", "60")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many downloads. Please try again in a minute."})
		return
	}

	viewer, asReader := platform.ReaderViewer(c)
	notFound := func() { c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"}) }
	if !ws.canViewWork(ctx, workID, viewer, asReader) {
		notFound()
		return
	}

	w := &export.Work{ID: workID.String()}
	var (
		status                           string
		hidden, spam, unrevealed, hiding bool
		warnings, archiveWarning, cat    string
		maxChapters                      sql.NullInt32
		published                        sql.NullTime
		fandoms, chars, rels, tags       pq.StringArray
	)
	err := ws.db.QueryRowContext(ctx, `
		SELECT title, COALESCE(summary, ''), COALESCE(notes, ''), COALESCE(language, ''), COALESCE(rating, ''),
			COALESCE(category, ''), COALESCE(warnings, ''), COALESCE(archive_warning, ''),
			COALESCE(fandoms, '{}'), COALESCE(characters, '{}'), COALESCE(relationships, '{}'), COALESCE(freeform_tags, '{}'),
			COALESCE(word_count, 0), max_chapters, COALESCE(status, 'draft'),
			COALESCE(hidden_by_admin, false), COALESCE(spam, false), COALESCE(in_unrevealed_collection, false),
			COALESCE(work_hides_creator(id), false), published_at, updated_at
		FROM works WHERE id = $1`, workID).Scan(
		&w.Title, &w.SummaryHTML, &w.NotesHTML, &w.Language, &w.Rating,
		&cat, &warnings, &archiveWarning,
		&fandoms, &chars, &rels, &tags,
		&w.Words, &maxChapters, &status,
		&hidden, &spam, &unrevealed,
		&hiding, &published, &w.Updated)
	if err == sql.ErrNoRows {
		notFound()
		return
	}
	if err != nil {
		log.Printf("download work %s: %v", workID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build download"})
		return
	}
	if status == "draft" || hidden || spam || unrevealed {
		notFound()
		return
	}
	if published.Valid {
		w.Published = published.Time
	}
	if maxChapters.Valid {
		w.ChaptersTotal = int(maxChapters.Int32)
	}
	w.Fandoms, w.Characters, w.Relationships, w.Tags = fandoms, chars, rels, tags
	if cat != "" {
		w.Categories = []string{cat}
	}
	for _, s := range []string{warnings, archiveWarning} {
		if s = strings.TrimSpace(s); s != "" && !contains(w.Warnings, s) {
			w.Warnings = append(w.Warnings, s)
		}
	}
	if base := strings.TrimRight(strings.TrimSpace(os.Getenv("SITE_URL")), "/"); base != "" {
		w.URL = base + "/works/" + workID.String()
	}

	// Names: a work that hides its creators names nobody, for everyone,
	// its own creators included. The series line is left out for the same
	// reason (it would sit next to the work).
	if !hiding {
		authors, err := ws.fetchWorkAuthors(ctx, workID, platform.CreatorOnly(viewer, asReader))
		if err != nil {
			log.Printf("download %s: authors: %v", workID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build download"})
			return
		}
		for _, a := range authors {
			w.Authors = append(w.Authors, a.PseudName)
		}
		if w.Series, err = ws.downloadSeries(c, workID, viewer, asReader); err != nil {
			log.Printf("download %s: series: %v", workID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build download"})
			return
		}
	}

	// Chapters: the ones this viewer may read. A draft chapter is shown only
	// to the work's creators (and never to an app); one an admin hid is
	// left out for everybody.
	rows, err := ws.db.QueryContext(ctx, `
		SELECT chapter_number, COALESCE(title, ''), COALESCE(summary, ''), COALESCE(notes, ''),
			COALESCE(end_notes, ''), COALESCE(content, '')
		FROM chapters
		WHERE work_id = $1 AND NOT COALESCE(hidden_by_admin, false)
			AND can_user_view_chapter(chapters.id, $2, $3)
		ORDER BY chapter_number`, workID, viewer, asReader)
	if err != nil {
		log.Printf("download %s: chapters: %v", workID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build download"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var ch export.Chapter
		if err := rows.Scan(&ch.Number, &ch.Title, &ch.SummaryHTML, &ch.NotesHTML, &ch.EndNotesHTML, &ch.ContentHTML); err != nil {
			log.Printf("download %s: scan chapter: %v", workID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build download"})
			return
		}
		w.Chapters = append(w.Chapters, ch)
	}
	if err := rows.Err(); err != nil || len(w.Chapters) == 0 {
		if err != nil {
			log.Printf("download %s: chapters: %v", workID, err)
		}
		notFound()
		return
	}

	h := c.Writer.Header()
	h.Set("Content-Type", format.ContentType)
	h.Set("Content-Disposition", export.ContentDisposition(w.Title, format.Ext))
	h.Set("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)
	var werr error
	switch format.Ext {
	case "epub":
		werr = export.EPUB(c.Writer, w)
	case "html":
		werr = export.HTML(c.Writer, w)
	default:
		werr = export.Text(c.Writer, w)
	}
	if werr != nil {
		// The status is already sent; all that is left is to say so.
		log.Printf("download %s as %s: %v", workID, format.Ext, werr)
	}
}

// downloadSeries lists the series this work is in that the viewer may see.
func (ws *WorkService) downloadSeries(c *gin.Context, workID uuid.UUID, viewer *uuid.UUID, asReader bool) ([]export.SeriesPart, error) {
	rows, err := ws.db.QueryContext(c.Request.Context(), `
		SELECT s.title, sw.position
		FROM series_works sw JOIN series s ON s.id = sw.series_id
		WHERE sw.work_id = $1 AND can_user_view_series(s.id, $2::uuid, $3::boolean)
		ORDER BY sw.created_at, s.id`, workID, viewer, asReader)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []export.SeriesPart
	for rows.Next() {
		var p export.SeriesPart
		if err := rows.Scan(&p.Title, &p.Position); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
