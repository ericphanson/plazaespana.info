package filter

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// MatchesLocation checks if event text mentions target location.
// Used as fallback when coordinates are missing.
func MatchesLocation(venueName, address, description string, keywords []string) bool {
	// Combine all text fields
	text := strings.ToLower(venueName + " " + address + " " + description)

	// Check if any keyword appears
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// normalizeText removes accents, converts to lowercase, collapses whitespace.
// This enables accent-insensitive matching for "Plaza de España" variants.
func normalizeText(s string) string {
	// Remove diacritics (accents) using Unicode normalization
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)

	// Convert to lowercase
	result = strings.ToLower(result)

	// Collapse multiple whitespace to single space
	result = strings.Join(strings.Fields(result), " ")

	return result
}

// plazaEspanaVariants returns all normalized variants of "Plaza de España" to search for.
// Includes abbreviated forms (Pza., Pl., etc.) commonly used in Spanish.
func plazaEspanaVariants() []string {
	return []string{
		"plaza de espana",
		"plaza espana",
		"pza espana",
		"pza de espana",
		"pl espana",
		"pl de espana",
		"plz espana",
		"pza. espana",
		"pza. de espana",
		"pl. espana",
		"pl. de espana",
	}
}

// MatchesPlazaEspana checks if any field mentions "Plaza de España" (accent-insensitive).
// Searches across Title, Venue, Address, and Description fields.
// Returns true if any variant of "Plaza de España" is found.
//
// This function is used for multi-venue city events that explicitly list
// Plaza de España as one of their venues, even if their canonical coordinates
// point to a different location.
func MatchesPlazaEspana(title, venue, address, description string) bool {
	// Combine all fields into searchable text
	combined := strings.Join([]string{title, venue, address, description}, " ")

	// Normalize (remove accents, lowercase, collapse spaces)
	normalized := normalizeText(combined)

	// Check all variants
	variants := plazaEspanaVariants()
	for _, variant := range variants {
		if strings.Contains(normalized, variant) {
			return true
		}
	}

	return false
}
