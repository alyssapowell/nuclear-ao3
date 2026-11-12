package works

import (
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"

	"nuclear-ao3/internal/platform"
)

var plainText = bluemonday.StripTagsPolicy()

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// excerpt is the start of a description as plain text, at most n characters.
func excerpt(description string, n int) string {
	text := html.UnescapeString(plainText.Sanitize(description))
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	return strings.TrimSpace(string([]rune(text)[:n])) + "…"
}

// SearchPseuds is "People Search" (AO3: /people/search). Here the people are
// pseuds: each result is one pseud's name, a short plain-text excerpt of its
// description, its icon and the number of its works that the viewer may read.
//
// Parameters: q (part of the pseud's name or description, case-insensitive),
// page and limit (default 20, at most 100).
//
// Nothing here ties pseuds of one account together. The results carry no
// account id and no grouping, and their order is relevance, then name, then
// id, so two pseuds of an account sit next to each other only when their names
// do. The work count counts only published works the viewer may read, never a
// work that hides its creator (anonymous, or in an anonymous or unrevealed
// collection), and is worked out as a plain reader (an app is treated like
// any other logged-in reader, as everywhere: platform.ReaderViewer).
func (ws *WorkService) SearchPseuds(c *gin.Context) {
	viewer, asReader := platform.ReaderViewer(c)
	query := strings.TrimSpace(c.Query("q"))
	page, limit, offset := pageParams(c)

	// The search text is $1 in the count and $3 in the listing, which also
	// takes the viewer ($1) and as-reader ($2) for the work counts.
	filter := func(p string) string {
		if query == "" {
			return " WHERE u.is_active"
		}
		return " WHERE u.is_active AND (lower(p.name::text) LIKE lower(" + p + ") OR lower(coalesce(p.description, '')) LIKE lower(" + p + "))"
	}
	like := "%" + likeEscaper.Replace(query) + "%"

	var total int
	countArgs := []interface{}{}
	args := []interface{}{viewer, asReader}
	rank := "0"
	if query != "" {
		countArgs = append(countArgs, like)
		// Best match first: the name is exactly the search, then starts with
		// it, then has it somewhere, then only the description does.
		args = append(args, like, query, likeEscaper.Replace(query)+"%")
		rank = `CASE WHEN lower(p.name::text) = lower($4) THEN 3
			WHEN lower(p.name::text) LIKE lower($5) THEN 2
			WHEN lower(p.name::text) LIKE lower($3) THEN 1 ELSE 0 END`
	}
	if err := ws.db.QueryRowContext(c.Request.Context(),
		`SELECT COUNT(*) FROM pseuds p JOIN users u ON u.id = p.user_id`+filter("$1"), countArgs...).Scan(&total); err != nil {
		log.Printf("pseud search: count: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}
	where := filter("$3")
	n := len(args)
	rows, err := ws.db.QueryContext(c.Request.Context(), `
		SELECT p.id, p.name, coalesce(p.description, ''), coalesce(p.icon_url, ''), coalesce(p.icon_alt_text, ''),
		       (SELECT count(*) FROM creatorships cr JOIN works w ON w.id = cr.creation_id
		        WHERE cr.pseud_id = p.id AND cr.creation_type = 'Work' AND cr.approved
		          AND w.status = 'published' AND can_user_view_work(w.id, $1::uuid, $2::boolean)`+hidesNoCreator+`)
		FROM pseuds p JOIN users u ON u.id = p.user_id`+where+`
		ORDER BY `+rank+` DESC, lower(p.name::text), p.name, p.id
		LIMIT $`+strconv.Itoa(n+1)+` OFFSET $`+strconv.Itoa(n+2), append(args, limit, offset)...)
	if err != nil {
		log.Printf("pseud search: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}
	defer rows.Close()

	type result struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		IconURL     string `json:"icon_url"`
		IconAlt     string `json:"icon_alt_text"`
		WorkCount   int    `json:"work_count"`
	}
	results := []result{}
	for rows.Next() {
		var r result
		var desc string
		if err := rows.Scan(&r.ID, &r.Name, &desc, &r.IconURL, &r.IconAlt, &r.WorkCount); err != nil {
			log.Printf("pseud search: scan: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
			return
		}
		r.Description = excerpt(desc, 200)
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		log.Printf("pseud search: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"pseuds": results,
		"pagination": gin.H{
			"page": page, "limit": limit, "total": total, "total_pages": (total + limit - 1) / limit,
		},
	})
}
