package export

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestContentDisposition(t *testing.T) {
	long := strings.Repeat("a", 300)
	for _, tc := range []struct {
		title, ext, want string
	}{
		{"Plain Title", "epub", `attachment; filename="Plain Title.epub"; filename*=UTF-8''Plain%20Title.epub`},
		{"Tést: Ünï \"quoted\" / slash", "html", `attachment; filename="T_st _n_ quoted  slash.html"; filename*=UTF-8''T%C3%A9st%20%C3%9Cn%C3%AF%20quoted%20%20slash.html`},
		{"日本語", "txt", `attachment; filename="work.txt"; filename*=UTF-8''%E6%97%A5%E6%9C%AC%E8%AA%9E.txt`},
		{"", "txt", `attachment; filename="work.txt"; filename*=UTF-8''work.txt`},
		{"evil\r\nSet-Cookie: x=1", "txt", `attachment; filename="evilSet-Cookie x_1.txt"`},
	} {
		got := ContentDisposition(tc.title, tc.ext)
		if !strings.HasPrefix(got, strings.Split(tc.want, "; filename*")[0]) {
			t.Errorf("%q: got %s\n want prefix of %s", tc.title, got, tc.want)
		}
		if strings.ContainsAny(got, "\r\n") {
			t.Errorf("%q: header has a line break", tc.title)
		}
	}
	if got := ContentDisposition(long, "epub"); len(got) > 600 || !strings.Contains(got, strings.Repeat("a", 100)+".epub") || strings.Contains(got, strings.Repeat("a", 101)) {
		t.Errorf("long title not cut to 100 characters: %s", got)
	}
	if got := ContentDisposition("Tést", "epub"); !strings.Contains(got, "filename*=UTF-8''T%C3%A9st.epub") {
		t.Errorf("filename*: %s", got)
	}
}

func TestFragmentIsWellFormedXHTML(t *testing.T) {
	in := `<p>a &amp; b &nbsp; <br>c</p><ul><li>x<li>y</ul><p>unclosed <b>bold<hr><img src="https://e.org/i.png" alt='a"b'>`
	out := fragment(in)
	d := xml.NewDecoder(strings.NewReader("<root>" + out + "</root>"))
	for {
		if _, err := d.Token(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("%v in %s", err, out)
		}
	}
	for _, want := range []string{"<br/>", "<hr/>", `alt="a&#34;b"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
}

func TestPlainText(t *testing.T) {
	got := plain(`<p>One</p><p>Two<br>three</p><ul><li>a</li><li>b</li></ul><a href="https://e.org/x">link</a>`)
	for _, want := range []string{"One\n\nTwo\nthree", "* a\n\n* b", "link (https://e.org/x)"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestAnonymousWhenNoAuthors(t *testing.T) {
	w := &Work{ID: "11111111-1111-1111-1111-111111111111", Title: "T", Chapters: []Chapter{{Number: 1, ContentHTML: "<p>x</p>"}}}
	var b bytes.Buffer
	if err := Text(&b, w); err != nil || !strings.Contains(b.String(), "by Anonymous") {
		t.Errorf("%v %s", err, b.String())
	}
	b.Reset()
	if err := EPUB(&b, w); err != nil || b.Len() == 0 {
		t.Fatal(err)
	}
}
