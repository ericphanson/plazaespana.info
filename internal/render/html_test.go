package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHTMLRenderer_Render(t *testing.T) {
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "index.tmpl.html")

	// Create minimal template
	tmpl := `<!doctype html>
<html lang="{{.Lang}}">
<head><title>Test</title></head>
<body>
<p>Updated: {{.LastUpdated}}</p>
{{range .CulturalEvents}}<article><h3>{{.Titulo}}</h3></article>{{end}}
{{range .CityEvents}}<article><h3>{{.Titulo}}</h3></article>{{end}}
</body>
</html>`

	if err := os.WriteFile(templatePath, []byte(tmpl), 0644); err != nil {
		t.Fatalf("Failed to write template: %v", err)
	}

	renderer := NewHTMLRenderer(templatePath)

	data := TemplateData{
		Lang:        "es",
		CSSHash:     "abc123",
		LastUpdated: time.Now().Format("2006-01-02 15:04"),
		CulturalEvents: []TemplateEvent{
			{Titulo: "Cultural Event 1"},
			{Titulo: "Cultural Event 2"},
		},
		CityEvents: []TemplateEvent{
			{Titulo: "City Event 1"},
		},
		TotalEvents: 3,
	}

	outputPath := filepath.Join(tmpDir, "index.html")
	err := renderer.Render(data, outputPath)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify output file exists
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Cultural Event 1") {
		t.Error("Output missing 'Cultural Event 1'")
	}
	if !strings.Contains(contentStr, "Cultural Event 2") {
		t.Error("Output missing 'Cultural Event 2'")
	}
	if !strings.Contains(contentStr, "City Event 1") {
		t.Error("Output missing 'City Event 1'")
	}
}

func TestHTMLRenderer_DualSection(t *testing.T) {
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "index.tmpl.html")

	// Create template with sections
	tmpl := `<!doctype html>
<html lang="{{.Lang}}">
<head><title>Dual Section Test</title></head>
<body>
{{if gt (len .CulturalEvents) 0}}
<section class="cultural-section">
  <h2>Cultural Events ({{len .CulturalEvents}})</h2>
  {{range .CulturalEvents}}<article class="cultural"><h3>{{.Titulo}}</h3></article>{{end}}
</section>
{{end}}
{{if gt (len .CityEvents) 0}}
<section class="city-section">
  <h2>City Events ({{len .CityEvents}})</h2>
  {{range .CityEvents}}<article class="city"><h3>{{.Titulo}}</h3></article>{{end}}
</section>
{{end}}
<p>Total: {{.TotalEvents}}</p>
</body>
</html>`

	if err := os.WriteFile(templatePath, []byte(tmpl), 0644); err != nil {
		t.Fatalf("Failed to write template: %v", err)
	}

	renderer := NewHTMLRenderer(templatePath)

	tests := []struct {
		name           string
		data           TemplateData
		wantCultural   string
		wantCity       string
		wantTotal      string
		wantSections   int
	}{
		{
			name: "Both event types present",
			data: TemplateData{
				Lang:        "es",
				CSSHash:     "test123",
				LastUpdated: "2025-10-20",
				CulturalEvents: []TemplateEvent{
					{Titulo: "Teatro Nacional"},
					{Titulo: "Museo Prado"},
				},
				CityEvents: []TemplateEvent{
					{Titulo: "Madrid Gaming Festival"},
				},
				TotalEvents: 3,
			},
			wantCultural: "Teatro Nacional",
			wantCity:     "Madrid Gaming Festival",
			wantTotal:    "Total: 3",
			wantSections: 2,
		},
		{
			name: "Only cultural events",
			data: TemplateData{
				Lang:        "es",
				CSSHash:     "test123",
				LastUpdated: "2025-10-20",
				CulturalEvents: []TemplateEvent{
					{Titulo: "Concierto Sinfónico"},
				},
				CityEvents:  []TemplateEvent{},
				TotalEvents: 1,
			},
			wantCultural: "Concierto Sinfónico",
			wantCity:     "",
			wantTotal:    "Total: 1",
			wantSections: 1,
		},
		{
			name: "Only city events",
			data: TemplateData{
				Lang:           "es",
				CSSHash:        "test123",
				LastUpdated:    "2025-10-20",
				CulturalEvents: []TemplateEvent{},
				CityEvents: []TemplateEvent{
					{Titulo: "Festival de Otoño"},
				},
				TotalEvents: 1,
			},
			wantCultural: "",
			wantCity:     "Festival de Otoño",
			wantTotal:    "Total: 1",
			wantSections: 1,
		},
		{
			name: "No events",
			data: TemplateData{
				Lang:           "es",
				CSSHash:        "test123",
				LastUpdated:    "2025-10-20",
				CulturalEvents: []TemplateEvent{},
				CityEvents:     []TemplateEvent{},
				TotalEvents:    0,
			},
			wantCultural: "",
			wantCity:     "",
			wantTotal:    "Total: 0",
			wantSections: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, "output_"+tt.name+".html")
			err := renderer.Render(tt.data, outputPath)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read output: %v", err)
			}

			contentStr := string(content)

			// Check for expected content
			if tt.wantCultural != "" && !strings.Contains(contentStr, tt.wantCultural) {
				t.Errorf("Output missing cultural event: %q", tt.wantCultural)
			}
			if tt.wantCity != "" && !strings.Contains(contentStr, tt.wantCity) {
				t.Errorf("Output missing city event: %q", tt.wantCity)
			}
			if !strings.Contains(contentStr, tt.wantTotal) {
				t.Errorf("Output missing total: %q", tt.wantTotal)
			}

			// Check section presence
			culturalCount := strings.Count(contentStr, "cultural-section")
			cityCount := strings.Count(contentStr, "city-section")
			totalSections := culturalCount + cityCount

			if totalSections != tt.wantSections {
				t.Errorf("Expected %d sections, got %d (cultural: %d, city: %d)",
					tt.wantSections, totalSections, culturalCount, cityCount)
			}
		})
	}
}

func TestHTMLRenderer_RealTemplate(t *testing.T) {
	// Test with the actual template file
	templatePath := filepath.Join("..", "..", "templates", "index.tmpl.html")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skip("Real template not found, skipping integration test")
	}

	renderer := NewHTMLRenderer(templatePath)
	tmpDir := t.TempDir()

	data := TemplateData{
		Lang:        "es",
		CSSHash:     "abc123",
		LastUpdated: time.Now().Format("2006-01-02 15:04 MST"),
		CulturalEvents: []TemplateEvent{
			{
				IDEvento:          "1",
				Titulo:            "Exposición de Arte Moderno",
				StartHuman:        "20/10/2025 18:00",
				NombreInstalacion: "Museo Reina Sofía",
				ContentURL:        "https://example.com/event1",
				Description:       "Una exposición fascinante de arte moderno español.",
			},
		},
		CityEvents: []TemplateEvent{
			{
				IDEvento:          "2",
				Titulo:            "Festival de Videojuegos",
				StartHuman:        "25/10/2025",
				NombreInstalacion: "IFEMA Madrid",
				ContentURL:        "https://example.com/event2",
				Description:       "El mayor festival de gaming de España.",
			},
		},
		TotalEvents: 2,
	}

	outputPath := filepath.Join(tmpDir, "index.html")
	err := renderer.Render(data, outputPath)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	contentStr := string(content)

	// Verify structure
	requiredElements := []string{
		"Exposición de Arte Moderno",
		"Festival de Videojuegos",
		"cultural-section",
		"city-section",
		"cultural-badge",
		"city-badge",
		"event-card",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(contentStr, elem) {
			t.Errorf("Output missing required element: %q", elem)
		}
	}

	// Verify HTML structure
	if !strings.Contains(contentStr, "<!doctype html>") {
		t.Error("Missing DOCTYPE declaration")
	}
	if !strings.Contains(contentStr, "<html lang=\"es\">") {
		t.Error("Missing or incorrect lang attribute")
	}
}
