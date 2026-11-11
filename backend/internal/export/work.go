package export

import "time"

// Work is everything a download shows. Text fields are plain text unless the
// name says HTML.
type Work struct {
	ID    string
	Title string
	// Authors are the pseud names to credit. Empty means "Anonymous" (a work
	// that hides its creators, or has none): the caller passes nobody.
	Authors  []string
	Language string
	URL      string // the work's page, "" if unknown

	Rating        string
	Warnings      []string
	Categories    []string
	Fandoms       []string
	Relationships []string
	Characters    []string
	Tags          []string // additional tags

	Series []SeriesPart

	SummaryHTML string
	NotesHTML   string

	Published time.Time
	Updated   time.Time
	Words     int
	// ChaptersTotal is the planned number of chapters, 0 when not known.
	ChaptersTotal int

	Chapters []Chapter
}

// SeriesPart says "Part N of Title".
type SeriesPart struct {
	Title    string
	Position int
}

// Chapter is one chapter. Its HTML fields are cleaned again on output.
type Chapter struct {
	Number       int
	Title        string
	SummaryHTML  string
	NotesHTML    string
	ContentHTML  string
	EndNotesHTML string
}

// Format is a download format.
type Format struct {
	Ext         string
	ContentType string
}

// Formats are the formats this package writes, in menu order.
var Formats = []Format{
	{"epub", "application/epub+zip"},
	{"html", "text/html; charset=utf-8"},
	{"txt", "text/plain; charset=utf-8"},
}

// Lookup returns the format for an extension.
func Lookup(ext string) (Format, bool) {
	for _, f := range Formats {
		if f.Ext == ext {
			return f, true
		}
	}
	return Format{}, false
}

// authors returns the names to credit, or "Anonymous".
func (w *Work) authors() []string {
	if len(w.Authors) == 0 {
		return []string{"Anonymous"}
	}
	return w.Authors
}

// ratingLabel gives the AO3 wording for a stored rating.
func ratingLabel(r string) string {
	switch r {
	case "general":
		return "General Audiences"
	case "teen":
		return "Teen And Up Audiences"
	case "mature":
		return "Mature"
	case "explicit":
		return "Explicit"
	case "not_rated", "":
		return "Not Rated"
	}
	return r
}

func date(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}
