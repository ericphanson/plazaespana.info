package render

import "strings"

// TruncateText truncates text to maxChars, adding ellipsis if truncated.
// Tries to break at word boundaries to avoid cutting words.
func TruncateText(text string, maxChars int) string {
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
