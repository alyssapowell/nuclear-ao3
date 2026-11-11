package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"nuclear-ao3/internal/demo"
)

// loadDemoData fills a trial archive with made-up accounts and works, once.
// It is used by "nuclear demo-data" and by DEMO_DATA=true at startup, and
// refuses unless TRIAL_NO_EMAIL=true and DEMO_PASSWORD is a real password.
// The database must already be migrated.
func loadDemoData(ctx context.Context, db *sql.DB, getenv func(string) string) error {
	password, err := demo.Password(getenv)
	if err != nil {
		return err
	}
	start := time.Now()
	summary, err := demo.Load(ctx, db, password, time.Now())
	if err != nil {
		return err
	}
	if summary.Skipped {
		log.Printf("demo data: the demo accounts (%s onward) already exist, so nothing was loaded. You can remove DEMO_DATA now.", demo.AccountEmail(1))
		return nil
	}
	log.Printf("demo data: loaded in %s: %s", time.Since(start).Round(time.Millisecond), summary)
	log.Printf("demo data: log in as %s to %s with DEMO_PASSWORD. Anyone who knows that password can log in as them. You can remove DEMO_DATA now.",
		demo.AccountEmail(1), demo.AccountEmail(demo.AccountCount))
	return nil
}
