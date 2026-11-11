package export

import (
	"fmt"
	"io"
	"strings"
)

// The page pieces below are shared by the HTML file and the EPUB. Each
// returns an XHTML fragment with everything escaped.

const stylesheet = `body { font-family: Georgia, serif; line-height: 1.5; margin: 1em; }
h1, h2, h3 { font-family: sans-serif; }
.meta dt { font-weight: bold; float: left; clear: left; margin-right: 0.5em; }
.meta dd { margin: 0 0 0.25em 0; }
.byline { font-style: italic; }
blockquote.userstuff { margin: 0.5em 1.5em; }
.chapter-notes { font-size: 0.9em; }
`

func (w *Work) sourceLine() string {
	if w.URL == "" {
		return ""
	}
	return fmt.Sprintf(`<p class="source">Posted originally on the Archive of Our Own at <a href="%s">%s</a>.</p>`, esc(w.URL), esc(w.URL))
}

func metaRow(b *strings.Builder, label string, values []string) {
	if len(values) == 0 {
		return
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = esc(v)
	}
	fmt.Fprintf(b, "<dt>%s</dt><dd>%s</dd>\n", esc(label), strings.Join(parts, ", "))
}

func (w *Work) chapterStats() string {
	total := "?"
	if w.ChaptersTotal > 0 {
		total = fmt.Sprint(w.ChaptersTotal)
	}
	return fmt.Sprintf("%d/%s", len(w.Chapters), total)
}

// preface is the title page: AO3's download preface.
func (w *Work) preface() string {
	var b strings.Builder
	b.WriteString(`<div class="meta">` + "\n")
	b.WriteString(w.sourceLine() + "\n")
	fmt.Fprintf(&b, `<h1 class="title">%s</h1>`+"\n", esc(w.Title))
	names := make([]string, len(w.authors()))
	for i, a := range w.authors() {
		names[i] = esc(a)
	}
	fmt.Fprintf(&b, `<div class="byline">by %s</div>`+"\n", strings.Join(names, ", "))
	b.WriteString("<dl>\n")
	metaRow(&b, "Rating:", []string{ratingLabel(w.Rating)})
	metaRow(&b, "Archive Warning:", w.Warnings)
	metaRow(&b, "Category:", w.Categories)
	metaRow(&b, "Fandom:", w.Fandoms)
	metaRow(&b, "Relationship:", w.Relationships)
	metaRow(&b, "Character:", w.Characters)
	metaRow(&b, "Additional Tags:", w.Tags)
	if w.Language != "" {
		metaRow(&b, "Language:", []string{w.Language})
	}
	for _, s := range w.Series {
		metaRow(&b, "Series:", []string{fmt.Sprintf("Part %d of %s", s.Position, s.Title)})
	}
	stats := []string{}
	if d := date(w.Published); d != "" {
		stats = append(stats, "Published: "+d)
	}
	if d := date(w.Updated); d != "" && d != date(w.Published) {
		stats = append(stats, "Updated: "+d)
	}
	stats = append(stats, fmt.Sprintf("Words: %d", w.Words), "Chapters: "+w.chapterStats())
	metaRow(&b, "Stats:", stats)
	b.WriteString("</dl>\n")
	if s := fragment(w.SummaryHTML); s != "" {
		fmt.Fprintf(&b, `<h2>Summary</h2><blockquote class="userstuff">%s</blockquote>`+"\n", s)
	}
	if s := fragment(w.NotesHTML); s != "" {
		fmt.Fprintf(&b, `<h2>Notes</h2><blockquote class="userstuff">%s</blockquote>`+"\n", s)
	}
	b.WriteString("</div>\n")
	return b.String()
}

// chapterHeading is "Chapter 2: Title", or "" for a work that is one
// untitled chapter.
func (w *Work) chapterHeading(c Chapter) string {
	if len(w.Chapters) == 1 && c.Title == "" {
		return ""
	}
	h := fmt.Sprintf("Chapter %d", c.Number)
	if c.Title != "" {
		h += ": " + c.Title
	}
	return h
}

func (w *Work) chapter(c Chapter) string {
	var b strings.Builder
	b.WriteString(`<div class="chapter">` + "\n")
	if h := w.chapterHeading(c); h != "" {
		fmt.Fprintf(&b, "<h2>%s</h2>\n", esc(h))
	}
	if s := fragment(c.SummaryHTML); s != "" {
		fmt.Fprintf(&b, `<div class="chapter-notes"><h3>Summary</h3><blockquote class="userstuff">%s</blockquote></div>`+"\n", s)
	}
	if s := fragment(c.NotesHTML); s != "" {
		fmt.Fprintf(&b, `<div class="chapter-notes"><h3>Notes</h3><blockquote class="userstuff">%s</blockquote></div>`+"\n", s)
	}
	fmt.Fprintf(&b, `<div class="userstuff">%s</div>`+"\n", fragment(c.ContentHTML))
	if s := fragment(c.EndNotesHTML); s != "" {
		fmt.Fprintf(&b, `<div class="chapter-notes"><h3>End Notes</h3><blockquote class="userstuff">%s</blockquote></div>`+"\n", s)
	}
	b.WriteString("</div>\n")
	return b.String()
}

// afterword is AO3's closing page.
func (w *Work) afterword() string {
	var b strings.Builder
	b.WriteString(`<div class="afterword">` + "\n<h2>Afterword</h2>\n")
	fmt.Fprintf(&b, "<p>Please drop by the Archive and comment to let the creator know if you enjoyed their work!</p>\n")
	if w.URL != "" {
		fmt.Fprintf(&b, `<p><a href="%s">%s</a></p>`+"\n", esc(w.URL), esc(w.URL))
	}
	b.WriteString("</div>\n")
	return b.String()
}

func (w *Work) lang() string {
	if validLang.MatchString(w.Language) {
		return w.Language
	}
	return "en"
}

// HTML writes a single self-contained HTML page.
func HTML(out io.Writer, w *Work) error {
	var b strings.Builder
	fmt.Fprintf(&b, "<!DOCTYPE html>\n<html lang=\"%s\">\n<head>\n<meta charset=\"utf-8\"/>\n<title>%s</title>\n", esc(w.lang()), esc(w.Title))
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1"/>` + "\n")
	b.WriteString("<style>" + stylesheet + "</style>\n</head>\n<body>\n")
	b.WriteString(w.preface())
	for _, c := range w.Chapters {
		b.WriteString(w.chapter(c))
	}
	b.WriteString(w.afterword())
	b.WriteString("</body>\n</html>\n")
	_, err := io.WriteString(out, b.String())
	return err
}

// Text writes the work as plain text.
func Text(out io.Writer, w *Work) error {
	var b strings.Builder
	b.WriteString(plainLine(w.Title) + "\n")
	b.WriteString("by " + plainLine(strings.Join(w.authors(), ", ")) + "\n\n")
	if w.URL != "" {
		b.WriteString("Posted originally on the Archive of Our Own at " + plainLine(w.URL) + "\n\n")
	}
	row := func(label string, values []string) {
		if len(values) > 0 {
			b.WriteString(label + " " + plainLine(strings.Join(values, ", ")) + "\n")
		}
	}
	row("Rating:", []string{ratingLabel(w.Rating)})
	row("Archive Warning:", w.Warnings)
	row("Category:", w.Categories)
	row("Fandom:", w.Fandoms)
	row("Relationship:", w.Relationships)
	row("Character:", w.Characters)
	row("Additional Tags:", w.Tags)
	if w.Language != "" {
		row("Language:", []string{w.Language})
	}
	for _, s := range w.Series {
		row("Series:", []string{fmt.Sprintf("Part %d of %s", s.Position, s.Title)})
	}
	stats := []string{}
	if d := date(w.Published); d != "" {
		stats = append(stats, "Published: "+d)
	}
	if d := date(w.Updated); d != "" && d != date(w.Published) {
		stats = append(stats, "Updated: "+d)
	}
	stats = append(stats, fmt.Sprintf("Words: %d", w.Words), "Chapters: "+w.chapterStats())
	row("Stats:", stats)
	if s := plain(w.SummaryHTML); s != "" {
		b.WriteString("\nSummary\n\n" + s + "\n")
	}
	if s := plain(w.NotesHTML); s != "" {
		b.WriteString("\nNotes\n\n" + s + "\n")
	}
	for _, c := range w.Chapters {
		b.WriteString("\n\n========\n\n")
		if h := w.chapterHeading(c); h != "" {
			b.WriteString(plainLine(h) + "\n\n")
		}
		if s := plain(c.SummaryHTML); s != "" {
			b.WriteString("Summary: " + s + "\n\n")
		}
		if s := plain(c.NotesHTML); s != "" {
			b.WriteString("Notes: " + s + "\n\n")
		}
		b.WriteString(plain(c.ContentHTML) + "\n")
		if s := plain(c.EndNotesHTML); s != "" {
			b.WriteString("\nEnd Notes: " + s + "\n")
		}
	}
	b.WriteString("\n\n========\n\nAfterword\n\nPlease drop by the Archive and comment to let the creator know if you enjoyed their work!\n")
	if w.URL != "" {
		b.WriteString(plainLine(w.URL) + "\n")
	}
	_, err := io.WriteString(out, b.String())
	return err
}

// plainLine makes a one-line plain string: no control characters or line
// breaks.
func plainLine(s string) string {
	s = strings.Map(validXMLRune, s)
	return strings.Join(strings.Fields(s), " ")
}
