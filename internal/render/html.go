package render

import (
	"fmt"
	"html/template"
	"os"
)

// HTMLRenderer renders events to HTML using a template.
type HTMLRenderer struct {
	templatePath string
}

// NewHTMLRenderer creates an HTML renderer with the given template path.
func NewHTMLRenderer(templatePath string) *HTMLRenderer {
	return &HTMLRenderer{templatePath: templatePath}
}

// Render generates HTML output and writes it atomically to outputPath.
func (r *HTMLRenderer) Render(data TemplateData, outputPath string) error {
	tmpl, err := template.ParseFiles(r.templatePath)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	// Atomic write: temp file + rename
	tmpPath := outputPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("executing template: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("renaming output: %w", err)
	}

	return nil
}
