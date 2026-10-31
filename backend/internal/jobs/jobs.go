// Package jobs runs background work (emails, notifications, exports) from
// a table in Postgres.
//
// Work is added with Enqueue, often in the same transaction as the change
// that caused it: if the transaction rolls back, the job is never sent, and
// once it commits, the job can't be lost. A worker runs each job with the
// handler registered for its kind. A job that fails is tried again later,
// waiting longer each time, and after max_attempts it is marked failed and
// kept for someone to look at (`nuclear jobs`) and retry (`nuclear jobs
// retry`).
//
// Any number of copies of the app can run workers: each job is claimed by
// exactly one of them. A job whose worker died is picked up again after
// lockTimeout.
package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"runtime/debug"
	"time"
)

// Handler does one job. Returning an error means "try again later". It must
// be safe to run twice: a worker can stop after the work is done but before
// the job is marked done.
type Handler func(ctx context.Context, payload json.RawMessage) error

// Execer is a *sql.DB or a *sql.Tx.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

const (
	// How long a job may run before another worker may assume its worker
	// died and take it over.
	lockTimeout = 10 * time.Minute
	// How long to wait between looks at an empty queue.
	pollInterval = time.Second
	// Finished jobs are deleted after this long; failed ones are kept.
	keepDone = 7 * 24 * time.Hour
)

// Enqueue adds a job of the given kind. payload is stored as JSON.
func Enqueue(ctx context.Context, db Execer, kind string, payload any) error {
	return EnqueueAt(ctx, db, kind, payload, time.Time{})
}

// EnqueueAt adds a job that won't run before runAt (zero means now).
func EnqueueAt(ctx context.Context, db Execer, kind string, payload any, runAt time.Time) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", kind, err)
	}
	if runAt.IsZero() {
		_, err = db.ExecContext(ctx, `INSERT INTO jobs (kind, payload) VALUES ($1, $2)`, kind, body)
	} else {
		_, err = db.ExecContext(ctx, `INSERT INTO jobs (kind, payload, run_at) VALUES ($1, $2, $3)`, kind, body, runAt)
	}
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", kind, err)
	}
	return nil
}

// Queue runs jobs with the handlers registered on it.
type Queue struct {
	db       *sql.DB
	handlers map[string]Handler
	timeout  time.Duration // how long one job may take
}

// New returns a queue with no handlers yet.
func New(db *sql.DB) *Queue {
	return &Queue{db: db, handlers: map[string]Handler{}, timeout: 5 * time.Minute}
}

// Handle registers the handler for a kind of job. Register every kind
// before calling Run.
func (q *Queue) Handle(kind string, h Handler) {
	q.handlers[kind] = h
}

// Run works through jobs until ctx is cancelled. Run several at once for
// more throughput.
func (q *Queue) Run(ctx context.Context) {
	lastCleanup := time.Time{}
	for ctx.Err() == nil {
		if time.Since(lastCleanup) > time.Hour {
			q.cleanup(ctx)
			lastCleanup = time.Now()
		}
		worked, err := q.RunOne(ctx)
		if err != nil {
			log.Printf("jobs: %v", err)
		}
		if !worked || err != nil {
			select {
			case <-ctx.Done():
			case <-time.After(pollInterval):
			}
		}
	}
}

// RunOne runs the next job that is due, if any, and reports whether it
// found one. The error is about the queue itself (such as the database
// being unreachable), not about the job, whose failure is recorded on it.
func (q *Queue) RunOne(ctx context.Context) (bool, error) {
	var (
		id                 int64
		kind               string
		payload            []byte
		attempts, maxTries int
	)
	err := q.db.QueryRowContext(ctx, `
		UPDATE jobs SET locked_at = now(), attempts = attempts + 1
		WHERE id = (
			SELECT id FROM jobs
			WHERE done_at IS NULL AND failed_at IS NULL AND run_at <= now()
			  AND (locked_at IS NULL OR locked_at < now() - $1::interval)
			ORDER BY run_at, id
			LIMIT 1
			FOR UPDATE SKIP LOCKED)
		RETURNING id, kind, payload, attempts, max_attempts`,
		fmt.Sprintf("%d seconds", int(lockTimeout.Seconds())),
	).Scan(&id, &kind, &payload, &attempts, &maxTries)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim job: %w", err)
	}

	jobErr := q.run(ctx, kind, payload)
	if jobErr == nil {
		_, err = q.db.ExecContext(ctx, `UPDATE jobs SET done_at = now(), locked_at = NULL, last_error = NULL WHERE id = $1`, id)
		return true, err
	}

	if attempts >= maxTries {
		log.Printf("jobs: %s job %d failed for good after %d attempts: %v", kind, id, attempts, jobErr)
		_, err = q.db.ExecContext(ctx, `UPDATE jobs SET failed_at = now(), locked_at = NULL, last_error = $2 WHERE id = $1`,
			id, jobErr.Error())
		return true, err
	}
	wait := backoff(attempts)
	log.Printf("jobs: %s job %d failed (attempt %d of %d), trying again in %s: %v", kind, id, attempts, maxTries, wait, jobErr)
	_, err = q.db.ExecContext(ctx, `UPDATE jobs SET run_at = now() + $2::interval, locked_at = NULL, last_error = $3 WHERE id = $1`,
		id, fmt.Sprintf("%d seconds", int(wait.Seconds())), jobErr.Error())
	return true, err
}

// run calls the handler, turning a panic into an error so one bad job can't
// take the server down.
func (q *Queue) run(ctx context.Context, kind string, payload []byte) (err error) {
	h, ok := q.handlers[kind]
	if !ok {
		return fmt.Errorf("no handler for job kind %q", kind)
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, q.timeout)
	defer cancel()
	return h(ctx, payload)
}

// backoff is how long to wait before attempt n+1: 30s, 1m, 2m, 4m ... up
// to 6 hours, so the default 10 attempts keep trying for about four hours.
func backoff(attempts int) time.Duration {
	d := time.Duration(30*math.Pow(2, float64(attempts-1))) * time.Second
	if d > 6*time.Hour {
		d = 6 * time.Hour
	}
	return d
}

func (q *Queue) cleanup(ctx context.Context) {
	if _, err := q.db.ExecContext(ctx, `DELETE FROM jobs WHERE done_at < now() - $1::interval`,
		fmt.Sprintf("%d seconds", int(keepDone.Seconds()))); err != nil {
		log.Printf("jobs: cleanup: %v", err)
	}
}

// Summary is how many jobs are in each state, by kind.
type Summary struct {
	Kind    string
	Waiting int // due now or later, not yet done
	Running int
	Failed  int
	Oldest  *time.Time // when the oldest waiting job was due
}

// Summarize counts jobs by kind and state.
func Summarize(ctx context.Context, db *sql.DB) ([]Summary, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT kind,
		       count(*) FILTER (WHERE done_at IS NULL AND failed_at IS NULL AND locked_at IS NULL),
		       count(*) FILTER (WHERE done_at IS NULL AND failed_at IS NULL AND locked_at IS NOT NULL),
		       count(*) FILTER (WHERE failed_at IS NOT NULL),
		       min(run_at) FILTER (WHERE done_at IS NULL AND failed_at IS NULL)
		FROM jobs GROUP BY kind ORDER BY kind`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Summary
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.Kind, &s.Waiting, &s.Running, &s.Failed, &s.Oldest); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Failure is one job that gave up.
type Failure struct {
	ID       int64
	Kind     string
	Attempts int
	Error    string
	FailedAt time.Time
}

// Failures lists the most recent failed jobs.
func Failures(ctx context.Context, db *sql.DB, limit int) ([]Failure, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, kind, attempts, coalesce(last_error, ''), failed_at
		FROM jobs WHERE failed_at IS NOT NULL ORDER BY failed_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Failure
	for rows.Next() {
		var f Failure
		if err := rows.Scan(&f.ID, &f.Kind, &f.Attempts, &f.Error, &f.FailedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// Retry puts failed jobs back in the queue with a fresh set of attempts:
// one job by id, or every failed job when id is 0. It returns how many.
func Retry(ctx context.Context, db *sql.DB, id int64) (int64, error) {
	res, err := db.ExecContext(ctx, `UPDATE jobs SET failed_at = NULL, attempts = 0, run_at = now()
		WHERE failed_at IS NOT NULL AND ($1 = 0 OR id = $1)`, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
