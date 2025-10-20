package render

import (
	"html"
	"regexp"
	"strings"
)

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// TruncateText truncates text to maxChars, adding ellipsis if truncated.
// Strips HTML tags and decodes HTML entities before truncating.
// Tries to break at word boundaries to avoid cutting words.
func TruncateText(text string, maxChars int) string {
	// Strip HTML tags
	text = htmlTagRegex.ReplaceAllString(text, "")

	// Decode HTML entities (&nbsp; → space, &amp; → &, etc.)
	text = html.UnescapeString(text)

	// Normalize whitespace (collapse multiple spaces)
	text = strings.Join(strings.Fields(text), " ")

	if len(text) <= maxChars {
		return text
	}

	// Find last space before maxChars to avoid cutting words
	truncated := text[:maxChars]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "…"
}
