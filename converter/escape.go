package converter

import "strings"

// MarkdownV2 special characters that must be escaped in text nodes.
var markdownV2Special = []string{
	`\`, `_`, `*`, `[`, `]`, `(`, `)`, `~`, "`", `>`, `#`, `+`, `-`, `=`, `|`, `{`, `}`, `.`, `!`,
}

// EscapeMarkdownV2 escapes all MarkdownV2 special characters in plain text.
func EscapeMarkdownV2(s string) string {
	// Backslash must be escaped first to avoid double-escaping
	for _, ch := range markdownV2Special {
		s = strings.ReplaceAll(s, ch, `\`+ch)
	}
	return s
}

// EscapeCodeSpan escapes only ` and \ inside inline code.
func EscapeCodeSpan(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "`", "\\`")
	return s
}

// EscapeURL escapes only ) and \ inside link URLs.
func EscapeURL(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `)`, `\)`)
	return s
}
