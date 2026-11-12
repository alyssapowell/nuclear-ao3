package works_test

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// get makes a request and returns the whole response.
func (e *fixEnv) get(path, token string) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func dl(id uuid.UUID, format string) string {
	return "/api/v1/works/" + id.String() + "/download." + format
}

// unzip returns every file of an EPUB, in archive order.
func unzipEPUB(t *testing.T, b []byte) (*zip.Reader, map[string]string) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	files := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		files[f.Name] = string(data)
	}
	return zr, files
}

// wellFormed fails the test unless s parses as XML from start to end.
func wellFormed(t *testing.T, name, s string) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(s))
	for {
		_, err := d.Token()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Errorf("%s is not well-formed XML: %v", name, err)
			return
		}
	}
}

func TestEveryFormatDownloadsForAPublicWork(t *testing.T) {
	e := newFixEnv(t)
	writer, _ := e.user("writer")
	work := e.work(writer, "A Public Tale", "published")
	e.exec(`UPDATE chapters SET content = '<p>Once upon <em>a time</em>.</p>' WHERE work_id = $1`, work)

	want := map[string]string{
		"epub": "application/epub+zip",
		"html": "text/html; charset=utf-8",
		"txt":  "text/plain; charset=utf-8",
	}
	for format, ctype := range want {
		w := e.get(dl(work, format), "")
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", format, w.Code, w.Body.String())
		}
		if got := w.Header().Get("Content-Type"); got != ctype {
			t.Errorf("%s: Content-Type %q, want %q", format, got, ctype)
		}
		cd := w.Header().Get("Content-Disposition")
		if !strings.HasPrefix(cd, "attachment;") || !strings.Contains(cd, `filename="A Public Tale.`+format+`"`) ||
			!strings.Contains(cd, "filename*=UTF-8''A%20Public%20Tale."+format) {
			t.Errorf("%s: Content-Disposition %q", format, cd)
		}
		body := w.Body.String()
		if format == "epub" {
			_, files := unzipEPUB(t, w.Body.Bytes())
			body = strings.Join(mapValues(files), "\n")
		}
		for _, s := range []string{"A Public Tale", "writer", "Once upon", "Test Fandom", "a summary", "General Audiences"} {
			if !strings.Contains(body, s) {
				t.Errorf("%s: missing %q", format, s)
			}
		}
	}

	// The same work by its legacy route and by the permanent one.
	if code := e.get("/api/v1/work/"+work.String()+"/download.html", "").Code; code != 200 {
		t.Errorf("/work/ alias: %d", code)
	}
	// Formats that are not offered say so; none pretends.
	for _, f := range []string{"pdf", "mobi", "azw3", "exe"} {
		if code := e.get(dl(work, f), "").Code; code != 404 {
			t.Errorf("%s: status %d, want 404", f, code)
		}
	}
}

func mapValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func TestEPUBStructure(t *testing.T) {
	e := newFixEnv(t)
	writer, _ := e.user("writer")
	work := e.work(writer, "Structure & <Test>", "published")
	e.exec(`UPDATE works SET max_chapters = 2, warnings = 'No Archive Warnings Apply', category = 'Gen',
		freeform_tags = $2, notes = '<p>Some <b>notes</b><br>here</p>' WHERE id = $1`, work, pq.Array([]string{"Fluff & Angst"}))
	e.exec(`UPDATE chapters SET content = '<p>one<br>two &nbsp; <img src="https://example.org/a.png" alt="pic"></p><hr>' WHERE work_id = $1`, work)
	e.exec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, published_at, summary, notes, end_notes)
		VALUES ($1, $2, 2, 'Second "part"', '<p>more</p>', false, now(), '', '', '<p>end note</p>')`, uuid.New(), work)

	w := e.get(dl(work, "epub"), "")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	zr, files := unzipEPUB(t, w.Body.Bytes())

	first := zr.File[0]
	if first.Name != "mimetype" {
		t.Fatalf("first entry is %q, want mimetype", first.Name)
	}
	if first.Method != zip.Store {
		t.Errorf("mimetype is compressed (method %d)", first.Method)
	}
	if len(first.Extra) != 0 {
		t.Errorf("mimetype has an extra field: %v", first.Extra)
	}
	if files["mimetype"] != "application/epub+zip" {
		t.Errorf("mimetype = %q", files["mimetype"])
	}
	for _, name := range []string{"META-INF/container.xml", "OEBPS/content.opf", "OEBPS/nav.xhtml", "OEBPS/style.css",
		"OEBPS/preface.xhtml", "OEBPS/chapter1.xhtml", "OEBPS/chapter2.xhtml", "OEBPS/afterword.xhtml"} {
		if _, ok := files[name]; !ok {
			t.Errorf("missing %s", name)
		}
	}
	for name, content := range files {
		if strings.HasSuffix(name, ".xhtml") || strings.HasSuffix(name, ".opf") || strings.HasSuffix(name, ".xml") {
			wellFormed(t, name, content)
		}
	}
	if !strings.Contains(files["META-INF/container.xml"], `full-path="OEBPS/content.opf"`) {
		t.Error("container.xml does not point to the package document")
	}

	// The package: every manifest file exists, every spine item is in the manifest.
	opf := files["OEBPS/content.opf"]
	var pkg struct {
		Version  string `xml:"version,attr"`
		Metadata struct {
			Title  string   `xml:"title"`
			Lang   string   `xml:"language"`
			ID     string   `xml:"identifier"`
			Author []string `xml:"creator"`
		} `xml:"metadata"`
		Manifest []struct {
			ID    string `xml:"id,attr"`
			Href  string `xml:"href,attr"`
			Props string `xml:"properties,attr"`
		} `xml:"manifest>item"`
		Spine []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
	}
	if err := xml.Unmarshal([]byte(opf), &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Version != "3.0" || pkg.Metadata.Title != "Structure & <Test>" || pkg.Metadata.Lang != "en" ||
		!strings.HasPrefix(pkg.Metadata.ID, "urn:uuid:") || len(pkg.Metadata.Author) != 1 || pkg.Metadata.Author[0] != "writer" {
		t.Errorf("package metadata: %+v", pkg.Metadata)
	}
	if !strings.Contains(opf, `property="dcterms:modified"`) {
		t.Error("no dcterms:modified")
	}
	ids := map[string]bool{}
	hasNav := false
	for _, it := range pkg.Manifest {
		ids[it.ID] = true
		if _, ok := files["OEBPS/"+it.Href]; !ok {
			t.Errorf("manifest names %s, which is not in the file", it.Href)
		}
		if it.Props == "nav" {
			hasNav = true
		}
	}
	if !hasNav {
		t.Error("no item with properties=nav")
	}
	if len(pkg.Spine) != 4 { // preface, two chapters, afterword
		t.Errorf("spine has %d items, want 4", len(pkg.Spine))
	}
	for _, s := range pkg.Spine {
		if !ids[s.IDRef] {
			t.Errorf("spine item %s is not in the manifest", s.IDRef)
		}
	}
	if !strings.Contains(files["OEBPS/nav.xhtml"], `epub:type="toc"`) || !strings.Contains(files["OEBPS/nav.xhtml"], "chapter2.xhtml") {
		t.Error("nav.xhtml has no table of contents")
	}
	if !strings.Contains(opf, "remote-resources") {
		t.Error("a chapter with a remote image must declare remote-resources")
	}

	// Content: the preface carries AO3's fields, escaped, and the chapter
	// markup is XHTML (void elements closed).
	preface := files["OEBPS/preface.xhtml"]
	for _, s := range []string{"Structure &amp; &lt;Test&gt;", "by writer", "Rating:", "Archive Warning:", "No Archive Warnings Apply",
		"Category:", "Gen", "Fandom:", "Test Fandom", "Additional Tags:", "Fluff &amp; Angst", "Stats:", "Words:", "Chapters: 2/2",
		"Summary", "a summary", "Notes", "<b>notes</b><br/>here"} {
		if !strings.Contains(preface, s) {
			t.Errorf("preface lacks %q", s)
		}
	}
	ch1 := files["OEBPS/chapter1.xhtml"]
	if !strings.Contains(ch1, "<br/>") || !strings.Contains(ch1, "<hr/>") || !strings.Contains(ch1, `alt="pic"`) {
		t.Errorf("chapter 1 markup: %s", ch1)
	}
	ch2 := files["OEBPS/chapter2.xhtml"]
	if !strings.Contains(ch2, "Chapter 2: Second &#34;part&#34;") || !strings.Contains(ch2, "end note") {
		t.Errorf("chapter 2: %s", ch2)
	}
	if !strings.Contains(files["OEBPS/afterword.xhtml"], "Afterword") {
		t.Error("no afterword")
	}
}

func TestDownloadOfAWorkThatHidesItsCreatorNamesNobody(t *testing.T) {
	e := newFixEnv(t)
	writer, writerTok := e.user("secretwriter")
	reader, readerTok := e.user("reader")
	_ = reader
	moon := e.newPseud(writer, "Moonlit Quill")

	for _, hide := range []string{"is_anonymous", "in_anon_collection"} {
		e.exec(`UPDATE works SET is_anonymous = false, in_anon_collection = false`)
		work := e.work(writer, "Hidden "+hide, "published")
		e.exec(`UPDATE creatorships SET pseud_id = $1 WHERE creation_id = $2`, moon, work)
		e.exec(`UPDATE works SET `+hide+` = true WHERE id = $1`, work)
		series := uuid.New()
		e.exec(`INSERT INTO series (id, title, user_id) VALUES ($1, 'Secret Series', $2)`, series, writer)
		e.exec(`INSERT INTO series_works (series_id, work_id, position) VALUES ($1, $2, 1)`, series, work)

		for _, tok := range []string{"", readerTok, writerTok} {
			for _, format := range []string{"epub", "html", "txt"} {
				w := e.get(dl(work, format), tok)
				if w.Code != 200 {
					t.Fatalf("%s %s as %q: status %d", hide, format, tok, w.Code)
				}
				all := w.Body.String()
				if format == "epub" {
					_, files := unzipEPUB(t, w.Body.Bytes())
					all = strings.Join(append(mapValues(files), keys(files)...), "\n")
					// the raw bytes too: names, comments and the zip's own records
					all += w.Body.String()
				}
				for _, secret := range []string{"Moonlit Quill", "secretwriter", writer.String(), moon.String(), "Secret Series"} {
					if strings.Contains(all, secret) {
						t.Errorf("%s %s as %q gives away %q", hide, format, tok, secret)
					}
				}
				if !strings.Contains(all, "Anonymous") {
					t.Errorf("%s %s as %q does not say Anonymous", hide, format, tok)
				}
			}
		}
	}

	// A work in a collection that has not revealed it is not downloadable at all.
	e.exec(`UPDATE works SET is_anonymous = false, in_anon_collection = false`)
	work := e.work(writer, "Unrevealed", "published")
	e.exec(`UPDATE works SET in_unrevealed_collection = true WHERE id = $1`, work)
	for _, tok := range []string{"", readerTok, writerTok} {
		if code := e.get(dl(work, "epub"), tok).Code; code != 404 {
			t.Errorf("unrevealed work as %q: %d, want 404", tok, code)
		}
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestRestrictedWorkDownloadNeedsALogin(t *testing.T) {
	e := newFixEnv(t)
	writer, _ := e.user("writer")
	_, readerTok := e.user("reader")
	work := e.work(writer, "Members Only", "published")
	e.exec(`UPDATE works SET restricted = true WHERE id = $1`, work)
	for _, f := range []string{"epub", "html", "txt"} {
		if code := e.get(dl(work, f), "").Code; code != 404 {
			t.Errorf("%s logged out: %d, want 404 (the same as reading it)", f, code)
		}
		if code := e.get(dl(work, f), readerTok).Code; code != 200 {
			t.Errorf("%s logged in: %d, want 200", f, code)
		}
	}
}

func TestHiddenDraftAndSpamWorksCannotBeDownloaded(t *testing.T) {
	e := newFixEnv(t)
	writer, writerTok := e.user("writer")
	_, adminTok := e.user("admin", "admin")
	_, readerTok := e.user("reader")

	hidden := e.work(writer, "Hidden", "hidden")
	draft := e.work(writer, "Draft", "draft")
	spam := e.work(writer, "Spam", "published")
	e.exec(`UPDATE works SET spam = true WHERE id = $1`, spam)
	for name, id := range map[string]uuid.UUID{"hidden": hidden, "draft": draft, "spam": spam} {
		// Reading is allowed to creators and staff for hidden works; a
		// download is not allowed to anyone. (Two formats each, to stay under
		// the per-address limit.)
		for _, tok := range []string{"", readerTok, writerTok, adminTok} {
			for _, f := range []string{"epub", "html"} {
				if code := e.get(dl(id, f), tok).Code; code != 404 {
					t.Errorf("%s work, %s, token %q: %d, want 404", name, f, tok, code)
				}
			}
		}
	}
}

func TestDraftChaptersAreOnlyInTheCreatorsCopy(t *testing.T) {
	e := newFixEnv(t)
	writer, writerTok := e.user("writer")
	_, readerTok := e.user("reader")
	work := e.work(writer, "Half Done", "published")
	e.exec(`UPDATE chapters SET content = '<p>posted chapter</p>' WHERE work_id = $1`, work)
	e.exec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, summary, notes, end_notes)
		VALUES ($1, $2, 2, 'later', '<p>unfinished secret</p>', true, '', '', '')`, uuid.New(), work)
	e.exec(`INSERT INTO chapters (id, work_id, chapter_number, title, content, is_draft, published_at, hidden_by_admin, summary, notes, end_notes)
		VALUES ($1, $2, 3, 'removed', '<p>admin hid this</p>', false, now(), true, '', '', '')`, uuid.New(), work)

	for _, tok := range []string{"", readerTok} {
		body := e.get(dl(work, "txt"), tok).Body.String()
		if !strings.Contains(body, "posted chapter") || strings.Contains(body, "unfinished secret") || strings.Contains(body, "admin hid this") {
			t.Errorf("reader %q copy:\n%s", tok, body)
		}
	}
	body := e.get(dl(work, "txt"), writerTok).Body.String()
	if !strings.Contains(body, "posted chapter") || !strings.Contains(body, "unfinished secret") {
		t.Errorf("creator's copy lacks their draft chapter:\n%s", body)
	}
	if strings.Contains(body, "admin hid this") {
		t.Error("a chapter an admin hid is in a download")
	}
}

func TestDownloadEscapesAndSanitizes(t *testing.T) {
	e := newFixEnv(t)
	writer, _ := e.user("writer")
	work := e.work(writer, `<script>alert("title")</script>`, "published")
	e.exec(`UPDATE works SET summary = $2, notes = $3, freeform_tags = $4, fandoms = $5 WHERE id = $1`, work,
		`<p onclick="alert(1)">sum</p><script>alert("summary")</script><img src=x onerror=alert(1)>`,
		`<a href="javascript:alert(1)">bad link</a><iframe src="https://evil.example"></iframe>`,
		pq.Array([]string{`<img src=x onerror=alert("tag")>`}), pq.Array([]string{`"><svg onload=alert(1)>`}))
	e.exec(`UPDATE chapters SET title = '<b>t</b>', content = $2, end_notes = $3 WHERE work_id = $1`, work,
		`<p>fine</p><script>alert("chapter")</script><p style="x" onmouseover="alert(2)">hover</p><a href="data:text/html,x">d</a><object data="x"></object>`,
		`<form action="/steal"><input name=a></form>end`)

	for _, format := range []string{"epub", "html"} {
		w := e.get(dl(work, format), "")
		if w.Code != 200 {
			t.Fatalf("%s: %d", format, w.Code)
		}
		all := w.Body.String()
		if format == "epub" {
			_, files := unzipEPUB(t, w.Body.Bytes())
			all = strings.Join(mapValues(files), "\n")
			for name, c := range files {
				if strings.HasSuffix(name, "xhtml") || strings.HasSuffix(name, "opf") {
					wellFormed(t, name, c)
				}
			}
		}
		for _, bad := range []string{"<script", "onclick", "onmouseover", "javascript:", "data:text", "<iframe", "<object",
			"<form", "<svg", `<img src=x`, `<b>t</b>`, "style=\"x\"", `<a href="data`} {
			if strings.Contains(all, bad) {
				t.Errorf("%s contains %q", format, bad)
			}
		}
		// Escaped text may still say "onerror"; a live tag must not carry it.
		if regexp.MustCompile(`<[^>]*\son(error|load)\s*=`).MatchString(all) {
			t.Errorf("%s has a tag with an event handler", format)
		}
		for _, good := range []string{"&lt;script&gt;alert(&#34;title&#34;)&lt;/script&gt;", "&lt;img src=x onerror=alert(&#34;tag&#34;)&gt;",
			"&lt;b&gt;t&lt;/b&gt;", "<p>fine</p>", "hover", "sum", "end"} {
			if !strings.Contains(all, good) {
				t.Errorf("%s lacks %q", format, good)
			}
		}
	}
	// Plain text keeps the words and none of the markup.
	txt := e.get(dl(work, "txt"), "").Body.String()
	if strings.Contains(txt, "<p") || strings.Contains(txt, "alert(\"chapter\")") || !strings.Contains(txt, "fine") {
		t.Errorf("txt: %s", txt)
	}
}

func TestDownloadRateLimitAndLegacyRedirect(t *testing.T) {
	e := newFixEnv(t)
	writer, _ := e.user("writer")
	work := e.work(writer, "Busy", "published")
	e.exec(`UPDATE works SET legacy_id = 4242 WHERE id = $1`, work)

	w := e.get("/api/v1/works/4242/download.epub", "")
	if w.Code != http.StatusMovedPermanently || w.Header().Get("Location") != "/api/v1/work/"+work.String()+"/download.epub" {
		t.Errorf("legacy id: %d %q", w.Code, w.Header().Get("Location"))
	}

	limited := 0
	for i := 0; i < 40; i++ {
		if e.get(dl(work, "txt"), "").Code == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited == 0 || limited > 12 {
		t.Errorf("%d of 40 quick downloads were refused; want a few, after the first 30", limited)
	}
}
