package render

import "testing"

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxChars int
		want     string
	}{
		{
			name:     "short text unchanged",
			text:     "Short text",
			maxChars: 100,
			want:     "Short text",
		},
		{
			name:     "exact length unchanged",
			text:     "Exactly 20 chars!!!",
			maxChars: 19,
			want:     "Exactly 20 chars!!!",
		},
		{
			name:     "truncate at word boundary",
			text:     "This is a long description that needs truncation",
			maxChars: 25,
			want:     "This is a long…",
		},
		{
			name:     "truncate without spaces",
			text:     "Verylongtextwithoutspaces",
			maxChars: 10,
			want:     "Verylongte…",
		},
		{
			name:     "truncate preserves words",
			text:     "Event happening at Plaza de España",
			maxChars: 20,
			want:     "Event happening at…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateText(tt.text, tt.maxChars)
			if got != tt.want {
				t.Errorf("TruncateText() = %q, want %q", got, tt.want)
			}
		})
	}
}
