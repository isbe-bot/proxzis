package parley

import (
	"html"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Banter is display text, never an instruction or trusted markup.
type Banter string

const MaxBanterLen = 140

// MaxBanterInputBytes bounds work before decoding entities or examining markup.
// Oversized input is muted rather than partially interpreting hostile markup.
const MaxBanterInputBytes = 16 * 1024

// SanitizeBanter produces bounded plain UTF-8 text. This is structural hygiene,
// not a semantic prompt-injection classifier; recipients must treat it as data.
func SanitizeBanter(raw string) string {
	if len(raw) == 0 || len(raw) > MaxBanterInputBytes {
		return ""
	}
	raw = html.UnescapeString(raw)
	var out strings.Builder
	out.Grow(min(len(raw), MaxBanterLen*utf8.UTFMax))
	count := 0
	space := false
	blocked := ""
	for i := 0; i < len(raw) && count < MaxBanterLen; {
		if raw[i] == '<' {
			end := strings.IndexByte(raw[i:], '>')
			if end < 0 {
				break
			} // Fail closed on incomplete markup.
			tag := strings.TrimSpace(raw[i+1 : i+end])
			closing := strings.HasPrefix(tag, "/")
			tag = strings.TrimSpace(strings.TrimPrefix(tag, "/"))
			nameEnd := strings.IndexFunc(tag, func(r rune) bool { return unicode.IsSpace(r) || r == '/' })
			if nameEnd >= 0 {
				tag = tag[:nameEnd]
			}
			if blocked != "" {
				if closing && strings.EqualFold(tag, blocked) {
					blocked = ""
				}
			} else if !closing && (strings.EqualFold(tag, "script") || strings.EqualFold(tag, "style")) {
				blocked = strings.ToLower(tag)
			}
			i += end + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(raw[i:])
		i += size
		if blocked != "" || (r == utf8.RuneError && size == 1) {
			continue
		}
		if unicode.IsSpace(r) {
			space = count > 0
			continue
		}
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) || r == 0xfffe || r == 0xffff {
			continue
		}
		switch r {
		case '`', '*', '_', '~', '\\', '{', '}', '[', ']', '(', ')', '#', '|', '"', '<', '>':
			continue
		}
		if space {
			out.WriteByte(' ')
			count++
			space = false
			if count >= MaxBanterLen {
				break
			}
		}
		out.WriteRune(r)
		count++
	}
	return strings.TrimSpace(out.String())
}

// QuarantineTag owns its trust boundary: callers may pass raw or previously
// sanitized text. Escaping is mandatory even after sanitizing (notably '&').
func QuarantineTag(text string) string {
	return `<opponent_banter untrusted="true">` + html.EscapeString(SanitizeBanter(text)) + `</opponent_banter>`
}
