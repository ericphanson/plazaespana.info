package filter

import "testing"

func TestMatchesLocation(t *testing.T) {
	tests := []struct {
		name        string
		venueName   string
		address     string
		description string
		keywords    []string
		want        bool
	}{
		{
			name:      "exact match in venue name",
			venueName: "Plaza de España",
			keywords:  []string{"plaza de españa"},
			want:      true,
		},
		{
			name:      "case insensitive match",
			venueName: "PLAZA DE ESPAÑA",
			keywords:  []string{"plaza de españa"},
			want:      true,
		},
		{
			name:      "partial match in venue name",
			venueName: "Auditorio Plaza de España",
			keywords:  []string{"plaza de españa"},
			want:      true,
		},
		{
			name:     "match in address",
			address:  "Calle de la Plaza de España, 1",
			keywords: []string{"plaza de españa"},
			want:     true,
		},
		{
			name:        "match in description",
			description: "Evento cerca de Plaza de España",
			keywords:    []string{"plaza de españa"},
			want:        true,
		},
		{
			name:      "no match",
			venueName: "Teatro Real",
			address:   "Plaza de Oriente",
			keywords:  []string{"plaza de españa"},
			want:      false,
		},
		{
			name:      "match any keyword",
			venueName: "Templo de Debod",
			keywords:  []string{"plaza de españa", "templo de debod"},
			want:      true,
		},
		{
			name:      "variation without 'de'",
			venueName: "Plaza España",
			keywords:  []string{"plaza españa"},
			want:      true,
		},
		{
			name:     "nearby landmark",
			address:  "Parque del Oeste",
			keywords: []string{"parque del oeste"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesLocation(tt.venueName, tt.address, tt.description, tt.keywords)
			if got != tt.want {
				t.Errorf("MatchesLocation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Plaza de España", "plaza de espana"},
		{"PLAZA   DE   ESPAÑA", "plaza de espana"},
		{"Pza. España", "pza. espana"},
		{"  extra   spaces  ", "extra spaces"},
		{"José María", "jose maria"},
		{"CAFÉ", "cafe"},
	}

	for _, tt := range tests {
		got := normalizeText(tt.input)
		if got != tt.want {
			t.Errorf("normalizeText(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMatchesPlazaEspana(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		venue       string
		address     string
		description string
		want        bool
	}{
		{
			name:  "title_with_accent",
			title: "Mercadillo en Plaza de España",
			want:  true,
		},
		{
			name:  "title_without_accent",
			title: "Mercadillo en Plaza de Espana",
			want:  true,
		},
		{
			name:        "description_abbreviated",
			description: "Varios puntos: Pza. España, Sol, Cibeles",
			want:        true,
		},
		{
			name:  "venue_field",
			venue: "Pl. de España",
			want:  true,
		},
		{
			name:        "no_mention",
			title:       "Evento en Plaza Mayor",
			description: "Cerca de Sol",
			want:        false,
		},
		{
			name:        "historical_reference",
			description: "Historia de la Plaza de España en el museo",
			want:        true, // Will match (filtering out historical refs is optional refinement)
		},
		{
			name:        "multi_venue",
			description: "Fiestas en Plaza de Pedro Zerolo, Plaza del Rey, Plaza de España, y Sol",
			want:        true,
		},
		{
			name:        "uppercase_variant",
			description: "PLAZA ESPAÑA",
			want:        true,
		},
		{
			name:    "address_field",
			address: "Pza de España, 28008 Madrid",
			want:    true,
		},
		{
			name:        "different_plaza",
			title:       "Evento en Plaza de Cibeles",
			description: "Junto a la fuente",
			want:        false,
		},
		{
			name:        "abbreviated_plz",
			description: "Plz España",
			want:        true,
		},
		{
			name:  "abbreviated_pl_no_period",
			venue: "Pl España",
			want:  true,
		},
		{
			name:        "combined_fields",
			title:       "Festival de música",
			description: "Múltiples ubicaciones incluyendo Plaza de España",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesPlazaEspana(tt.title, tt.venue, tt.address, tt.description)
			if got != tt.want {
				t.Errorf("MatchesPlazaEspana() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlazaEspanaVariants(t *testing.T) {
	variants := plazaEspanaVariants()

	// Verify all variants are normalized (lowercase, no accents)
	for _, variant := range variants {
		normalized := normalizeText(variant)
		if normalized != variant {
			t.Errorf("Variant %q is not normalized (got %q)", variant, normalized)
		}
	}

	// Verify expected variants are present
	expectedVariants := []string{
		"plaza de espana",
		"plaza espana",
		"pza espana",
		"pl espana",
	}

	for _, expected := range expectedVariants {
		found := false
		for _, variant := range variants {
			if variant == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected variant %q not found in plazaEspanaVariants()", expected)
		}
	}
}
