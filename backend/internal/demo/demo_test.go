package demo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"nuclear-ao3/internal/testdb"
)

const testPassword = "correct horse battery"

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestPasswordSettings(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string // a part of the message; empty means fine
	}{
		{"not a trial", map[string]string{"DEMO_PASSWORD": testPassword}, "TRIAL_NO_EMAIL=true"},
		{"trial off", map[string]string{"TRIAL_NO_EMAIL": "false", "DEMO_PASSWORD": testPassword}, "TRIAL_NO_EMAIL=true"},
		{"no password", map[string]string{"TRIAL_NO_EMAIL": "true"}, "DEMO_PASSWORD is not set"},
		{"short password", map[string]string{"TRIAL_NO_EMAIL": "true", "DEMO_PASSWORD": "elevenchars"}, "12 to 40"},
		{"long password", map[string]string{"TRIAL_NO_EMAIL": "true", "DEMO_PASSWORD": strings.Repeat("x", 41)}, "12 to 40"},
		{"twelve is enough", map[string]string{"TRIAL_NO_EMAIL": "true", "DEMO_PASSWORD": "twelve chars"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pw, err := Password(env(tt.env))
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatal(err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("error = %v, want one mentioning %q", err, tt.wantErr)
			case err != nil && pw != "":
				t.Errorf("a refusal returned the password %q", pw)
			}
			if err != nil && strings.Contains(err.Error(), tt.env["DEMO_PASSWORD"]) && tt.env["DEMO_PASSWORD"] != "" {
				t.Errorf("the message repeats the password: %v", err)
			}
		})
	}
}

// The same seed and day give the same archive.
func TestPlanIsDeterministic(t *testing.T) {
	now := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	a, b := newPlan(now), newPlan(now)
	if *a.summary() != *b.summary() {
		t.Fatalf("two plans differ:\n%v\n%v", a.summary(), b.summary())
	}
	if a.works[7].id != b.works[7].id || a.works[7].title != b.works[7].title ||
		a.works[7].chapters[0].content != b.works[7].chapters[0].content {
		t.Error("the same work came out differently")
	}
}

// The volume asked for, and the shape of the data.
func TestPlanVolume(t *testing.T) {
	p := newPlan(time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC))
	s := p.summary()
	within := func(what string, got, lo, hi int) {
		t.Helper()
		if got < lo || got > hi {
			t.Errorf("%s = %d, want %d to %d", what, got, lo, hi)
		}
	}
	within("accounts", s.Accounts, 25, 25)
	within("works", s.Works, 150, 150)
	within("chapters", s.Chapters, 340, 460)
	within("comments", s.Comments, 500, 650)
	within("bookmarks", s.Bookmarks, 180, 200)
	within("series", s.Series, 8, 10)
	within("collections", s.Collections, 3, 3)
	within("drafts", s.Drafts, 5, 5)
	within("anonymous works", s.Anonymous, 6, 6)
	if s.GuestComments == 0 || s.Replies == 0 || s.GuestKudos == 0 || s.PrivateBookmarks == 0 {
		t.Errorf("a kind of data is missing: %+v", s)
	}
	several := 0
	for _, a := range p.accounts {
		if len(a.pseuds) > 1 {
			several++
		}
	}
	within("accounts with several pseuds", several, 5, 12)
	for _, w := range p.works {
		for _, c := range w.chapters {
			if c.words < 100 {
				t.Errorf("work %d chapter %d has only %d words", w.idx, c.number, c.words)
			}
		}
	}
}

// Pseuds of one person must not look like each other, and nobody else's
// either: no word is in two pen names.
func TestPenNamesShareNoWord(t *testing.T) {
	p := newPlan(time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC))
	seen := map[string]string{}
	for _, ps := range p.pseuds {
		var words []string
		for _, f := range strings.FieldsFunc(ps.name, func(r rune) bool { return r == ' ' || r == '_' }) {
			start := 0
			for i, r := range f { // split CamelCase
				if i > 0 && unicode.IsUpper(r) {
					words = append(words, strings.ToLower(f[start:i]))
					start = i
				}
			}
			words = append(words, strings.ToLower(f[start:]))
		}
		for _, w := range words {
			if w == "" || len(w) < 3 {
				continue
			}
			if other, ok := seen[w]; ok && other != ps.name {
				t.Errorf("%q and %q both have %q in them", ps.name, other, w)
			}
			seen[w] = ps.name
		}
	}
}

func count(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
	return n
}

func TestLoad(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	now := time.Now()

	// Something that was already in the archive stays as it is.
	if _, err := db.Exec(`INSERT INTO users (email, password_hash) VALUES ('real@example.com', 'x')`); err != nil {
		t.Fatal(err)
	}

	// Rows the migrations and the test made are not the demo's.
	tagsBefore := count(t, db, `SELECT count(*) FROM tags`)
	usersBefore := count(t, db, `SELECT count(*) FROM users`)

	start := time.Now()
	s, err := Load(ctx, db, testPassword, now)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("loaded in %v: %v", time.Since(start).Round(time.Millisecond), s)
	if s.Skipped {
		t.Fatal("the first load was skipped")
	}

	t.Run("counts match the summary", func(t *testing.T) {
		for _, c := range []struct {
			what string
			want int
			q    string
		}{
			{"accounts", s.Accounts, `SELECT count(*) FROM users WHERE email ~ '^demo[0-9]{2}@example\.org$'`},
			{"pseuds", s.Pseuds, `SELECT count(*) FROM pseuds p JOIN users u ON u.id = p.user_id WHERE u.email ~ '^demo'`},
			{"works", s.Works, `SELECT count(*) FROM works`},
			{"drafts", s.Drafts, `SELECT count(*) FROM works WHERE is_draft`},
			{"anonymous works", s.Anonymous, `SELECT count(*) FROM works WHERE is_anonymous`},
			{"members-only works", s.Restricted, `SELECT count(*) FROM works WHERE restricted`},
			{"chapters", s.Chapters, `SELECT count(*) FROM chapters`},
			{"comments", s.Comments, `SELECT count(*) FROM comments`},
			{"guest comments", s.GuestComments, `SELECT count(*) FROM comments WHERE user_id IS NULL`},
			{"replies", s.Replies, `SELECT count(*) FROM comments WHERE parent_comment_id IS NOT NULL`},
			{"kudos", s.Kudos, `SELECT count(*) FROM kudos`},
			{"guest kudos", s.GuestKudos, `SELECT count(*) FROM kudos WHERE user_id IS NULL`},
			{"bookmarks", s.Bookmarks, `SELECT count(*) FROM bookmarks`},
			{"private bookmarks", s.PrivateBookmarks, `SELECT count(*) FROM bookmarks WHERE is_private`},
			{"series", s.Series, `SELECT count(*) FROM series`},
			{"collections", s.Collections, `SELECT count(*) FROM collections`},
			{"subscriptions", s.Subscriptions, `SELECT count(*) FROM subscriptions`},
			{"blocks", s.Blocks, `SELECT count(*) FROM user_blocks`},
			{"mutes", s.Mutes, `SELECT count(*) FROM user_mutes`},
			{"tags", s.Tags, `SELECT count(*) - ` + fmt.Sprint(tagsBefore) + ` FROM tags`},
			{"creatorships", s.Works + coCreated, `SELECT count(*) FROM creatorships`},
		} {
			if got := count(t, db, c.q); got != c.want {
				t.Errorf("%s: %d in the database, %d in the summary", c.what, got, c.want)
			}
		}
		if got := count(t, db, `SELECT count(*) FROM users`); got != usersBefore+AccountCount {
			t.Errorf("%d accounts in all, want the %d demo ones and the %d that were there", got, AccountCount, usersBefore)
		}
	})

	t.Run("second load adds nothing", func(t *testing.T) {
		before := count(t, db, `SELECT (SELECT count(*) FROM users) + (SELECT count(*) FROM works) + (SELECT count(*) FROM comments) + (SELECT count(*) FROM tags)`)
		again, err := Load(ctx, db, testPassword, now)
		if err != nil {
			t.Fatal(err)
		}
		if !again.Skipped {
			t.Error("the second load was not skipped")
		}
		if after := count(t, db, `SELECT (SELECT count(*) FROM users) + (SELECT count(*) FROM works) + (SELECT count(*) FROM comments) + (SELECT count(*) FROM tags)`); after != before {
			t.Errorf("rows went from %d to %d", before, after)
		}
	})

	t.Run("accounts have a password, no username, no role", func(t *testing.T) {
		if n := count(t, db, `SELECT count(*) FROM users WHERE email ~ '^demo' AND username IS NOT NULL`); n != 0 {
			t.Errorf("%d demo accounts have a username", n)
		}
		if n := count(t, db, `SELECT count(*) FROM user_roles r JOIN users u ON u.id = r.user_id WHERE u.email ~ '^demo' AND r.role <> 'user'`); n != 0 {
			t.Errorf("%d demo accounts hold a role", n)
		}
		if n := count(t, db, `SELECT count(*) FROM users u WHERE u.email ~ '^demo' AND (NOT u.is_active OR NOT u.is_verified)`); n != 0 {
			t.Errorf("%d demo accounts can't log in", n)
		}
		var hash string
		if err := db.QueryRow(`SELECT password_hash FROM users WHERE email = 'demo07@example.org'`).Scan(&hash); err != nil {
			t.Fatal(err)
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(testPassword)) != nil {
			t.Error("the password does not match")
		}
		if n := count(t, db, `SELECT count(*) FROM users u WHERE email ~ '^demo' AND NOT EXISTS (SELECT 1 FROM pseuds p WHERE p.user_id = u.id AND p.is_default)`); n != 0 {
			t.Errorf("%d accounts have no default pseud", n)
		}
		if n := count(t, db, `SELECT count(*) FROM (SELECT user_id FROM pseuds GROUP BY user_id HAVING count(*) > 1) x`); n < 5 {
			t.Errorf("only %d accounts have several pseuds", n)
		}
	})

	// Every total the app keeps must agree with the rows it counts.
	t.Run("counters agree with the rows", func(t *testing.T) {
		for _, c := range []struct{ what, q string }{
			{"chapter_count", `SELECT count(*) FROM works w WHERE chapter_count <> GREATEST(1, (SELECT count(*) FROM chapters c WHERE c.work_id = w.id AND (w.is_draft OR NOT c.is_draft)))`},
			{"word_count", `SELECT count(*) FROM works w WHERE word_count <> (SELECT COALESCE(sum(word_count), 0) FROM chapters c WHERE c.work_id = w.id)`},
			{"kudos_count", `SELECT count(*) FROM works w WHERE kudos_count <> (SELECT count(*) FROM kudos k WHERE k.work_id = w.id AND NOT k.shadowed)`},
			{"comment_count", `SELECT count(*) FROM works w WHERE comment_count <> (SELECT count(*) FROM comments c WHERE c.work_id = w.id AND NOT c.is_deleted AND NOT c.shadowed)`},
			{"bookmark_count", `SELECT count(*) FROM works w WHERE bookmark_count <> (SELECT count(*) FROM bookmarks b WHERE b.work_id = w.id)`},
			{"work_statistics", `SELECT count(*) FROM works w LEFT JOIN work_statistics s ON s.work_id = w.id
				WHERE s.work_id IS NULL OR s.hit_count <> w.hit_count OR s.kudos_count <> w.kudos_count OR s.comment_count <> w.comment_count
				OR s.bookmark_count <> w.bookmark_count OR s.hits <> w.hit_count`},
			{"tag use_count", `SELECT count(*) FROM tags t WHERE use_count <> (SELECT count(*) FROM work_tags wt WHERE wt.tag_id = t.id)`},
			{"reply_count", `SELECT count(*) FROM comments c WHERE reply_count <> (SELECT count(*) FROM comments r WHERE r.parent_comment_id = c.id)`},
			{"thread_level", `SELECT count(*) FROM comments r JOIN comments c ON c.id = r.parent_comment_id WHERE r.thread_level <> c.thread_level + 1`},
			{"series work_count", `SELECT count(*) FROM series s WHERE work_count <> (SELECT count(*) FROM series_works sw WHERE sw.series_id = s.id) OR work_count < 2`},
			{"series word_count", `SELECT count(*) FROM series s WHERE word_count <> (SELECT COALESCE(sum(w.word_count), 0) FROM series_works sw JOIN works w ON w.id = sw.work_id WHERE sw.series_id = s.id)`},
			{"collection work_count", `SELECT count(*) FROM collections c WHERE work_count <> (SELECT count(*) FROM collection_items i WHERE i.collection_id = c.id AND i.is_approved) OR work_count = 0`},
			{"user kudos given", `SELECT count(*) FROM user_statistics s WHERE kudos_given_count <> (SELECT count(*) FROM kudos k WHERE k.user_id = s.user_id)`},
			{"user bookmarks", `SELECT count(*) FROM user_statistics s WHERE bookmarks_count <> (SELECT count(*) FROM bookmarks b WHERE b.user_id = s.user_id)`},
			{"user comments", `SELECT count(*) FROM user_statistics s WHERE comments_count <> (SELECT count(*) FROM comments c WHERE c.user_id = s.user_id)`},
			{"user works", `SELECT count(*) FROM user_statistics s WHERE works_count <> (SELECT count(*) FROM works w WHERE w.user_id = s.user_id AND NOT w.is_draft AND NOT (w.is_anonymous OR w.in_anon_collection OR w.in_unrevealed_collection))`},
			{"user kudos received", `SELECT count(*) FROM user_statistics s WHERE kudos_received_count <> (SELECT count(*) FROM kudos k JOIN works w ON w.id = k.work_id WHERE w.user_id = s.user_id AND NOT (w.is_anonymous OR w.in_anon_collection OR w.in_unrevealed_collection))`},
			{"user series", `SELECT count(*) FROM user_statistics s WHERE series_count <> (SELECT count(*) FROM series x WHERE x.user_id = s.user_id)`},
		} {
			if n := count(t, db, c.q); n != 0 {
				t.Errorf("%s disagrees with the rows for %d records", c.what, n)
			}
		}
	})

	t.Run("works are as the app writes them", func(t *testing.T) {
		for _, c := range []struct{ what, q string }{
			{"a draft with a posted date or a posted chapter", `SELECT count(*) FROM works w WHERE is_draft AND (published_at IS NOT NULL OR status <> 'draft' OR EXISTS (SELECT 1 FROM chapters c WHERE c.work_id = w.id AND NOT c.is_draft))`},
			{"a posted work that is not", `SELECT count(*) FROM works w WHERE NOT is_draft AND (published_at IS NULL OR status <> 'published' OR EXISTS (SELECT 1 FROM chapters c WHERE c.work_id = w.id AND (c.is_draft OR c.published_at IS NULL)))`},
			{"chapter numbers not 1..n", `SELECT count(*) FROM works w WHERE (SELECT max(chapter_number) FROM chapters c WHERE c.work_id = w.id) <> (SELECT count(*) FROM chapters c WHERE c.work_id = w.id)`},
			{"a complete work with another chapter count", `SELECT count(*) FROM works WHERE is_complete AND max_chapters IS DISTINCT FROM chapter_count`},
			{"a work in progress marked complete", `SELECT count(*) FROM works WHERE NOT is_complete AND NOT is_draft AND max_chapters = chapter_count`},
			{"dates outside two years", `SELECT count(*) FROM works WHERE created_at < now() - interval '2 years' OR updated_at < created_at OR updated_at > now()`},
			{"chapters out of date order", `SELECT count(*) FROM chapters a JOIN chapters b ON a.work_id = b.work_id AND b.chapter_number = a.chapter_number + 1 WHERE b.published_at < a.published_at`},
			{"a work before its creator joined", `SELECT count(*) FROM works w JOIN users u ON u.id = w.user_id WHERE w.created_at < u.created_at`},
			{"a comment before its work", `SELECT count(*) FROM comments c JOIN works w ON w.id = c.work_id WHERE c.created_at < w.created_at OR c.created_at > now()`},
			{"a reply before its parent", `SELECT count(*) FROM comments r JOIN comments c ON c.id = r.parent_comment_id WHERE r.created_at < c.created_at`},
			{"a kudos or bookmark before its work", `SELECT (SELECT count(*) FROM kudos k JOIN works w ON w.id = k.work_id WHERE k.created_at < w.created_at) + (SELECT count(*) FROM bookmarks b JOIN works w ON w.id = b.work_id WHERE b.created_at < w.created_at)`},
			{"activity on a draft", `SELECT (SELECT count(*) FROM kudos k JOIN works w ON w.id = k.work_id WHERE w.is_draft) + (SELECT count(*) FROM comments c JOIN works w ON w.id = c.work_id WHERE w.is_draft) + (SELECT count(*) FROM bookmarks b JOIN works w ON w.id = b.work_id WHERE w.is_draft)`},
			{"a guest on a members-only work", `SELECT (SELECT count(*) FROM kudos k JOIN works w ON w.id = k.work_id WHERE w.restricted AND k.user_id IS NULL) + (SELECT count(*) FROM comments c JOIN works w ON w.id = c.work_id WHERE w.restricted AND c.user_id IS NULL)`},
			{"a creator giving kudos to their own work", `SELECT count(*) FROM kudos k JOIN creatorships c ON c.creation_id = k.work_id JOIN pseuds p ON p.id = c.pseud_id WHERE p.user_id = k.user_id`},
			{"an interaction across a block", `SELECT count(*) FROM user_blocks b JOIN comments c ON c.user_id = b.blocked_id JOIN works w ON w.id = c.work_id WHERE w.user_id = b.blocker_id`},
			{"a work without a creator", `SELECT count(*) FROM works w WHERE NOT EXISTS (SELECT 1 FROM creatorships c WHERE c.creation_id = w.id AND c.approved)`},
			{"a work missing a fandom, rating or warning tag", `SELECT count(*) FROM works w WHERE cardinality(fandoms) = 0 OR warnings IS NULL OR NOT EXISTS (SELECT 1 FROM work_tags wt JOIN tags t ON t.id = wt.tag_id WHERE wt.work_id = w.id AND t.type = 'warning')`},
			{"a denormalised tag that has no link", `SELECT count(*) FROM works w, unnest(w.fandoms || w.characters || w.relationships || w.freeform_tags) n WHERE NOT EXISTS (SELECT 1 FROM work_tags wt JOIN tags t ON t.id = wt.tag_id WHERE wt.work_id = w.id AND t.name = n)`},
			{"an empty search vector", `SELECT count(*) FROM works WHERE search_vector IS NULL OR search_vector = ''::tsvector`},
			{"an anonymous work not in a hiding state", `SELECT count(*) FROM works WHERE (SELECT count(*) FROM works WHERE in_anon_collection) <> 5 OR (SELECT count(*) FROM works WHERE in_unrevealed_collection) <> 4`},
			{"a collection holding a work it should not", `SELECT count(*) FROM collection_items i JOIN collections c ON c.id = i.collection_id JOIN works w ON w.id = i.work_id WHERE w.is_draft OR (c.is_anonymous AND NOT w.in_anon_collection) OR (c.name = 'lantern-swap' AND NOT w.in_unrevealed_collection)`},
			{"a bookmark under another account's pseud", `SELECT count(*) FROM bookmarks b JOIN pseuds p ON p.id = b.pseud_id WHERE p.user_id <> b.user_id`},
			{"a comment under another account's pseud", `SELECT count(*) FROM comments c JOIN pseuds p ON p.id = c.pseudonym_id WHERE p.user_id <> c.user_id`},
			{"a chapter comment on another work's chapter", `SELECT count(*) FROM comments c JOIN chapters ch ON ch.id = c.chapter_id WHERE ch.work_id <> c.work_id`},
			{"a subscription to something unseen", `SELECT count(*) FROM subscriptions s WHERE NOT ((s.type = 'author' AND EXISTS (SELECT 1 FROM pseuds WHERE id = s.target_id)) OR (s.type = 'work' AND EXISTS (SELECT 1 FROM works WHERE id = s.target_id)) OR (s.type = 'series' AND EXISTS (SELECT 1 FROM series WHERE id = s.target_id)))`},
		} {
			if n := count(t, db, c.q); n != 0 {
				t.Errorf("%s: %d", c.what, n)
			}
		}
		if n := count(t, db, `SELECT count(*) FROM works WHERE restricted`); n < 10 {
			t.Errorf("only %d members-only works", n)
		}
		if n := count(t, db, `SELECT count(*) FROM works WHERE NOT is_draft AND NOT is_complete`); n < 5 {
			t.Errorf("only %d works in progress", n)
		}
		if n := count(t, db, `SELECT count(*) FROM works WHERE chapter_count >= 15`); n < 1 {
			t.Error("no long work")
		}
	})

	t.Run("tags", func(t *testing.T) {
		for _, kind := range []string{"fandom", "character", "relationship", "freeform", "warning", "category"} {
			if n := count(t, db, `SELECT count(*) FROM tags WHERE type = $1`, kind); n == 0 {
				t.Errorf("no %s tags", kind)
			}
		}
		for _, f := range []string{"Sherlock Holmes - Arthur Conan Doyle", "Pride and Prejudice - Jane Austen", "Les Misérables - Victor Hugo",
			"Dracula - Bram Stoker", "Frankenstein - Mary Shelley", "Little Women - Louisa May Alcott", "Arthurian Mythology", "Original Work"} {
			if n := count(t, db, `SELECT use_count FROM tags WHERE name = $1 AND type = 'fandom'`, f); n == 0 {
				t.Errorf("fandom %q is not used", f)
			}
		}
		if n := count(t, db, `SELECT count(*) FROM tags WHERE NOT is_canonical`); n != 0 {
			t.Errorf("%d tags are not canonical", n)
		}
	})

	t.Run("notifications are dated with their comments", func(t *testing.T) {
		if n := count(t, db, `SELECT count(*) FROM notifications`); n == 0 {
			t.Fatal("the comment trigger wrote no notifications")
		}
		if n := count(t, db, `SELECT count(*) FROM notifications WHERE created_at > now() - interval '1 hour'`); n != 0 {
			t.Errorf("%d notifications are dated now", n)
		}
	})
}

// A load that fails leaves nothing behind.
func TestLoadIsOneTransaction(t *testing.T) {
	db := testdb.New(t)
	// A table the loader needs, made impossible to write to.
	usersBefore := count(t, db, `SELECT count(*) FROM users`)
	tagsBefore := count(t, db, `SELECT count(*) FROM tags`)
	if _, err := db.Exec(`ALTER TABLE user_mutes ADD CONSTRAINT never CHECK (false) NOT VALID`); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(context.Background(), db, testPassword, time.Now()); err == nil {
		t.Fatal("the load worked")
	}
	for _, table := range []string{"works", "comments", "chapters", "kudos", "bookmarks", "creatorships", "work_tags"} {
		if n := count(t, db, `SELECT count(*) FROM `+table); n != 0 {
			t.Errorf("%d rows left in %s", n, table)
		}
	}
	// What the migrations put there is all that is left.
	if n := count(t, db, `SELECT count(*) FROM users`); n != usersBefore {
		t.Errorf("%d accounts, was %d", n, usersBefore)
	}
	if n := count(t, db, `SELECT count(*) FROM tags`); n != tagsBefore {
		t.Errorf("%d tags, was %d", n, tagsBefore)
	}
}
