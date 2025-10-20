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
			name:    "match in address",
			address: "Calle de la Plaza de España, 1",
			keywords: []string{"plaza de españa"},
			want:    true,
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
			name:      "nearby landmark",
			address:   "Parque del Oeste",
			keywords:  []string{"parque del oeste"},
			want:      true,
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
