package html

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// batchSize is how many rows are read and rewritten in one transaction.
const batchSize = 200

// storedHTML lists every column that holds user-written HTML. Every table has
// a uuid primary key called id. Add a column here when you add a field that
// users can write HTML into.
var storedHTML = []struct {
	table   string
	columns []string
}{
	{"works", []string{"summary", "notes", "end_notes"}},
	{"chapters", []string{"summary", "notes", "end_notes", "content"}},
	{"comments", []string{"content"}},
	{"series", []string{"description", "notes"}},
	{"collections", []string{"description"}},
	{"bookmarks", []string{"notes"}},
	{"users", []string{"bio"}},
	{"pseuds", []string{"description"}},
	{"tags", []string{"description"}},
}

// CleanStored runs Clean over every column in storedHTML and rewrites the
// rows that change. It returns how many rows it rewrote.
//
// It is safe to run again and again: a row that is already clean is not
// touched, and one that changes underneath it (someone saves an edit while
// the job runs) is skipped, because a saved edit is already clean.
//
// Cleaning is not editing. Each batch runs with nuclear.maintenance set, which
// the triggers that bump updated_at, recount or notify check (migration 031),
// so updated_at stays as it was and nobody is told anything.
func CleanStored(ctx context.Context, db *sql.DB) (int, error) {
	total := 0
	for _, t := range storedHTML {
		last := "00000000-0000-0000-0000-000000000000"
		for {
			n, next, err := cleanBatch(ctx, db, t.table, t.columns, last)
			if err != nil {
				return total, fmt.Errorf("clean %s: %w", t.table, err)
			}
			total += n
			if next == "" {
				break
			}
			last = next
		}
	}
	return total, nil
}

// cleanBatch cleans the batchSize rows after id `after`. It returns the id of
// the last row it looked at, or "" when there are no more.
func cleanBatch(ctx context.Context, db *sql.DB, table string, columns []string, after string) (changed int, last string, err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, "", err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT set_config('nuclear.maintenance', 'on', true)`); err != nil {
		return 0, "", err
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`SELECT id::text, %s FROM %s WHERE id > $1::uuid ORDER BY id LIMIT %d`,
		strings.Join(columns, ", "), table, batchSize), after)
	if err != nil {
		return 0, "", err
	}
	type fix struct {
		id        string
		set       []string // columns to rewrite
		newValues []any
		oldValues []any
	}
	var fixes []fix
	seen := 0
	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		dest := make([]any, 0, len(columns)+1)
		var id string
		dest = append(dest, &id)
		for i := range values {
			dest = append(dest, &values[i])
		}
		if err := rows.Scan(dest...); err != nil {
			rows.Close()
			return 0, "", err
		}
		seen++
		last = id
		f := fix{id: id}
		for i, v := range values {
			if !v.Valid {
				continue
			}
			if cleaned := Clean(v.String); cleaned != v.String {
				f.set = append(f.set, columns[i])
				f.newValues = append(f.newValues, cleaned)
				f.oldValues = append(f.oldValues, v.String)
			}
		}
		if len(f.set) > 0 {
			fixes = append(fixes, f)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, "", err
	}
	rows.Close()

	for _, f := range fixes {
		var sets, wheres []string
		args := []any{f.id}
		for i, col := range f.set {
			args = append(args, f.newValues[i])
			sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
			args = append(args, f.oldValues[i])
			wheres = append(wheres, fmt.Sprintf("%s = $%d", col, len(args)))
		}
		res, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET %s WHERE id = $1::uuid AND %s`,
			table, strings.Join(sets, ", "), strings.Join(wheres, " AND ")), args...)
		if err != nil {
			return 0, "", err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			changed++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, "", err
	}
	if seen < batchSize {
		last = ""
	}
	return changed, last, nil
}
