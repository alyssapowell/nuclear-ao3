package works

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"nuclear-ao3/internal/testdb"
)

const dirty = `<p>hi</p><img src=x onerror=alert(1)><script>alert(2)</script><a href="javascript:alert(3)" onclick="x()">link</a>`

func assertClean(t *testing.T, where, got string) {
	t.Helper()
	for _, bad := range []string{"onerror", "onclick", "<script", "javascript:", "alert("} {
		if strings.Contains(got, bad) {
			t.Errorf("%s still contains %q: %s", where, bad, got)
		}
	}
	if !strings.Contains(got, "<p>hi</p>") {
		t.Errorf("%s lost its good content: %s", where, got)
	}
}

func TestHandlersStoreCleanedHTML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testdb.New(t)
	cfg := &TestDBConfig{DB: db}
	userID, _, err := cfg.CreateTestUser("writer", "writer@example.org")
	if err != nil {
		t.Fatal(err)
	}
	ws := &WorkService{db: db}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", userID.String()) })
	r.POST("/works", ws.CreateWork)
	r.PUT("/works/:work_id", ws.UpdateWork)
	r.POST("/works/:work_id/chapters", ws.CreateChapter)
	r.PUT("/works/:work_id/chapters/:chapter_id", ws.UpdateChapter)
	r.POST("/works/:work_id/comments", ws.CreateComment)

	send := func(method, path string, body any) map[string]any {
		t.Helper()
		b, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code >= 300 {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body)
		}
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}

	created := send("POST", "/works", map[string]any{
		"title": "T", "summary": dirty, "notes": dirty, "language": "en", "rating": "general",
		"fandoms": []string{"F"}, "chapter_summary": dirty, "chapter_notes": dirty,
		"chapter_end_notes": dirty, "chapter_content": dirty,
	})
	workID := created["work"].(map[string]any)["id"].(string)

	var summary, notes, cSummary, cNotes, cEnd, cContent string
	db.QueryRow(`SELECT summary, notes FROM works WHERE id = $1`, workID).Scan(&summary, &notes)
	db.QueryRow(`SELECT summary, notes, end_notes, content FROM chapters WHERE work_id = $1`, workID).Scan(&cSummary, &cNotes, &cEnd, &cContent)
	for name, v := range map[string]string{"work summary": summary, "work notes": notes, "chapter summary": cSummary,
		"chapter notes": cNotes, "chapter end notes": cEnd, "chapter content": cContent} {
		assertClean(t, name, v)
	}

	send("PUT", "/works/"+workID, map[string]any{"summary": dirty, "notes": dirty, "status": "posted"})
	db.QueryRow(`SELECT summary, notes FROM works WHERE id = $1`, workID).Scan(&summary, &notes)
	assertClean(t, "updated work summary", summary)
	assertClean(t, "updated work notes", notes)

	chapter := send("POST", "/works/"+workID+"/chapters", map[string]any{
		"title": "Two", "summary": dirty, "notes": dirty, "end_notes": dirty, "content": dirty, "status": "posted",
	})
	chapterID := chapter["chapter"].(map[string]any)["id"].(string)
	send("PUT", "/works/"+workID+"/chapters/"+chapterID, map[string]any{"content": dirty, "notes": dirty})
	db.QueryRow(`SELECT summary, notes, end_notes, content FROM chapters WHERE id = $1`, chapterID).Scan(&cSummary, &cNotes, &cEnd, &cContent)
	for name, v := range map[string]string{"new chapter summary": cSummary, "new chapter notes": cNotes,
		"new chapter end notes": cEnd, "new chapter content": cContent} {
		assertClean(t, name, v)
	}

	send("POST", "/works/"+workID+"/comments", map[string]any{"content": dirty})
	var comment string
	if err := db.QueryRow(`SELECT content FROM comments WHERE work_id = $1`, workID).Scan(&comment); err != nil {
		t.Fatal(err)
	}
	assertClean(t, "comment", comment)
}
