package export

import (
	"strings"
	"unicode"
)

const maxFilenameRunes = 100

// ContentDisposition returns an attachment header value for a download named
// after title: an ASCII filename for old clients and the real name as
// filename* (RFC 5987 / 6266).
func ContentDisposition(title, ext string) string {
	clean := strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '/' || r == '\\' || r == '"' || r == ':' || r == '*' || r == '?' || r == '<' || r == '>' || r == '|' {
			return -1
		}
		return r
	}, title))
	runes := []rune(clean)
	if len(runes) > maxFilenameRunes {
		clean = strings.TrimSpace(string(runes[:maxFilenameRunes]))
	}
	if clean == "" {
		clean = "work"
	}

	var ascii strings.Builder
	for _, r := range clean {
		switch {
		case r < 0x80 && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '.' || r == '-' || r == '_' || r == '\'' || r == '(' || r == ')' || r == ','):
			ascii.WriteRune(r)
		default:
			ascii.WriteByte('_')
		}
	}
	fallback := strings.Trim(ascii.String(), " .")
	if strings.Trim(fallback, "_ ") == "" {
		fallback = "work"
	}
	return `attachment; filename="` + fallback + "." + ext + `"; filename*=UTF-8''` + pctEncode(clean+"."+ext)
}

// pctEncode percent-encodes everything outside RFC 5987's attr-char.
func pctEncode(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			strings.IndexByte("!#$&+-.^_`|~", c) >= 0:
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
		}
	}
	return b.String()
}
