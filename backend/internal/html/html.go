// Package html cleans HTML that users write (summaries, notes, chapter text,
// comments, profiles) before it is stored.
//
// The rules are AO3's own: the same tags, the same attributes on each tag and
// the same link protocols as the Sanitize configuration in AO3's
// config/initializers/gem-plugin_config/sanitizer_config.rb (the ARCHIVE
// config). Anything else is removed, in particular event handlers such as
// onerror, style attributes, javascript: and data: URLs, and the elements
// script, style, iframe, object, embed and form.
//
// Two deliberate differences from AO3: links may use only http, https and
// mailto (AO3 also allows ftp), and media embeds and CSS classes, which AO3
// allows in some fields, are not supported yet.
//
// Titles, tag names, pseud names and usernames are plain text, not HTML, and
// must not be passed through here: the web page escapes them when it shows
// them.
package html

import (
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var policy = newPolicy()

// Clean returns s with everything AO3 does not allow removed. It is safe to
// call on its own output: Clean(Clean(s)) == Clean(s).
func Clean(s string) string {
	if s == "" {
		return ""
	}
	out := policy.Sanitize(s)
	// bluemonday can only add rel="nofollow" by itself. The user's own rel is
	// never kept, and any quote inside text or another attribute is escaped,
	// so this exact string can only be the attribute we added.
	return strings.ReplaceAll(out, `rel="nofollow"`, `rel="nofollow noopener"`)
}

var (
	letters    = regexp.MustCompile(`^[a-zA-Z]+$`)
	anchorName = regexp.MustCompile(`^[a-zA-Z0-9_:.-]+$`)
	listType   = regexp.MustCompile(`^(?i:[1aAiI]|disc|circle|square)$`)
	boolOpen   = regexp.MustCompile(`^(?i:open)?$`)
	imageURL   = regexp.MustCompile(`^(?i:https?://)`)
)

// Tags and attributes from AO3's sanitizer_config.rb, lines 6 to 42.
func newPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	// Elements (sanitizer_config.rb:6-11).
	p.AllowElements(
		"a", "abbr", "acronym", "address", "b", "big", "blockquote", "br", "caption", "center", "cite", "code", "col",
		"colgroup", "details", "figcaption", "figure", "dd", "del", "dfn", "div", "dl", "dt", "em", "h1", "h2", "h3", "h4", "h5", "h6", "hr",
		"i", "img", "ins", "kbd", "li", "ol", "p", "pre", "q", "rp", "rt", "ruby", "s", "samp", "small", "span", "strike", "strong",
		"sub", "summary", "sup", "table", "tbody", "td", "tfoot", "th", "thead", "tr", "tt", "u", "ul", "var",
	)

	// Attributes on every element (sanitizer_config.rb:13).
	p.AllowAttrs("align").Matching(letters).Globally()
	p.AllowAttrs("dir").Matching(letters).Globally()
	p.AllowAttrs("title").Globally()

	// Attributes on particular elements (sanitizer_config.rb:14-26).
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("name").Matching(anchorName).OnElements("a")
	p.AllowAttrs("cite").OnElements("blockquote", "q")
	p.AllowAttrs("span").Matching(bluemonday.Integer).OnElements("col", "colgroup")
	p.AllowAttrs("width").Matching(bluemonday.NumberOrPercent).OnElements("col", "colgroup", "hr", "img", "table", "td", "th")
	p.AllowAttrs("open").Matching(boolOpen).OnElements("details")
	p.AllowAttrs("alt").OnElements("img")
	p.AllowAttrs("border").Matching(bluemonday.Integer).OnElements("img", "table")
	p.AllowAttrs("height").Matching(bluemonday.NumberOrPercent).OnElements("img", "td", "th")
	p.AllowAttrs("src").Matching(imageURL).OnElements("img")
	p.AllowAttrs("start").Matching(bluemonday.Integer).OnElements("ol")
	p.AllowAttrs("type").Matching(listType).OnElements("ol", "ul")
	p.AllowAttrs("summary").OnElements("table")
	p.AllowAttrs("abbr", "axis").OnElements("td", "th")
	p.AllowAttrs("colspan", "rowspan").Matching(bluemonday.Integer).OnElements("td", "th")
	p.AllowAttrs("scope").Matching(letters).OnElements("th")

	// Link protocols (sanitizer_config.rb:33-40; rel at 29-31). Relative URLs are allowed
	// for links and citations as on AO3; images need a full http(s) address.
	p.AllowURLSchemes("http", "https", "mailto")
	p.AllowRelativeURLs(true)
	p.RequireNoFollowOnLinks(true)

	// Elements whose text is dropped along with the tag
	// (sanitizer_config.rb:43).
	p.SkipElementsContent("iframe", "math", "noembed", "noframes", "noscript", "plaintext", "script", "style", "svg", "xmp",
		"object", "template", "textarea", "title")

	return p
}

// CleanPtr is Clean for optional fields: nil stays nil.
func CleanPtr(s *string) *string {
	if s == nil {
		return nil
	}
	c := Clean(*s)
	return &c
}
