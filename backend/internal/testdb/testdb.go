// Package testdb gives tests their own real, fully migrated Postgres database.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"nuclear-ao3/migrations"

	_ "github.com/lib/pq"
)

// New creates a database with every migration applied and drops it when the
// test ends. It needs TEST_DATABASE_URL pointing at a Postgres server where
// the user may create databases; without it the test is skipped.
func New(t *testing.T) *sql.DB {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	admin, err := sql.Open("postgres", base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })

	name := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		admin.Exec("DROP DATABASE IF EXISTS " + name)
	})
	if _, err := migrations.Up(context.Background(), db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}
