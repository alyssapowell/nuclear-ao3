package demo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

// insert writes rows with one statement per few thousand values, so loading
// takes a handful of round trips to the database instead of thousands.
func insert(ctx context.Context, tx *sql.Tx, table string, cols []string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	per := 30000 / len(cols)
	for start := 0; start < len(rows); start += per {
		end := min(start+per, len(rows))
		var q strings.Builder
		args := make([]any, 0, (end-start)*len(cols))
		fmt.Fprintf(&q, "INSERT INTO %s (%s) VALUES ", table, strings.Join(cols, ", "))
		for i, row := range rows[start:end] {
			if i > 0 {
				q.WriteByte(',')
			}
			q.WriteByte('(')
			for j := range cols {
				if j > 0 {
					q.WriteByte(',')
				}
				args = append(args, row[j])
				fmt.Fprintf(&q, "$%d", len(args))
			}
			q.WriteByte(')')
		}
		if _, err := tx.ExecContext(ctx, q.String(), args...); err != nil {
			return fmt.Errorf("insert into %s: %w", table, err)
		}
	}
	return nil
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// write puts the plan into the database. The order follows what the app
// does: accounts and pseuds, then works with their creatorships, chapters
// and tags, then what readers did. Triggers keep the chapter, word, kudos,
// comment, bookmark, tag and series counts and the profile statistics; the
// few totals that only app code keeps are set here.
func (p *plan) write(ctx context.Context, tx *sql.Tx, hash string) error {
	var rows [][]any

	// Accounts: confirmed at once, like a sign-up in trial mode, with no
	// username (sign-up takes a pen name, which becomes the default pseud).
	// The database gives each one the plain "user" role; nothing more.
	for _, a := range p.accounts {
		rows = append(rows, []any{a.id, a.email, hash, true, true, a.joined, a.joined, a.lastLogin})
	}
	if err := insert(ctx, tx, "users",
		[]string{"id", "email", "password_hash", "is_active", "is_verified", "created_at", "updated_at", "last_login_at"}, rows); err != nil {
		return err
	}

	rows = nil
	for _, a := range p.accounts {
		for i, ps := range a.pseuds {
			rows = append(rows, []any{ps.id, a.id, ps.name, nilIfEmpty(ps.desc), i == 0, ps.created, ps.created})
		}
	}
	if err := insert(ctx, tx, "pseuds",
		[]string{"id", "user_id", "name", "description", "is_default", "created_at", "updated_at"}, rows); err != nil {
		return err
	}

	// Tags that are wrangled, as they would be on an archive with wranglers.
	rows = nil
	for _, t := range p.tags {
		rows = append(rows, []any{t.id, t.name, t.kind, true, true, t.created, t.created})
	}
	if err := insert(ctx, tx, "tags",
		[]string{"id", "name", "type", "is_canonical", "canonical", "created_at", "updated_at"}, rows); err != nil {
		return err
	}

	// Works. chapter_count and word_count start at their defaults: the
	// chapters' trigger sets them.
	rows = nil
	for _, w := range p.works {
		status := "published"
		var published any = w.published
		if w.draft {
			status, published = "draft", nil
		}
		rows = append(rows, []any{w.id, w.owner.id, w.title, w.summary, w.notes, "en", w.rating,
			pq.Array(w.categories), pq.Array(w.warnings), pq.Array(w.fandoms), pq.Array(w.characters),
			pq.Array(w.rels), pq.Array(w.freeforms), w.maxChapters, w.complete, status, w.draft,
			w.restricted, w.commentPolicy, w.commentsOff, w.anon, w.inAnon, w.inUnrevealed,
			w.hits, published, w.published, w.updated})
	}
	if err := insert(ctx, tx, "works",
		[]string{"id", "user_id", "title", "summary", "notes", "language", "rating",
			"category", "warnings", "fandoms", "characters", "relationships", "freeform_tags",
			"max_chapters", "is_complete", "status", "is_draft",
			"restricted", "comment_policy", "disable_comments", "is_anonymous", "in_anon_collection", "in_unrevealed_collection",
			"hit_count", "published_at", "created_at", "updated_at"}, rows); err != nil {
		return err
	}

	rows = nil
	for _, w := range p.works {
		for _, ps := range w.creators {
			rows = append(rows, []any{w.id, "Work", ps.id, true, w.published, w.published})
		}
	}
	if err := insert(ctx, tx, "creatorships",
		[]string{"creation_id", "creation_type", "pseud_id", "approved", "created_at", "updated_at"}, rows); err != nil {
		return err
	}

	rows = nil
	for _, w := range p.works {
		for _, c := range w.chapters {
			var published any
			if !c.draft {
				published = c.published
			}
			rows = append(rows, []any{c.id, w.id, c.number, c.title, c.summary, c.notes, c.endNotes, c.content, c.words,
				c.draft, published, c.created, c.created})
		}
	}
	if err := insert(ctx, tx, "chapters",
		[]string{"id", "work_id", "chapter_number", "title", "summary", "notes", "end_notes", "content", "word_count",
			"is_draft", "published_at", "created_at", "updated_at"}, rows); err != nil {
		return err
	}

	// The tag links, in the same order the app attaches them. Tag totals
	// (use_count) come from the trigger.
	rows = nil
	for _, w := range p.works {
		for _, g := range []struct {
			kind  string
			names []string
		}{{"fandom", w.fandoms}, {"character", w.characters}, {"relationship", w.rels}, {"freeform", w.freeforms},
			{"warning", w.warnings}, {"category", w.categories}} {
			for _, name := range g.names {
				rows = append(rows, []any{w.id, p.tagByKey[name].id, w.published})
			}
		}
	}
	if err := insert(ctx, tx, "work_tags", []string{"work_id", "tag_id", "created_at"}, rows); err != nil {
		return err
	}

	// Series. Their counts come from the trigger on series_works.
	rows = nil
	for _, s := range p.series {
		rows = append(rows, []any{s.id, s.title, s.desc, s.notes, s.owner.id, s.complete, s.created, s.created})
	}
	if err := insert(ctx, tx, "series",
		[]string{"id", "title", "description", "notes", "user_id", "is_complete", "created_at", "updated_at"}, rows); err != nil {
		return err
	}
	rows = nil
	for _, s := range p.series {
		for i, w := range s.works {
			rows = append(rows, []any{s.id, w.id, i + 1, w.published})
		}
	}
	if err := insert(ctx, tx, "series_works", []string{"series_id", "work_id", "position", "created_at"}, rows); err != nil {
		return err
	}

	// Collections. Their work count is kept by app code, so it is counted here.
	rows = nil
	var items [][]any
	for _, c := range p.colls {
		rows = append(rows, []any{c.id, c.name, c.title, c.desc, c.owner.id, c.pseud.id, c.open, c.moderated, c.anonymous, len(c.works), c.created, c.created})
		for _, w := range c.works {
			addedBy := w.owner
			if !c.open {
				addedBy = c.owner // the maintainer adds works to a closed collection
			}
			added := later(c.created, w.published)
			items = append(items, []any{c.id, w.id, addedBy.id, true, added, added})
		}
	}
	if err := insert(ctx, tx, "collections",
		[]string{"id", "name", "title", "description", "user_id", "pseud_id", "is_open", "is_moderated", "is_anonymous", "work_count", "created_at", "updated_at"}, rows); err != nil {
		return err
	}
	if err := insert(ctx, tx, "collection_items",
		[]string{"collection_id", "work_id", "added_by", "is_approved", "added_at", "approved_at"}, items); err != nil {
		return err
	}

	// Comments come parents first, so a reply finds its parent (the trigger
	// sets the reply count and the thread level).
	rows = nil
	for _, c := range p.comments {
		var user, pseudID, guest, guestEmail, chapterID, parentID any
		if c.acct != nil {
			user, pseudID = c.acct.id, c.pseud.id
		} else {
			guest, guestEmail = c.guestName, c.guestEmail
		}
		if c.chapter != nil {
			chapterID = c.chapter.id
		}
		if c.parent != nil {
			parentID = c.parent.id
		}
		rows = append(rows, []any{c.id, c.work.id, chapterID, user, pseudID, parentID, c.content, guest, guestEmail, c.ip,
			c.created, c.created, nullTime(c.edited)})
	}
	if err := insert(ctx, tx, "comments",
		[]string{"id", "work_id", "chapter_id", "user_id", "pseudonym_id", "parent_comment_id", "content", "guest_name", "guest_email", "ip_address",
			"created_at", "updated_at", "edited_at"}, rows); err != nil {
		return err
	}

	rows = nil
	for _, k := range p.kudos {
		var user, guest any
		if k.acct != nil {
			user = k.acct.id
		} else {
			guest = k.guest
		}
		rows = append(rows, []any{k.id, k.work.id, user, guest, k.ip, k.created})
	}
	if err := insert(ctx, tx, "kudos", []string{"id", "work_id", "user_id", "guest_session", "ip_address", "created_at"}, rows); err != nil {
		return err
	}

	rows = nil
	for _, b := range p.marks {
		rows = append(rows, []any{b.id, b.work.id, b.acct.id, b.pseud.id, b.notes, pq.Array(b.tags), b.private, b.rec, b.created, b.created})
	}
	if err := insert(ctx, tx, "bookmarks",
		[]string{"id", "work_id", "user_id", "pseud_id", "notes", "tags", "is_private", "is_rec", "created_at", "updated_at"}, rows); err != nil {
		return err
	}

	rows = nil
	for _, s := range p.subs {
		rows = append(rows, []any{s.id, s.acct.id, s.kind, s.target, s.name, pq.Array(s.events), s.freq, true, s.created, s.created})
	}
	if err := insert(ctx, tx, "subscriptions",
		[]string{"id", "user_id", "type", "target_id", "target_name", "events", "frequency", "is_active", "created_at", "updated_at"}, rows); err != nil {
		return err
	}

	rows = nil
	for _, b := range p.blocks {
		rows = append(rows, []any{b.id, b.blocker.id, b.blocked.id, b.pseud.id, b.pseud.name, b.created})
	}
	if err := insert(ctx, tx, "user_blocks", []string{"id", "blocker_id", "blocked_id", "pseud_id", "pseud_name", "created_at"}, rows); err != nil {
		return err
	}
	rows = nil
	for _, m := range p.mutes {
		rows = append(rows, []any{m.id, m.muter.id, m.pseud.id, m.created, m.created})
	}
	if err := insert(ctx, tx, "user_mutes", []string{"id", "muter_id", "muted_pseud_id", "created_at", "updated_at"}, rows); err != nil {
		return err
	}

	return p.settle(ctx, tx)
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// settle puts back what the triggers touched while loading: every insert
// above set a work's or series' updated_at to the moment of loading, and
// the works are rewritten at the very end so that each author's profile
// statistics are worked out from everything loaded (a kudos given to an
// author is only counted when one of their works is next written).
func (p *plan) settle(ctx context.Context, tx *sql.Tx) error {
	var ids, stamps, hitIDs []string
	var hits []int64
	for _, w := range p.works {
		ids = append(ids, w.id.String())
		stamps = append(stamps, w.updated.Format(time.RFC3339Nano))
		if w.hits > 0 {
			hitIDs = append(hitIDs, w.id.String())
			hits = append(hits, int64(w.hits))
		}
	}
	var sids, sstamps []string
	for _, s := range p.series {
		sids = append(sids, s.id.String())
		sstamps = append(sstamps, s.updated.Format(time.RFC3339Nano))
	}
	var cids []string
	for _, c := range p.comments {
		cids = append(cids, c.id.String())
	}

	steps := []struct {
		what string
		sql  string
		args []any
	}{
		{"series dates", `UPDATE series s SET updated_at = v.u FROM unnest($1::uuid[], $2::timestamptz[]) AS v(id, u) WHERE s.id = v.id`,
			[]any{pq.Array(sids), pq.Array(sstamps)}},
		// Views are kept in both places the app keeps them (works.hit_count
		// is set above; the listing totals are work_statistics.hits).
		{"hits", `UPDATE work_statistics ws SET hits = v.h FROM unnest($1::uuid[], $2::int[]) AS v(id, h) WHERE ws.work_id = v.id`,
			[]any{pq.Array(hitIDs), pq.Array(hits)}},
		// The comment trigger wrote a notification for each comment at the
		// moment of loading: date them when the comment was made, and mark
		// the old ones read.
		{"notification dates", `UPDATE notifications n SET created_at = c.created_at,
				is_read = c.created_at < now() - interval '10 days',
				read_at = CASE WHEN c.created_at < now() - interval '10 days' THEN c.created_at + interval '1 day' END
			FROM comments c WHERE c.id = ANY($1::uuid[]) AND n.type IN ('work_comment', 'comment_reply')
			AND n.data->>'comment_id' = c.id::text`, []any{pq.Array(cids)}},
		{"work dates", `UPDATE works w SET updated_at = v.u FROM unnest($1::uuid[], $2::timestamptz[]) AS v(id, u) WHERE w.id = v.id`,
			[]any{pq.Array(ids), pq.Array(stamps)}},
	}
	for _, s := range steps {
		if _, err := tx.ExecContext(ctx, s.sql, s.args...); err != nil {
			return fmt.Errorf("%s: %w", s.what, err)
		}
	}
	return nil
}
