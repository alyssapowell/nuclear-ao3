package search

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"nuclear-ao3/internal/platform"
)

// The deepest result a search can page to. Past this, Postgres has to skip
// over too many rows; readers should narrow the search instead.
const maxResultDepth = 10000

// WorkQuery is one work search: free text plus AO3's filters.
type WorkQuery struct {
	// AsReader is true when an app asks for a person: they are then a plain
	// logged-in reader, with no creator, owner or staff exceptions and no
	// count of their own shadowed posts (platform.ReaderViewer). Set by the
	// handler, never read from the request.
	AsReader bool

	Text    string // anything: title, tags, summary
	Title   string // words that must appear in the title
	Creator string // a pseud name; anonymous works never match

	Fandoms, Characters, Relationships, Tags []string // must have all of these
	ExcludeTags                              []string // must have none of these, in any tag field

	Ratings    []string // any of these
	Warnings   []string // all of these
	Categories []string // all of these
	Languages  []string // any of these

	WordCountMin, WordCountMax int    // 0 means no limit
	KudosMin, CommentsMin      int    // 0 means no limit
	Complete                   string // "complete", "in_progress", or "" for both
	Crossovers                 string // "exclude", "only", or "" for both
	SingleChapter              bool
	UpdatedWithin              string // "week", "month", "3months", "year", or ""

	Sort  string // see sortColumns; "relevance" by default
	Order string // "asc" or "desc"; each sort has a sensible default
	Page  int
	Limit int
}

// The form shows AO3's rating names; the database stores short ones.
var ratingValues = map[string]string{
	"general audiences":     "general",
	"teen and up audiences": "teen",
	"mature":                "mature",
	"explicit":              "explicit",
	"not rated":             "not_rated",
}

var updatedWithin = map[string]string{
	"week":    "7 days",
	"month":   "1 month",
	"3months": "3 months",
	"year":    "1 year",
}

// What each sort orders by, and whether it runs high-to-low by default.
var sortColumns = map[string]struct {
	expr string
	desc bool
}{
	"updated_at":   {"w.updated_at", true},
	"published_at": {"w.published_at", true},
	"title":        {"lower(w.title)", false},
	"word_count":   {"w.word_count", true},
	"hits":         {"w.hit_count", true},
	"kudos":        {"w.kudos_count", true},
	"comments":     {"w.comment_count", true},
	"bookmarks":    {"w.bookmark_count", true},
}

// ParseWorkQuery reads a search from the URL's query string. Lists can be
// given as repeated parameters (?fandoms=A&fandoms=B).
func ParseWorkQuery(c *gin.Context) (WorkQuery, error) {
	q := WorkQuery{
		Text:          strings.TrimSpace(c.Query("q")),
		Title:         strings.TrimSpace(c.Query("title")),
		Creator:       strings.TrimSpace(c.Query("creator")),
		Fandoms:       nonEmpty(c.QueryArray("fandoms")),
		Characters:    nonEmpty(c.QueryArray("characters")),
		Relationships: nonEmpty(c.QueryArray("relationships")),
		Tags:          nonEmpty(c.QueryArray("tags")),
		ExcludeTags:   nonEmpty(c.QueryArray("exclude_tags")),
		Warnings:      nonEmpty(c.QueryArray("warnings")),
		Categories:    nonEmpty(c.QueryArray("category")),
		Languages:     nonEmpty(c.QueryArray("languages")),
		Complete:      c.Query("status"),
		Crossovers:    c.Query("crossovers"),
		SingleChapter: c.Query("single_chapter") == "true",
		UpdatedWithin: c.Query("updated_within"),
		Sort:          c.DefaultQuery("sort", "relevance"),
		Order:         c.Query("order"),
	}
	for _, r := range nonEmpty(c.QueryArray("rating")) {
		if v, ok := ratingValues[strings.ToLower(r)]; ok {
			r = v
		}
		q.Ratings = append(q.Ratings, r)
	}

	var err error
	if q.WordCountMin, err = intParam(c, "word_count_min", 0); err != nil {
		return q, err
	}
	if q.WordCountMax, err = intParam(c, "word_count_max", 0); err != nil {
		return q, err
	}
	if q.KudosMin, err = intParam(c, "kudos_min", 0); err != nil {
		return q, err
	}
	if q.CommentsMin, err = intParam(c, "comments_min", 0); err != nil {
		return q, err
	}
	if q.Page, err = intParam(c, "page", 1); err != nil {
		return q, err
	}
	if q.Limit, err = intParam(c, "limit", 20); err != nil {
		return q, err
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}
	if q.Page*q.Limit > maxResultDepth {
		return q, fmt.Errorf("can't page past result %d; narrow the search instead", maxResultDepth)
	}
	if q.Complete == "in-progress" {
		q.Complete = "in_progress"
	}
	if q.Complete != "" && q.Complete != "complete" && q.Complete != "in_progress" {
		q.Complete = ""
	}
	if _, ok := sortColumns[q.Sort]; !ok {
		q.Sort = "relevance"
	}
	return q, nil
}

func intParam(c *gin.Context, name string, def int) (int, error) {
	s := c.Query(name)
	if s == "" {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a whole number", name)
	}
	return n, nil
}

func nonEmpty(values []string) []string {
	var out []string
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// sqlBuilder collects WHERE conditions and their arguments.
type sqlBuilder struct {
	where []string
	args  []any
}

// arg adds a query argument and returns its placeholder ($1, $2, ...).
func (b *sqlBuilder) arg(v any) string {
	b.args = append(b.args, v)
	return "$" + strconv.Itoa(len(b.args))
}

func (b *sqlBuilder) add(cond string) { b.where = append(b.where, cond) }

// The warnings and category columns hold a Postgres array written as text
// ("{a,b}"), or occasionally a single plain value. This reads either.
func textList(col string) string {
	return fmt.Sprintf(`(CASE WHEN %[1]s LIKE '{%%}' THEN %[1]s::text[]
		WHEN coalesce(%[1]s, '') = '' THEN '{}'::text[] ELSE ARRAY[%[1]s] END)`, col)
}

// build turns a query into WHERE conditions. The text query's placeholder is
// returned so the caller can rank by it.
func (q WorkQuery) build(viewer *uuid.UUID) (b sqlBuilder, textArg string) {
	// The same rule the work page uses: drafts, works restricted to users,
	// blocks and mutes are all decided by can_user_view_work.
	b.add("w.status <> 'draft' AND w.is_draft = false")
	b.add("NOT coalesce(w.in_unrevealed_collection, false)")
	viewerArg := b.arg(viewer)
	b.add(fmt.Sprintf("can_user_view_work(w.id, %s, %s::boolean)", viewerArg, b.arg(q.AsReader)))

	if q.Text != "" {
		textArg = b.arg(q.Text)
		b.add(fmt.Sprintf("w.search_vector @@ websearch_to_tsquery('english', %s)", textArg))
	}
	if q.Title != "" {
		b.add(fmt.Sprintf("strpos(lower(w.title), lower(%s)) > 0", b.arg(q.Title)))
	}
	if q.Creator != "" {
		b.add(fmt.Sprintf(`NOT (coalesce(w.is_anonymous, false) OR coalesce(w.in_anon_collection, false))
			AND EXISTS (SELECT 1 FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id
				WHERE c.creation_id = w.id AND c.creation_type = 'Work' AND c.approved
				AND lower(p.name) = lower(%s))`, b.arg(q.Creator)))
	}
	for col, tags := range map[string][]string{
		"w.fandoms": q.Fandoms, "w.characters": q.Characters,
		"w.relationships": q.Relationships, "w.freeform_tags": q.Tags,
	} {
		if len(tags) > 0 {
			b.add(fmt.Sprintf("%s @> %s", col, b.arg(pq.Array(tags))))
		}
	}
	if len(q.ExcludeTags) > 0 {
		ex := b.arg(pq.Array(q.ExcludeTags))
		b.add(fmt.Sprintf(`NOT (coalesce(w.fandoms, '{}') && %[1]s OR coalesce(w.characters, '{}') && %[1]s
			OR coalesce(w.relationships, '{}') && %[1]s OR coalesce(w.freeform_tags, '{}') && %[1]s)`, ex))
	}
	if len(q.Ratings) > 0 {
		b.add(fmt.Sprintf("w.rating = ANY(%s)", b.arg(pq.Array(q.Ratings))))
	}
	if len(q.Warnings) > 0 {
		b.add(fmt.Sprintf("%s @> %s", textList("w.warnings"), b.arg(pq.Array(q.Warnings))))
	}
	if len(q.Categories) > 0 {
		b.add(fmt.Sprintf("%s @> %s", textList("w.category"), b.arg(pq.Array(q.Categories))))
	}
	if len(q.Languages) > 0 {
		b.add(fmt.Sprintf("w.language = ANY(%s)", b.arg(pq.Array(q.Languages))))
	}
	if q.WordCountMin > 0 {
		b.add(fmt.Sprintf("w.word_count >= %s", b.arg(q.WordCountMin)))
	}
	if q.WordCountMax > 0 {
		b.add(fmt.Sprintf("w.word_count <= %s", b.arg(q.WordCountMax)))
	}
	// Filtering by kudos or comments, like sorting by them, uses the public
	// count, not the count the viewer sees (which includes their own shadowed
	// items). It is a trade-off: a silently blocked person can still see the
	// filter treat them as absent, but a filter's cut-off is a blunt tool and
	// not a precise way to detect a block.
	if q.KudosMin > 0 {
		b.add(fmt.Sprintf("w.kudos_count >= %s", b.arg(q.KudosMin)))
	}
	if q.CommentsMin > 0 {
		b.add(fmt.Sprintf("w.comment_count >= %s", b.arg(q.CommentsMin)))
	}
	switch q.Complete {
	case "complete":
		b.add("w.is_complete")
	case "in_progress":
		b.add("NOT coalesce(w.is_complete, false)")
	}
	switch q.Crossovers {
	case "only":
		b.add("cardinality(coalesce(w.fandoms, '{}')) > 1")
	case "exclude":
		b.add("cardinality(coalesce(w.fandoms, '{}')) <= 1")
	}
	if q.SingleChapter {
		b.add("w.chapter_count = 1")
	}
	if interval, ok := updatedWithin[q.UpdatedWithin]; ok {
		b.add(fmt.Sprintf("w.updated_at >= now() - %s::interval", b.arg(interval)))
	}
	return b, textArg
}

func (q WorkQuery) orderBy(textArg string) string {
	col, ok := sortColumns[q.Sort]
	if !ok { // relevance
		if textArg == "" {
			col = sortColumns["updated_at"]
		} else {
			col.expr = fmt.Sprintf("ts_rank_cd(w.search_vector, websearch_to_tsquery('english', %s))", textArg)
			col.desc = true
		}
	}
	desc := col.desc
	if q.Order == "asc" {
		desc = false
	} else if q.Order == "desc" {
		desc = true
	}
	dir := "ASC"
	if desc {
		dir = "DESC NULLS LAST"
	}
	return fmt.Sprintf("%s %s, w.id", col.expr, dir)
}

// Author is how a work's creators appear in the API: by pseud only. Messages
// to a creator are sent to the pseud's id through the API.
type Author struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// WorkResult is one work in the search results.
type WorkResult struct {
	ID            uuid.UUID  `json:"id"`
	Title         string     `json:"title"`
	Summary       string     `json:"summary"`
	Authors       []Author   `json:"authors"` // empty for anonymous works
	Anonymous     bool       `json:"anonymous"`
	Rating        string     `json:"rating"`
	Warnings      []string   `json:"warnings"`
	Categories    []string   `json:"categories"`
	Language      string     `json:"language"`
	Fandoms       []string   `json:"fandoms"`
	Characters    []string   `json:"characters"`
	Relationships []string   `json:"relationships"`
	FreeformTags  []string   `json:"freeform_tags"`
	WordCount     int        `json:"word_count"`
	ChapterCount  int        `json:"chapter_count"`
	MaxChapters   *int       `json:"max_chapters"`
	IsComplete    bool       `json:"is_complete"`
	KudosCount    int        `json:"kudos_count"`
	HitCount      int        `json:"hit_count"`
	CommentCount  int        `json:"comment_count"`
	BookmarkCount int        `json:"bookmark_count"`
	PublishedAt   *time.Time `json:"published_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Facet is one filter value and how many results have it.
type Facet struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// WorkResults is a page of results, the total, and the filter counts AO3
// shows beside them.
type WorkResults struct {
	Works      []WorkResult       `json:"works"`
	Total      int                `json:"total"`
	Pagination map[string]int     `json:"pagination"`
	Facets     map[string][]Facet `json:"facets"`
}

// How many of the most common values each facet lists.
const facetSize = 10

// Search runs a work search for viewer (nil when logged out).
func (s *Service) Search(q WorkQuery, viewer *uuid.UUID) (*WorkResults, error) {
	b, textArg := q.build(viewer)
	where := strings.Join(b.where, "\n  AND ")

	out := &WorkResults{Works: []WorkResult{}, Facets: map[string][]Facet{}}

	// The total and the filter counts, from one pass over the matching works.
	facetSQL := fmt.Sprintf(`WITH m AS (
  SELECT w.rating, w.fandoms, w.characters, w.relationships, w.freeform_tags
  FROM works w WHERE %s)
SELECT 'total', '', count(*) FROM m
UNION ALL (SELECT 'rating', rating, count(*) FROM m GROUP BY rating ORDER BY 3 DESC)
UNION ALL (SELECT 'fandoms', t, count(*) FROM m, unnest(m.fandoms) t GROUP BY t ORDER BY 3 DESC, 2 LIMIT %[2]d)
UNION ALL (SELECT 'characters', t, count(*) FROM m, unnest(m.characters) t GROUP BY t ORDER BY 3 DESC, 2 LIMIT %[2]d)
UNION ALL (SELECT 'relationships', t, count(*) FROM m, unnest(m.relationships) t GROUP BY t ORDER BY 3 DESC, 2 LIMIT %[2]d)
UNION ALL (SELECT 'freeform_tags', t, count(*) FROM m, unnest(m.freeform_tags) t GROUP BY t ORDER BY 3 DESC, 2 LIMIT %[2]d)`,
		where, facetSize)
	rows, err := s.db.Query(facetSQL, b.args...)
	if err != nil {
		return nil, fmt.Errorf("count results: %w", err)
	}
	for rows.Next() {
		var kind, name string
		var n int
		if err := rows.Scan(&kind, &name, &n); err != nil {
			rows.Close()
			return nil, err
		}
		if kind == "total" {
			out.Total = n
		} else {
			out.Facets[kind] = append(out.Facets[kind], Facet{Name: name, Count: n})
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// The counts shown include the viewer's own shadowed kudos and comments.
	// (The sort above and the filters in build() keep the public counts.)
	viewerArg := ownerExpr(b.arg(viewer), b.arg(q.AsReader))
	pageSQL := fmt.Sprintf(`SELECT w.id, w.title, coalesce(w.summary, ''), w.rating,
  %s, %s, w.language,
  coalesce(w.fandoms, '{}'), coalesce(w.characters, '{}'), coalesce(w.relationships, '{}'), coalesce(w.freeform_tags, '{}'),
  w.word_count, w.chapter_count, coalesce(w.max_chapters, w.expected_chapters), coalesce(w.is_complete, false),
  w.kudos_count + own_shadowed_kudos(w.id, %[7]s), w.hit_count,
  w.comment_count + own_shadowed_comments(w.id, %[7]s), w.bookmark_count, w.published_at, w.updated_at,
  coalesce(w.is_anonymous, false) OR coalesce(w.in_anon_collection, false) AS anonymous,
  CASE WHEN coalesce(w.is_anonymous, false) OR coalesce(w.in_anon_collection, false) THEN '[]'::json
  ELSE (SELECT coalesce(json_agg(json_build_object('id', p.id, 'name', p.name) ORDER BY p.name), '[]'::json)
        FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id
        WHERE c.creation_id = w.id AND c.creation_type = 'Work' AND c.approved) END
FROM works w
WHERE %[3]s
ORDER BY %[4]s
LIMIT %[5]d OFFSET %[6]d`,
		textList("w.warnings"), textList("w.category"), where, q.orderBy(textArg),
		q.Limit, (q.Page-1)*q.Limit, viewerArg)
	rows, err = s.db.Query(pageSQL, b.args...)
	if err != nil {
		return nil, fmt.Errorf("fetch results: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var r WorkResult
		var authors []byte
		var maxChapters sql.NullInt64
		if err := rows.Scan(&r.ID, &r.Title, &r.Summary, &r.Rating,
			(*pq.StringArray)(&r.Warnings), (*pq.StringArray)(&r.Categories), &r.Language,
			(*pq.StringArray)(&r.Fandoms), (*pq.StringArray)(&r.Characters),
			(*pq.StringArray)(&r.Relationships), (*pq.StringArray)(&r.FreeformTags),
			&r.WordCount, &r.ChapterCount, &maxChapters, &r.IsComplete,
			&r.KudosCount, &r.HitCount, &r.CommentCount, &r.BookmarkCount, &r.PublishedAt, &r.UpdatedAt,
			&r.Anonymous, &authors); err != nil {
			return nil, err
		}
		if maxChapters.Valid {
			n := int(maxChapters.Int64)
			r.MaxChapters = &n
		}
		if err := json.Unmarshal(authors, &r.Authors); err != nil {
			return nil, err
		}
		out.Works = append(out.Works, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	pages := (out.Total + q.Limit - 1) / q.Limit
	out.Pagination = map[string]int{"page": q.Page, "limit": q.Limit, "total": out.Total, "pages": pages}
	return out, nil
}

// SearchWorks handles GET /api/v1/search/works.
func (s *Service) SearchWorks(c *gin.Context) {
	q, err := ParseWorkQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	viewer, asReader := platform.ReaderViewer(c)
	q.AsReader = asReader
	res, err := s.Search(q, viewer)
	if err != nil {
		log.Printf("search: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}
	c.JSON(http.StatusOK, res)
}

// prefixQuery turns what someone has typed so far into a tsquery that
// matches words starting with each typed word: "harr pot" becomes
// 'harr':* & 'pot':*. Anything but letters and digits is dropped, so the
// result is always a valid query.
func prefixQuery(typed string) string {
	words := strings.FieldsFunc(typed, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	for i, w := range words {
		words[i] = "'" + w + "':*"
	}
	return strings.Join(words, " & ")
}

// likePrefix escapes LIKE's wildcards so typed text is matched literally.
func likePrefix(typed string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(typed) + "%"
}

// Suggestions handles GET /api/v1/search/suggestions?q=..., used while typing
// in the search box: a few works, tags and pseuds that start with the text.
func (s *Service) Suggestions(c *gin.Context) {
	typed := strings.TrimSpace(c.Query("q"))
	type workSuggestion struct {
		ID    uuid.UUID `json:"id"`
		Title string    `json:"title"`
		Kudos int       `json:"kudos"`
	}
	type tagSuggestion struct {
		Name  string `json:"name"`
		Type  string `json:"type"`
		Count int    `json:"count"`
	}
	out := struct {
		Works   []workSuggestion `json:"works"`
		Tags    []tagSuggestion  `json:"tags"`
		Authors []Author         `json:"authors"`
	}{[]workSuggestion{}, []tagSuggestion{}, []Author{}}

	tsq := prefixQuery(typed)
	if len([]rune(typed)) < 2 || tsq == "" {
		c.JSON(http.StatusOK, out)
		return
	}

	fail := func(err error) {
		log.Printf("search suggestions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Suggestions failed"})
	}

	b := sqlBuilder{}
	b.add("w.status <> 'draft' AND w.is_draft = false")
	b.add("NOT coalesce(w.in_unrevealed_collection, false)")
	viewer, asReader := platform.ReaderViewer(c)
	viewerArg := b.arg(viewer)
	readerArg := b.arg(asReader)
	b.add(fmt.Sprintf("can_user_view_work(w.id, %s, %s::boolean)", viewerArg, readerArg))
	b.add(fmt.Sprintf("w.search_vector @@ to_tsquery('english', %s)", b.arg(tsq)))
	// The count shown includes the viewer's own shadowed kudos; the order
	// uses the public count.
	rows, err := s.db.Query(`SELECT w.id, w.title, w.kudos_count + own_shadowed_kudos(w.id, `+ownerExpr(viewerArg, readerArg)+`)
		FROM works w WHERE `+
		strings.Join(b.where, " AND ")+` ORDER BY w.kudos_count DESC, w.id LIMIT 5`, b.args...)
	if err != nil {
		fail(err)
		return
	}
	for rows.Next() {
		var w workSuggestion
		if err := rows.Scan(&w.ID, &w.Title, &w.Kudos); err != nil {
			rows.Close()
			fail(err)
			return
		}
		out.Works = append(out.Works, w)
	}
	rows.Close()

	rows, err = s.db.Query(`SELECT name, type, coalesce(use_count, 0) FROM tags
		WHERE name LIKE $1 ORDER BY use_count DESC NULLS LAST, name LIMIT 5`, likePrefix(typed))
	if err != nil {
		fail(err)
		return
	}
	for rows.Next() {
		var t tagSuggestion
		if err := rows.Scan(&t.Name, &t.Type, &t.Count); err != nil {
			rows.Close()
			fail(err)
			return
		}
		out.Tags = append(out.Tags, t)
	}
	rows.Close()

	rows, err = s.db.Query(`SELECT id, name FROM pseuds
		WHERE lower(name) LIKE lower($1) ORDER BY name LIMIT 5`, likePrefix(typed))
	if err != nil {
		fail(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var a Author
		if err := rows.Scan(&a.ID, &a.Name); err != nil {
			fail(err)
			return
		}
		out.Authors = append(out.Authors, a)
	}
	c.JSON(http.StatusOK, out)
}

// ownerExpr is the SQL for the viewer as a creator or owner would count: NULL
// when the as-reader placeholder is true (an app), the viewer otherwise. The
// arguments are placeholders such as "$3".
func ownerExpr(viewerArg, readerArg string) string {
	return "(CASE WHEN " + readerArg + "::boolean THEN NULL ELSE " + viewerArg + "::uuid END)"
}
