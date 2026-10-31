package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"nuclear-ao3/internal/testdb"
)

type state struct {
	attempts       int
	done, failed   bool
	lastError      string
	dueInTheFuture bool
}

func jobState(t *testing.T, db *sql.DB, kind string) state {
	t.Helper()
	var s state
	var lastError sql.NullString
	err := db.QueryRow(`SELECT attempts, done_at IS NOT NULL, failed_at IS NOT NULL, last_error, run_at > now()
		FROM jobs WHERE kind = $1`, kind).Scan(&s.attempts, &s.done, &s.failed, &lastError, &s.dueInTheFuture)
	if err != nil {
		t.Fatal(err)
	}
	s.lastError = lastError.String
	return s
}

func TestAJobRunsOnceAndIsMarkedDone(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	q := New(db)
	var got struct{ To string }
	q.Handle("email", func(ctx context.Context, p json.RawMessage) error { return json.Unmarshal(p, &got) })

	if err := Enqueue(ctx, db, "email", map[string]string{"to": "reader@example.org"}); err != nil {
		t.Fatal(err)
	}
	if worked, err := q.RunOne(ctx); !worked || err != nil {
		t.Fatalf("worked=%v err=%v", worked, err)
	}
	if got.To != "reader@example.org" {
		t.Errorf("handler got %+v", got)
	}
	if s := jobState(t, db, "email"); !s.done || s.attempts != 1 {
		t.Errorf("state %+v", s)
	}
	if worked, _ := q.RunOne(ctx); worked {
		t.Error("a finished job ran again")
	}
}

func TestAFailingJobIsRetriedThenGivenUpOnThenRetriedByHand(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	q := New(db)
	q.Handle("flaky", func(context.Context, json.RawMessage) error { return errors.New("mail server down") })
	Enqueue(ctx, db, "flaky", nil)
	db.Exec(`UPDATE jobs SET max_attempts = 3`)

	q.RunOne(ctx)
	s := jobState(t, db, "flaky")
	if s.done || s.failed || !s.dueInTheFuture || s.lastError != "mail server down" {
		t.Fatalf("after one failure: %+v", s)
	}
	if worked, _ := q.RunOne(ctx); worked {
		t.Fatal("a job waiting to retry ran early")
	}

	for i := 0; i < 2; i++ {
		db.Exec(`UPDATE jobs SET run_at = now()`) // skip the wait
		q.RunOne(ctx)
	}
	if s := jobState(t, db, "flaky"); !s.failed || s.attempts != 3 {
		t.Fatalf("after the last attempt: %+v", s)
	}
	fails, _ := Failures(ctx, db, 10)
	if len(fails) != 1 || fails[0].Error != "mail server down" {
		t.Errorf("failures: %+v", fails)
	}

	q.Handle("flaky", func(context.Context, json.RawMessage) error { return nil })
	if n, err := Retry(ctx, db, 0); n != 1 || err != nil {
		t.Fatalf("retry: %d %v", n, err)
	}
	q.RunOne(ctx)
	if s := jobState(t, db, "flaky"); !s.done {
		t.Errorf("after a manual retry: %+v", s)
	}
}

func TestAPanickingOrUnknownJobDoesNotStopTheQueue(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	q := New(db)
	q.Handle("boom", func(context.Context, json.RawMessage) error { panic("bad data") })
	Enqueue(ctx, db, "boom", nil)
	Enqueue(ctx, db, "nobody-handles-this", nil)

	for i := 0; i < 2; i++ {
		if _, err := q.RunOne(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if s := jobState(t, db, "boom"); s.done || s.lastError == "" {
		t.Errorf("panicking job: %+v", s)
	}
	if s := jobState(t, db, "nobody-handles-this"); s.done || s.lastError == "" {
		t.Errorf("unknown job: %+v", s)
	}
}

func TestAJobFromARolledBackTransactionNeverExists(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	tx, _ := db.Begin()
	Enqueue(ctx, tx, "email", nil)
	tx.Rollback()
	var n int
	db.QueryRow(`SELECT count(*) FROM jobs`).Scan(&n)
	if n != 0 {
		t.Errorf("%d jobs left after rollback", n)
	}
}

func TestWorkersNeverRunTheSameJobTwice(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	const n = 40
	for i := 0; i < n; i++ {
		Enqueue(ctx, db, "count", i)
	}
	var runs int64
	seen := sync.Map{}
	q := New(db)
	q.Handle("count", func(_ context.Context, p json.RawMessage) error {
		atomic.AddInt64(&runs, 1)
		if _, dup := seen.LoadOrStore(string(p), true); dup {
			t.Errorf("job %s ran twice", p)
		}
		return nil
	})
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				worked, err := q.RunOne(ctx)
				if err != nil {
					t.Error(err)
					return
				}
				if !worked {
					return
				}
			}
		}()
	}
	wg.Wait()
	if runs != n {
		t.Errorf("ran %d jobs, want %d", runs, n)
	}
}

func TestAJobWhoseWorkerDiedIsPickedUpAgain(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	Enqueue(ctx, db, "export", nil)
	// A worker claimed it and then the server went away.
	db.Exec(`UPDATE jobs SET locked_at = now(), attempts = 1`)
	q := New(db)
	ran := false
	q.Handle("export", func(context.Context, json.RawMessage) error { ran = true; return nil })

	if worked, _ := q.RunOne(ctx); worked {
		t.Fatal("took over a job that may still be running")
	}
	db.Exec(`UPDATE jobs SET locked_at = now() - interval '1 hour'`)
	if worked, _ := q.RunOne(ctx); !worked || !ran {
		t.Fatal("didn't pick up the abandoned job")
	}
}

func TestSummary(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	Enqueue(ctx, db, "email", nil)
	Enqueue(ctx, db, "email", nil)
	db.Exec(`UPDATE jobs SET failed_at = now() WHERE id = (SELECT min(id) FROM jobs)`)
	got, err := Summarize(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Waiting != 1 || got[0].Failed != 1 || got[0].Oldest == nil {
		t.Errorf("summary %+v", got)
	}
}
