package parley

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Banter represents safe, quarantined banter text.
type Banter string

const MaxBanterLen = 140

var (
	tagPattern = regexp.MustCompile(`(?i)<[^>]*>`)
)

// SanitizeBanter sanitizes banter text:
// 1. Strips HTML/XML tags completely (including tag contents if script/style/etc).
// 2. Strips backticks, markdown control characters (*, _, ~, #, |, etc.), and dangerous control sequences.
// 3. Replaces newlines and tabs with spaces.
// 4. Caps at 140 UTF-8 characters.
func SanitizeBanter(raw string) string {
	if raw == "" {
		return ""
	}

	// Remove script blocks completely
	reScript := regexp.MustCompile(`(?is)<script.*?>.*?</script>`)
	cleaned := reScript.ReplaceAllString(raw, "")

	// Remove any remaining tags
	cleaned = tagPattern.ReplaceAllString(cleaned, "")

	// Remove markdown control characters & structural punctuation:
	// backticks, *, _, ~, \, {, }, [, ], (, ), #, |, double-quotes
	var sb strings.Builder
	for _, r := range cleaned {
		switch r {
		case '`', '*', '_', '~', '\\', '{', '}', '[', ']', '(', ')', '#', '|', '"':
			continue
		case '\r', '\n', '\t':
			sb.WriteRune(' ')
		default:
			// Filter unprintable/control ASCII
			if r < 32 || r == 127 {
				continue
			}
			sb.WriteRune(r)
		}
	}

	result := strings.TrimSpace(sb.String())
	// Collapse multiple spaces
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	// Cap at 140 UTF-8 characters
	if utf8.RuneCountInString(result) > MaxBanterLen {
		runes := []rune(result)
		result = string(runes[:MaxBanterLen])
	}

	return strings.TrimSpace(result)
}

// QuarantineTag wraps safe banter in structural XML quarantine tags.
func QuarantineTag(sanitized string) string {
	if sanitized == "" {
		return `<opponent_banter untrusted="true"></opponent_banter>`
	}
	return `<opponent_banter untrusted="true">` + sanitized + `</opponent_banter>`
}
