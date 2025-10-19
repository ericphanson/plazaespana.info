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
{{range .Events}}<article><h2>{{.Titulo}}</h2></article>{{end}}
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
		Events: []TemplateEvent{
			{Titulo: "Test Event 1"},
			{Titulo: "Test Event 2"},
		},
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
	if !strings.Contains(contentStr, "Test Event 1") {
		t.Error("Output missing 'Test Event 1'")
	}
	if !strings.Contains(contentStr, "Test Event 2") {
		t.Error("Output missing 'Test Event 2'")
	}
}
