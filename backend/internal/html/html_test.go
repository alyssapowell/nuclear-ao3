package html

import "testing"

func TestCleanRemovesWhatAO3DoesNotAllow(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"onerror", `<img src=x onerror=alert(1)>`, ``},
		{"onerror on a good image", `<img src="https://example.org/a.png" onerror="alert(1)">`, `<img src="https://example.org/a.png">`},
		{"onclick", `<p onclick="alert(1)">hi</p>`, `<p>hi</p>`},
		{"javascript href", `<a href="javascript:alert(1)">x</a>`, `x`},
		{"javascript href mixed case", `<a href="JaVaScRiPt:alert(1)">x</a>`, `x`},
		{"javascript href with a tab entity", `<a href="jav&#x09;ascript:alert(1)">x</a>`, `x`},
		{"javascript href with a newline entity", `<a href="jav&#x0A;ascript:alert(1)">x</a>`, `x`},
		{"javascript href with leading space", `<a href=" javascript:alert(1)">x</a>`, `x`},
		{"javascript href fully entity encoded", `<a href="&#106;&#97;&#118;&#97;&#115;&#99;&#114;&#105;&#112;&#116;&#58;alert(1)">x</a>`, `x`},
		{"vbscript href", `<a href="vbscript:msgbox(1)">x</a>`, `x`},
		{"data href", `<a href="data:text/html,<script>alert(1)</script>">x</a>`, `x`},
		{"data src", `<img src="data:image/svg+xml;base64,PHN2Zz4=">`, ``},
		{"javascript src", `<img src="javascript:alert(1)">`, ``},
		{"script", `a<script>alert(1)</script>b`, `ab`},
		{"svg onload", `<svg onload=alert(1)><circle/></svg>ok`, `ok`},
		{"style element", `<style>body{display:none}</style>ok`, `ok`},
		{"style attribute", `<p style="position:fixed">x</p>`, `<p>x</p>`},
		{"iframe", `<iframe src="https://example.org"></iframe>ok`, `ok`},
		{"object", `<object data="x.swf"></object>ok`, `ok`},
		{"embed", `<embed src="x.swf">ok`, `ok`},
		{"form", `<form action="https://evil.example"><input name=a></form>ok`, `ok`},
		{"link target is dropped", `<a href="https://example.org" target="_blank">x</a>`, `<a href="https://example.org" rel="nofollow noopener">x</a>`},
		{"link rel from the user is replaced", `<a href="https://example.org" rel="opener">x</a>`, `<a href="https://example.org" rel="nofollow noopener">x</a>`},
		{"class and id are dropped", `<p class="a" id="b">x</p>`, `<p>x</p>`},
		{"ftp link", `<a href="ftp://example.org/f">x</a>`, `x`},
		{"blockquote cite javascript", `<blockquote cite="javascript:alert(1)">x</blockquote>`, `<blockquote>x</blockquote>`},
		{"unclosed tag cannot break out", `<img src="https://example.org/a.png" alt="x`, ``},
		{"comment", `a<!-- <script>alert(1)</script> -->b`, `ab`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Clean(c.in); got != c.want {
				t.Errorf("Clean(%q)\n got %q\nwant %q", c.in, got, c.want)
			}
		})
	}
}

func TestCleanKeepsWhatAO3Allows(t *testing.T) {
	same := []string{
		`<p>Hello</p>`,
		`<em>a</em> <strong>b</strong> <i>c</i> <b>d</b> <u>e</u> <s>f</s>`,
		`<a href="https://example.org/page" rel="nofollow noopener">link</a>`,
		`<a href="mailto:someone@example.org" rel="nofollow noopener">mail</a>`,
		`<a href="http://example.org" rel="nofollow noopener">plain http</a>`,
		`<blockquote><p>quoted</p></blockquote>`,
		`<blockquote cite="https://example.org/q">quoted</blockquote>`,
		`<img src="https://example.org/a.png" alt="a picture" width="10" height="20">`,
		`<img src="http://example.org/a.png">`,
		`<table border="1"><thead><tr><th scope="col">h</th></tr></thead><tbody><tr><td colspan="2">c</td></tr></tbody></table>`,
		`<ruby>漢<rp>(</rp><rt>kan</rt><rp>)</rp></ruby>`,
		`<ul type="disc"><li>a</li></ul><ol start="3"><li>b</li></ol>`,
		`<p align="center" dir="rtl" title="t">x</p>`,
		`<details open=""><summary>s</summary>text</details>`,
		`<h1>a</h1><hr width="50%"><pre>code</pre><br/>`,
		`<p>1 &lt; 2 &amp; 3 &gt; 2</p>`,
		`<dl><dt>t</dt><dd>d</dd></dl>`,
	}
	for _, s := range same {
		want := s
		if got := Clean(s); got != want {
			// bluemonday writes void elements as <br/> and so on; compare
			// against a second pass to make sure it is only that.
			if Clean(got) != got {
				t.Errorf("Clean is not stable for %q: %q then %q", s, got, Clean(got))
			}
			t.Errorf("Clean(%q) = %q", s, got)
		}
	}
}

func TestCleanIsSafeToRunTwice(t *testing.T) {
	inputs := []string{
		`<img src=x onerror=alert(1)>text`,
		`<a href="https://example.org" target=_blank>x</a>`,
		`a &nbsp; b &amp; c <3 d`,
		`<p>unclosed <b>bold`,
		`plain text with > and < signs`,
		`<table><tr><td>x`,
	}
	for _, in := range inputs {
		once := Clean(in)
		if twice := Clean(once); twice != once {
			t.Errorf("not stable for %q: %q then %q", in, once, twice)
		}
	}
}

func TestCleanLeavesEmptyAlone(t *testing.T) {
	if Clean("") != "" {
		t.Error("empty string changed")
	}
}
