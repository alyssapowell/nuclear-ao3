package export

import (
	"fmt"
	stdhtml "html"
	"regexp"
	"strings"

	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	cleanhtml "nuclear-ao3/internal/html"
)

// esc escapes text for XML and HTML, in text and in attribute values, and
// drops characters XML cannot hold (control characters, U+FFFE and U+FFFF).
func esc(s string) string {
	return stdhtml.EscapeString(strings.Map(validXMLRune, s))
}

func validXMLRune(r rune) rune {
	switch {
	case r == '\t' || r == '\n' || r == '\r':
		return r
	case r < 0x20, r == 0xFFFE, r == 0xFFFF, r >= 0xD800 && r <= 0xDFFF:
		return -1
	}
	return r
}

var (
	elementName   = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	attributeName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

// voidElements have no end tag.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true,
	"img": true, "input": true, "link": true, "meta": true, "source": true,
	"track": true, "wbr": true,
}

// parse cleans s again with the site's HTML rules and parses the result.
func parse(s string) []*nethtml.Node {
	s = cleanhtml.Clean(s)
	if strings.TrimSpace(s) == "" {
		return nil
	}
	ctx := &nethtml.Node{Type: nethtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := nethtml.ParseFragment(strings.NewReader(s), ctx)
	if err != nil {
		// Fall back to the text alone, escaped.
		return []*nethtml.Node{{Type: nethtml.TextNode, Data: s}}
	}
	return nodes
}

// fragment returns s cleaned and written as XHTML that is also valid HTML:
// every element is closed, empty elements are written with an end tag unless
// they are void, and all text and attribute values are escaped.
func fragment(s string) string {
	var b strings.Builder
	for _, n := range parse(s) {
		writeXHTML(&b, n)
	}
	return b.String()
}

func writeXHTML(b *strings.Builder, n *nethtml.Node) {
	switch n.Type {
	case nethtml.TextNode:
		b.WriteString(esc(n.Data))
	case nethtml.ElementNode:
		if !elementName.MatchString(n.Data) {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				writeXHTML(b, c)
			}
			return
		}
		b.WriteByte('<')
		b.WriteString(n.Data)
		for _, a := range n.Attr {
			if a.Namespace != "" || !attributeName.MatchString(a.Key) {
				continue
			}
			fmt.Fprintf(b, ` %s="%s"`, a.Key, esc(a.Val))
		}
		if voidElements[n.Data] {
			b.WriteString("/>")
			return
		}
		b.WriteByte('>')
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			writeXHTML(b, c)
		}
		b.WriteString("</" + n.Data + ">")
	}
	// Comments, doctypes and the like are dropped.
}

// hasImage says whether s holds an image (an EPUB must declare remote
// resources).
func hasImage(s string) bool {
	return strings.Contains(fragment(s), "<img")
}

var blockElements = map[string]bool{
	"p": true, "div": true, "blockquote": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "ul": true, "ol": true, "li": true,
	"pre": true, "table": true, "tr": true, "hr": true, "dl": true, "dt": true, "dd": true,
}

var (
	spaceRun      = regexp.MustCompile(`[ \t\r\n\f]+`)
	manyNL        = regexp.MustCompile(`\n{3,}`)
	trailingSpace = regexp.MustCompile(`[ \t]+\n`)
)

// plain returns s as plain text.
func plain(s string) string {
	var b strings.Builder
	for _, n := range parse(s) {
		writeText(&b, n, false)
	}
	out := strings.ReplaceAll(b.String(), " ", " ")
	out = trailingSpace.ReplaceAllString(out, "\n")
	out = manyNL.ReplaceAllString(out, "\n\n")
	return strings.TrimSpace(out)
}

func writeText(b *strings.Builder, n *nethtml.Node, pre bool) {
	switch n.Type {
	case nethtml.TextNode:
		t := strings.Map(validXMLRune, n.Data)
		if !pre {
			t = spaceRun.ReplaceAllString(t, " ")
		}
		b.WriteString(t)
	case nethtml.ElementNode:
		name := n.Data
		block := blockElements[name]
		if block {
			b.WriteString("\n\n")
		}
		switch name {
		case "br":
			b.WriteString("\n")
		case "hr":
			b.WriteString("* * *")
		case "li":
			b.WriteString("* ")
		case "img":
			for _, a := range n.Attr {
				if a.Key == "alt" && a.Val != "" {
					b.WriteString("[" + strings.Map(validXMLRune, a.Val) + "]")
				}
			}
		}
		start := b.Len()
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			writeText(b, c, pre || name == "pre")
		}
		if name == "a" {
			text := b.String()[start:]
			for _, a := range n.Attr {
				if a.Key == "href" && a.Val != "" && strings.TrimSpace(text) != a.Val {
					b.WriteString(" (" + a.Val + ")")
				}
			}
		}
		if block {
			b.WriteString("\n\n")
		}
	}
}
