// Package migrations holds the database schema as numbered SQL files and
// applies the ones a database hasn't had yet.
//
// To change the schema, add a new file named NNN_short_description.sql with
// the next number. Never edit a file that has already been released: databases
// that already ran it won't run it again.
//
// Each file runs inside its own transaction, so a file either applies
// completely or not at all. Applied files are recorded in the
// schema_migrations table.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
)

//go:embed *.sql
var files embed.FS

// lockID is an arbitrary constant. Holding a Postgres advisory lock on it
// means that when several copies of the app start at once, only one of them
// migrates and the rest wait for it.
const lockID = 7_032_026

// Migration is one numbered SQL file.
type Migration struct {
	Version string // the file name without ".sql", e.g. "001_create_users_and_auth"
	SQL     string
}

// All returns every migration in the order it must run.
func All() ([]Migration, error) {
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	out := make([]Migration, 0, len(names))
	for _, name := range names {
		body, err := files.ReadFile(name)
		if err != nil {
			return nil, err
		}
		out = append(out, Migration{Version: strings.TrimSuffix(name, ".sql"), SQL: string(body)})
	}
	return out, nil
}

// Up applies every migration the database hasn't had yet, oldest first, and
// returns the versions it applied.
func Up(ctx context.Context, db *sql.DB) ([]string, error) {
	all, err := All()
	if err != nil {
		return nil, err
	}
	return up(ctx, db, all)
}

func up(ctx context.Context, db *sql.DB, all []Migration) ([]string, error) {

	// The lock belongs to one connection, so hold one for the whole run.
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return nil, fmt.Errorf("waiting for migration lock: %w", err)
	}
	defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, lockID)

	if err := refuseUntrackedDatabase(ctx, conn); err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	)`); err != nil {
		return nil, err
	}

	done, err := appliedVersions(ctx, conn)
	if err != nil {
		return nil, err
	}

	var applied []string
	for _, m := range all {
		if done[m.Version] {
			continue
		}
		if err := apply(ctx, conn, m); err != nil {
			return applied, err
		}
		log.Printf("migrations: applied %s", m.Version)
		applied = append(applied, m.Version)
	}
	return applied, nil
}

// Pending returns the versions that Up would apply, without applying them.
func Pending(ctx context.Context, db *sql.DB) ([]string, error) {
	all, err := All()
	if err != nil {
		return nil, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	done, err := appliedVersions(ctx, conn)
	if err != nil {
		return nil, err
	}
	var pending []string
	for _, m := range all {
		if !done[m.Version] {
			pending = append(pending, m.Version)
		}
	}
	return pending, nil
}

func apply(ctx context.Context, conn *sql.Conn, m Migration) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// No arguments, so the whole file is sent as one multi-statement query.
	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		return fmt.Errorf("migration %s failed, nothing from it was applied: %w", m.Version, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, m.Version); err != nil {
		return err
	}
	return tx.Commit()
}

func appliedVersions(ctx context.Context, conn *sql.Conn) (map[string]bool, error) {
	done := map[string]bool{}
	var exists bool
	if err := conn.QueryRowContext(ctx, `SELECT to_regclass('schema_migrations') IS NOT NULL`).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return done, nil
	}
	rows, err := conn.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		done[v] = true
	}
	return done, rows.Err()
}

// refuseUntrackedDatabase stops a run against a database that has tables but
// no record of which migrations built them. Older setups loaded the SQL files
// through Postgres's first-start scripts, which stopped at the first error, so
// there's no way to tell how far they got. Guessing could run a file twice or
// skip one.
func refuseUntrackedDatabase(ctx context.Context, conn *sql.Conn) error {
	var tracked, hasUsers bool
	err := conn.QueryRowContext(ctx, `SELECT
		to_regclass('schema_migrations') IS NOT NULL,
		to_regclass('users') IS NOT NULL`).Scan(&tracked, &hasUsers)
	if err != nil {
		return err
	}
	if !tracked && hasUsers {
		return fmt.Errorf("this database already has tables but no schema_migrations record, " +
			"so it was set up by an older version and its state is unknown. " +
			"See docs/RUNBOOK.md, \"Database was created before migrations were tracked\"")
	}
	return nil
}
