package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// testDB creates an empty database for one test and drops it afterwards.
// It needs TEST_DATABASE_URL pointing at a Postgres server where the user may
// create databases; without it the test is skipped.
func testDB(t *testing.T) *sql.DB {
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

	name := fmt.Sprintf("migrations_test_%d", time.Now().UnixNano())
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
	return db
}

func TestEveryMigrationAppliesToAnEmptyDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	all, err := All()
	if err != nil {
		t.Fatal(err)
	}
	applied, err := Up(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != len(all) {
		t.Fatalf("applied %d of %d migrations", len(applied), len(all))
	}

	again, err := Up(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatalf("second run applied %v; it should find nothing to do", again)
	}
}

func TestFailedMigrationLeavesNothingBehind(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	broken := []Migration{
		{Version: "001_ok", SQL: `CREATE TABLE first (id int);`},
		{Version: "002_broken", SQL: `CREATE TABLE half_done (id int); SELECT * FROM no_such_table;`},
	}
	if _, err := up(ctx, db, broken); err == nil {
		t.Fatal("expected the broken migration to fail")
	}

	var halfDone, recorded bool
	db.QueryRow(`SELECT to_regclass('half_done') IS NOT NULL`).Scan(&halfDone)
	db.QueryRow(`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = '002_broken')`).Scan(&recorded)
	if halfDone || recorded {
		t.Fatalf("a failed migration left changes behind (table created: %v, recorded: %v)", halfDone, recorded)
	}
	var firstApplied bool
	db.QueryRow(`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = '001_ok')`).Scan(&firstApplied)
	if !firstApplied {
		t.Fatal("the migration before the broken one should stay applied")
	}
}

func TestRefusesDatabaseWithUntrackedTables(t *testing.T) {
	db := testDB(t)
	if _, err := db.Exec(`CREATE TABLE users (id int)`); err != nil {
		t.Fatal(err)
	}
	if _, err := Up(context.Background(), db); err == nil || !strings.Contains(err.Error(), "RUNBOOK") {
		t.Fatalf("expected a refusal pointing at the runbook, got %v", err)
	}
}

// TestCodeOnlyUsesTablesThatExist reads the Go code for table names after
// FROM, JOIN, INTO and UPDATE, and fails if any of them isn't created by the
// migrations. Before this test existed, notifications, subscriptions and
// collections all queried tables that no migration made.
func TestCodeOnlyUsesTablesThatExist(t *testing.T) {
	db := testDB(t)
	if _, err := Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	known := map[string]bool{}
	rows, err := db.Query(`
		SELECT tablename FROM pg_tables WHERE schemaname = 'public'
		UNION SELECT viewname FROM pg_views WHERE schemaname = 'public'
		UNION SELECT proname FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace WHERE nspname = 'public'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name string
		rows.Scan(&name)
		known[name] = true
	}
	rows.Close()
	// Names that appear after FROM/JOIN but aren't tables.
	for _, n := range []string{"information_schema", "pg_proc", "pg_stat_activity",
		"pg_tables", "pg_indexes", "set", "pg_class", "pg_namespace", "unnest", "generate_series", "jsonb_array_elements_text"} {
		known[n] = true
	}

	sqlRef := regexp.MustCompile(`\b(?:FROM|JOIN|INTO|UPDATE)\s+([a-z_][a-z0-9_]*)\b`) // SQL here is written in capitals
	// Names a query defines for itself with WITH name AS (...).
	cteDef := regexp.MustCompile(`(?:\bWITH(?:\s+RECURSIVE)?|,)\s+([a-z_][a-z0-9_]*)\s+AS\s*\(`)
	missing := map[string][]string{}
	for _, dir := range []string{"../internal", "../shared"} {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") ||
				strings.HasSuffix(path, "test_utils.go") {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			// Only look inside backquoted strings, where this code keeps its SQL.
			for _, lit := range regexp.MustCompile("`[^`]*`").FindAllString(string(src), -1) {
				local := map[string]bool{}
				for _, m := range cteDef.FindAllStringSubmatch(lit, -1) {
					local[strings.ToLower(m[1])] = true
				}
				for _, m := range sqlRef.FindAllStringSubmatch(lit, -1) {
					name := strings.ToLower(m[1])
					if !known[name] && !local[name] {
						missing[name] = append(missing[name], path)
					}
				}
			}
			return nil
		})
	}
	if len(missing) > 0 {
		var lines []string
		for name, paths := range missing {
			sort.Strings(paths)
			lines = append(lines, fmt.Sprintf("  %s (used in %s)", name, paths[0]))
		}
		sort.Strings(lines)
		t.Fatalf("the code uses tables no migration creates; add a migration:\n%s", strings.Join(lines, "\n"))
	}
}
