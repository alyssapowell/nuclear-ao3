package works

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/platform"
	"nuclear-ao3/shared/models"
)

// Reading history and Marked for Later (AO3's readings table).
//
// A reading belongs to the account and is shown only to it. It has no pseud,
// no public route reads it, and every route here lives under /api/v1/my/,
// which apps are never given (platform.appNeverPrefixes). Opening a work as
// an app does not record anything either.
//
// Recording is off the reading path: opening a work adds to an in-memory
// buffer, and the same 30 second flusher that writes hit counts writes the
// buffer in one statement.

// readingKey is one (account, work) pair.
type readingKey struct {
	user, work uuid.UUID
}

type pendingReading struct {
	views   int       // views since the last flush
	chapter int       // last chapter number opened, 0 if none
	at      time.Time // last opened
}

// readingBuffer holds readings not yet written. The zero value is ready.
type readingBuffer struct {
	mu      sync.Mutex
	pending map[readingKey]*pendingReading
}

// add notes that user opened work (chapter 0 means the whole work page). A
// person moving between chapters within one flush interval is one visit.
func (b *readingBuffer) add(user, work uuid.UUID, chapter int, at time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pending == nil {
		b.pending = map[readingKey]*pendingReading{}
	}
	k := readingKey{user, work}
	p := b.pending[k]
	if p == nil {
		b.pending[k] = &pendingReading{views: 1, chapter: chapter, at: at}
		return
	}
	if chapter != 0 {
		p.chapter = chapter
	}
	p.at = at
}

// merge puts back readings whose write failed.
func (b *readingBuffer) merge(failed map[readingKey]*pendingReading) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pending == nil {
		b.pending = map[readingKey]*pendingReading{}
	}
	for k, f := range failed {
		if p := b.pending[k]; p != nil {
			p.views += f.views
		} else {
			b.pending[k] = f
		}
	}
}

func (b *readingBuffer) take() map[readingKey]*pendingReading {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.pending
	b.pending = nil
	return out
}

// forget drops what is waiting for one work of one account, or for all of
// them when work is uuid.Nil.
func (b *readingBuffer) forget(user, work uuid.UUID) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for k := range b.pending {
		if k.user == user && (work == uuid.Nil || k.work == work) {
			delete(b.pending, k)
		}
	}
}

// recordReading notes that the person opened a work. Guests and apps are not
// recorded. Whether the person has history turned on is checked when the
// buffer is written, so reading costs no query.
func (ws *WorkService) recordReading(c *gin.Context, workID uuid.UUID, chapter int) {
	viewer, asReader := platform.ReaderViewer(c)
	if viewer == nil || asReader {
		return
	}
	ws.readings.add(*viewer, workID, chapter, time.Now())
}

// FlushReadings writes the buffered readings in one statement. Readings of
// people who have turned history off, and of works that are not published,
// are dropped. If the write fails they are kept for the next flush.
func (ws *WorkService) FlushReadings(ctx context.Context) error {
	batch := ws.readings.take()
	if len(batch) == 0 {
		return nil
	}
	keys := make([]readingKey, 0, len(batch))
	for k := range batch {
		keys = append(keys, k)
	}
	// Sorted so copies of the app lock rows in the same order.
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].user != keys[j].user {
			return keys[i].user.String() < keys[j].user.String()
		}
		return keys[i].work.String() < keys[j].work.String()
	})
	users, workIDs := make([]string, len(keys)), make([]string, len(keys))
	ats := make([]time.Time, len(keys))
	views, chapters := make([]int64, len(keys)), make([]int64, len(keys))
	for i, k := range keys {
		p := batch[k]
		users[i], workIDs[i], ats[i] = k.user.String(), k.work.String(), p.at
		views[i], chapters[i] = int64(p.views), int64(p.chapter)
	}
	_, err := ws.db.ExecContext(ctx, `
		INSERT INTO readings (user_id, work_id, last_viewed_at, view_count, last_chapter)
		SELECT v.u, v.w, v.t, v.n, NULLIF(v.c, 0)
		FROM unnest($1::uuid[], $2::uuid[], $3::timestamptz[], $4::int[], $5::int[]) AS v(u, w, t, n, c)
		JOIN users usr ON usr.id = v.u AND usr.history_enabled
		JOIN works wk ON wk.id = v.w AND wk.status = 'published'
		ON CONFLICT (user_id, work_id) DO UPDATE SET
			last_viewed_at = GREATEST(readings.last_viewed_at, EXCLUDED.last_viewed_at),
			view_count     = readings.view_count + EXCLUDED.view_count,
			last_chapter   = COALESCE(EXCLUDED.last_chapter, readings.last_chapter)`,
		pq.Array(users), pq.Array(workIDs), pq.Array(ats), pq.Array(views), pq.Array(chapters))
	if err != nil {
		ws.readings.merge(batch)
		return fmt.Errorf("flush readings: %w", err)
	}
	return nil
}

func (ws *WorkService) historyEnabled(ctx context.Context, user uuid.UUID) (bool, error) {
	var on bool
	err := ws.db.QueryRowContext(ctx, `SELECT history_enabled FROM users WHERE id = $1`, user).Scan(&on)
	return on, err
}

func myID(c *gin.Context) (uuid.UUID, bool) {
	v, _ := c.Get("user_uuid")
	id, ok := v.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
	}
	return id, ok
}

// GetMyHistory lists the works the person has read, newest first. With
// show=to-read it lists only those marked for later.
func (ws *WorkService) GetMyHistory(c *gin.Context) {
	me, ok := myID(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	page, limit, offset := pageParams(c)
	where := ` WHERE r.user_id = $1`
	if c.Query("show") == "to-read" {
		where += ` AND r.marked_for_later`
	}
	enabled, err := ws.historyEnabled(ctx, me)
	if err != nil {
		log.Printf("history: settings of %s: %v", me, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}
	var total int
	if err := ws.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM readings r`+where, me).Scan(&total); err != nil {
		log.Printf("history: count for %s: %v", me, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}
	// A work that has since been hidden or restricted away from the person
	// still has its row, but nothing about it is returned.
	rows, err := ws.db.QueryContext(ctx, `
		SELECT r.work_id, r.last_viewed_at, r.view_count, r.last_chapter, r.marked_for_later,
			can_user_view_work(w.id, $1::uuid, false) AND w.status = 'published',
			w.title, COALESCE(w.summary, ''), w.rating, w.fandoms, w.characters, w.relationships,
			w.freeform_tags, w.word_count, w.chapter_count, w.max_chapters, w.is_complete,
			w.published_at, w.updated_at, COALESCE(w.hit_count, 0),
			COALESCE(w.kudos_count, 0) + own_shadowed_kudos(w.id, $1::uuid),
			COALESCE(w.comment_count, 0) + own_shadowed_comments(w.id, $1::uuid),
			COALESCE(w.bookmark_count, 0)
		FROM readings r JOIN works w ON w.id = r.work_id`+where+`
		ORDER BY r.last_viewed_at DESC, r.work_id LIMIT $2 OFFSET $3`, me, limit, offset)
	if err != nil {
		log.Printf("history: list for %s: %v", me, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}
	defer rows.Close()

	items := []gin.H{}
	for rows.Next() {
		var workID uuid.UUID
		var viewed time.Time
		var views int
		var lastChapter sql.NullInt64
		var marked, avail bool
		var w models.Work
		var fandoms, characters, relationships, freeform pq.StringArray
		var maxChapters sql.NullInt64
		var published sql.NullTime
		if err := rows.Scan(&workID, &viewed, &views, &lastChapter, &marked, &avail,
			&w.Title, &w.Summary, &w.Rating, &fandoms, &characters, &relationships, &freeform,
			&w.WordCount, &w.ChapterCount, &maxChapters, &w.IsComplete,
			&published, &w.UpdatedAt, &w.Hits, &w.Kudos, &w.Comments, &w.Bookmarks); err != nil {
			log.Printf("history: scan for %s: %v", me, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
			return
		}
		item := gin.H{
			"work_id":          workID,
			"last_visited":     viewed,
			"visit_count":      views,
			"marked_for_later": marked,
			"available":        avail,
		}
		if lastChapter.Valid {
			item["last_chapter"] = lastChapter.Int64
		}
		if avail {
			blurb := gin.H{
				"id": workID, "title": w.Title, "summary": w.Summary, "rating": w.Rating,
				"fandoms": []string(fandoms), "characters": []string(characters),
				"relationships": []string(relationships), "freeform_tags": []string(freeform),
				"word_count": w.WordCount, "chapter_count": w.ChapterCount, "is_complete": w.IsComplete,
				"published_at": nil, "updated_at": w.UpdatedAt,
				"hits": w.Hits, "kudos": w.Kudos, "comments": w.Comments, "bookmarks": w.Bookmarks,
			}
			if published.Valid {
				blurb["published_at"] = published.Time
			}
			if maxChapters.Valid {
				blurb["max_chapters"] = maxChapters.Int64
			}
			item["work"] = blurb
		}
		items = append(items, item)
	}
	rows.Close()
	// Authors after the list is read: a work that hides its creator names
	// nobody here.
	for _, it := range items {
		if it["available"] != true {
			continue
		}
		authors, err := ws.fetchWorkAuthors(ctx, it["work_id"].(uuid.UUID), &me)
		if err != nil {
			authors = []models.WorkAuthor{}
		}
		it["work"].(gin.H)["authors"] = authors
	}
	c.JSON(http.StatusOK, gin.H{
		"history_enabled": enabled,
		"readings":        items,
		"pagination":      pagination(page, limit, total),
	})
}

// GetMyHistoryEntry says whether the person has history on and whether this
// work is marked for later. The work page uses it to show its button.
func (ws *WorkService) GetMyHistoryEntry(c *gin.Context) {
	me, ok := myID(c)
	if !ok {
		return
	}
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}
	ctx := c.Request.Context()
	enabled, err := ws.historyEnabled(ctx, me)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}
	var marked bool
	err = ws.db.QueryRowContext(ctx, `SELECT marked_for_later FROM readings WHERE user_id = $1 AND work_id = $2`, me, workID).Scan(&marked)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"history_enabled": enabled, "marked_for_later": marked})
}

// SetMarkedForLater marks a work for later or marks it as read. AO3 offers
// the button only with history on, so it is refused otherwise.
func (ws *WorkService) SetMarkedForLater(c *gin.Context) {
	me, ok := myID(c)
	if !ok {
		return
	}
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}
	var body struct {
		Marked *bool `json:"marked"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Marked == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "marked must be true or false"})
		return
	}
	ctx := c.Request.Context()
	enabled, err := ws.historyEnabled(ctx, me)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update history"})
		return
	}
	if !enabled {
		c.JSON(http.StatusConflict, gin.H{"error": "Turn on reading history to use Mark for Later"})
		return
	}
	if !ws.canViewWork(ctx, workID, &me, false) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	// The row exists once the work has been opened and the buffer flushed;
	// marking right away creates it (the open then adds to it).
	if _, err := ws.db.ExecContext(ctx, `
		INSERT INTO readings (user_id, work_id, last_viewed_at, view_count, marked_for_later)
		VALUES ($1, $2, NOW(), 0, $3)
		ON CONFLICT (user_id, work_id) DO UPDATE SET marked_for_later = EXCLUDED.marked_for_later`,
		me, workID, *body.Marked); err != nil {
		log.Printf("history: mark %s for %s: %v", workID, me, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"marked_for_later": *body.Marked})
}

// DeleteMyHistoryEntry removes one work from the person's history.
func (ws *WorkService) DeleteMyHistoryEntry(c *gin.Context) {
	me, ok := myID(c)
	if !ok {
		return
	}
	workID, err := uuid.Parse(c.Param("work_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work ID"})
		return
	}
	ws.readings.forget(me, workID)
	if _, err := ws.db.ExecContext(c.Request.Context(), `DELETE FROM readings WHERE user_id = $1 AND work_id = $2`, me, workID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Removed from history"})
}

// ClearMyHistory removes everything: history and Marked for Later alike, as
// AO3's "Clear History" does.
func (ws *WorkService) ClearMyHistory(c *gin.Context) {
	me, ok := myID(c)
	if !ok {
		return
	}
	ws.readings.forget(me, uuid.Nil)
	if _, err := ws.db.ExecContext(c.Request.Context(), `DELETE FROM readings WHERE user_id = $1`, me); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "History cleared"})
}

// GetHistorySettings reads the history on/off preference.
func (ws *WorkService) GetHistorySettings(c *gin.Context) {
	me, ok := myID(c)
	if !ok {
		return
	}
	on, err := ws.historyEnabled(c.Request.Context(), me)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"history_enabled": on})
}

// SetHistorySettings turns history on or off. Turning it off stops recording
// and drops what is waiting to be written; what is already saved stays until
// the person clears it.
func (ws *WorkService) SetHistorySettings(c *gin.Context) {
	me, ok := myID(c)
	if !ok {
		return
	}
	var body struct {
		HistoryEnabled *bool `json:"history_enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.HistoryEnabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "history_enabled must be true or false"})
		return
	}
	if !*body.HistoryEnabled {
		ws.readings.forget(me, uuid.Nil)
	}
	if _, err := ws.db.ExecContext(c.Request.Context(), `UPDATE users SET history_enabled = $2 WHERE id = $1`, me, *body.HistoryEnabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"history_enabled": *body.HistoryEnabled})
}
