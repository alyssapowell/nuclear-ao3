package notify

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/jobs"
	"nuclear-ao3/shared/models"
	"nuclear-ao3/shared/notifications"
)

// PostedJobKind is the job that tells subscribers a work was posted or got a
// new chapter. It is queued in the same transaction that posts, so posting
// stays fast and the announcement can't be lost.
const PostedJobKind = "work-posted"

type postedJob struct {
	WorkID    uuid.UUID  `json:"work_id"`
	ChapterID *uuid.UUID `json:"chapter_id,omitempty"` // set for a new chapter, unset for a newly posted work
}

// EnqueuePosted queues the announcement of a newly posted work (chapterID
// nil) or of a newly posted chapter of a work that is already posted.
func EnqueuePosted(ctx context.Context, db jobs.Execer, workID uuid.UUID, chapterID *uuid.UUID) error {
	return jobs.Enqueue(ctx, db, PostedJobKind, postedJob{WorkID: workID, ChapterID: chapterID})
}

// HandlePosted is the handler of PostedJobKind.
//
// One person gets one notification per event, however many of their
// subscriptions match: subscribing to two pseuds of the work, or to a pseud
// and also the work or its series, is still one. What is sent follows the
// person's own notification preferences (in-app, email, frequency) through the
// shared notification engine.
//
// Subscribers to a creator pseud are never told when the work hides its
// creators (anonymous, in an anonymous or still unrevealed collection), and a
// later reveal does not announce anything. Subscribers to the work or series
// themselves are still told, without any names. Nobody is told about a draft
// or an admin-hidden work or chapter, or about something they could not read
// as a plain reader, nor if there is a block between them and any of the
// work's creators. The creators themselves are never told.
//
// A notification names only the pseuds that this subscriber follows and that
// are on the work, never the account and never other pseuds of it.
func (s *NotificationService) HandlePosted(ctx context.Context, payload json.RawMessage) error {
	var job postedJob
	if err := json.Unmarshal(payload, &job); err != nil {
		return nil // can't ever succeed; don't retry
	}

	var (
		title        string
		hides, ready bool
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT w.title, coalesce(work_hides_creator(w.id), false),
		       w.status = 'published' AND NOT w.is_draft AND NOT w.hidden_by_admin AND NOT w.spam
		FROM works w WHERE w.id = $1`, job.WorkID).Scan(&title, &hides, &ready)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("posted %s: %w", job.WorkID, err)
	}
	if !ready {
		return nil
	}

	event := models.EventNewWork
	dispatch := "work:" + job.WorkID.String()
	var chapterArg any
	if job.ChapterID != nil {
		event = models.EventWorkUpdated
		dispatch = "chapter:" + job.ChapterID.String()
		chapterArg = *job.ChapterID
		var chapterOK bool
		err := s.db.QueryRowContext(ctx, `SELECT NOT is_draft AND NOT hidden_by_admin FROM chapters
			WHERE id = $1 AND work_id = $2`, *job.ChapterID, job.WorkID).Scan(&chapterOK)
		if err == sql.ErrNoRows || (err == nil && !chapterOK) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("posted %s: %w", job.WorkID, err)
		}
	}
	// A subscription to the work or series matches the event itself plus the
	// events the subscription UI has always offered for them.
	var workEvents, seriesEvents []string
	if job.ChapterID != nil {
		workEvents = []string{string(models.EventWorkUpdated)}
		seriesEvents = []string{string(models.EventWorkUpdated), string(models.EventSeriesUpdated)}
	} else {
		workEvents = []string{string(models.EventNewWork)}
		seriesEvents = []string{string(models.EventNewWork), string(models.EventSeriesUpdated)}
	}

	rows, err := s.db.QueryContext(ctx, `
		WITH creators AS (
			SELECT p.id AS pseud_id, p.name, p.user_id
			FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id
			WHERE c.creation_id = $1 AND c.creation_type = 'Work' AND c.approved),
		matches AS (
			SELECT s.user_id, cr.name AS pseud_name
			FROM subscriptions s JOIN creators cr ON cr.pseud_id = s.target_id
			WHERE s.type = 'author' AND NOT $2::boolean AND $3 = ANY(s.events)
			  AND s.is_active AND s.frequency <> 'never'
			UNION ALL
			SELECT s.user_id, NULL FROM subscriptions s
			WHERE s.type = 'work' AND s.target_id = $1 AND s.events && $4::text[]
			  AND s.is_active AND s.frequency <> 'never'
			UNION ALL
			SELECT s.user_id, NULL FROM subscriptions s
			WHERE s.type = 'series' AND s.events && $5::text[]
			  AND s.is_active AND s.frequency <> 'never'
			  AND s.target_id IN (SELECT series_id FROM series_works WHERE work_id = $1))
		SELECT m.user_id,
		       coalesce(array_agg(DISTINCT m.pseud_name ORDER BY m.pseud_name) FILTER (WHERE m.pseud_name IS NOT NULL), '{}')
		FROM matches m JOIN users u ON u.id = m.user_id AND u.is_active
		WHERE NOT EXISTS (SELECT 1 FROM creators cr WHERE cr.user_id = m.user_id
		                  OR accounts_blocked(cr.user_id, m.user_id))
		  AND can_user_view_work($1, m.user_id, true)
		  AND ($6::uuid IS NULL OR can_user_view_chapter($6, m.user_id, true))
		GROUP BY m.user_id ORDER BY m.user_id`,
		job.WorkID, hides, string(event), pq.Array(workEvents), pq.Array(seriesEvents), chapterArg)
	if err != nil {
		return fmt.Errorf("posted %s: %w", job.WorkID, err)
	}
	type recipient struct {
		id    uuid.UUID
		names []string
	}
	var recipients []recipient
	for rows.Next() {
		var r recipient
		if err := rows.Scan(&r.id, pq.Array(&r.names)); err != nil {
			rows.Close()
			return fmt.Errorf("posted %s: %w", job.WorkID, err)
		}
		recipients = append(recipients, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("posted %s: %w", job.WorkID, err)
	}

	for _, r := range recipients {
		// The job may run twice (a worker can die before the job is marked
		// done); nobody gets the same announcement twice.
		var already bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM notification_items
			WHERE user_id = $1 AND source_id = $2 AND event = $3 AND extra_data->>'dispatch' = $4)`,
			r.id, job.WorkID, string(event), dispatch).Scan(&already); err != nil {
			return fmt.Errorf("posted %s: %w", job.WorkID, err)
		}
		if already {
			continue
		}

		what := "New work"
		if job.ChapterID != nil {
			what = "New chapter"
		}
		description := fmt.Sprintf("%s: %q.", what, title)
		if len(r.names) > 0 {
			description = fmt.Sprintf("%s: %q by %s.", what, title, strings.Join(r.names, ", "))
		}
		ev := &notifications.EventData{
			Type:        event,
			SourceID:    job.WorkID,
			SourceType:  "work",
			Title:       what + ": " + title,
			Description: description,
			ActionURL:   "/works/" + job.WorkID.String(),
			ExtraData:   map[string]interface{}{"dispatch": dispatch},
			Recipients:  []uuid.UUID{r.id},
		}
		if err := s.notificationSvc.ProcessEvent(ctx, ev); err != nil {
			log.Printf("posted %s: %v", job.WorkID, err)
		}
	}
	return nil
}
