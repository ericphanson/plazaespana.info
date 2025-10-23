package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGeneratedHTML_NoPlaceholders verifies that generated HTML files
// do not reference placeholder CSS files.
func TestGeneratedHTML_NoPlaceholders(t *testing.T) {
	publicDir := "../../public"

	tests := []struct {
		name     string
		htmlFile string
	}{
		{"Index page", filepath.Join(publicDir, "index.html")},
		{"Build report", filepath.Join(publicDir, "build-report.html")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if file doesn't exist (site not generated yet)
			if _, err := os.Stat(tt.htmlFile); os.IsNotExist(err) {
				t.Skipf("HTML file not found: %s (run 'just generate' first)", tt.htmlFile)
			}

			content, err := os.ReadFile(tt.htmlFile)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", tt.htmlFile, err)
			}

			html := string(content)

			// Check for placeholder CSS references
			if strings.Contains(html, "placeholder") {
				t.Errorf("%s contains 'placeholder' CSS reference - CSS hash not set correctly", tt.htmlFile)
			}

			// Verify actual hash pattern exists (site.XXXXXXXX.css or build-report.XXXXXXXX.css)
			hasHashedCSS := strings.Contains(html, "site.") && strings.Contains(html, ".css") ||
				strings.Contains(html, "build-report.") && strings.Contains(html, ".css")
			if !hasHashedCSS {
				t.Errorf("%s does not contain hashed CSS reference", tt.htmlFile)
			}
		})
	}
}

// TestGeneratedHTML_CSPCompliant verifies that generated HTML files
// are CSP-compliant (no inline styles).
func TestGeneratedHTML_CSPCompliant(t *testing.T) {
	publicDir := "../../public"

	tests := []struct {
		name     string
		htmlFile string
	}{
		{"Index page", filepath.Join(publicDir, "index.html")},
		{"Build report", filepath.Join(publicDir, "build-report.html")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if file doesn't exist (site not generated yet)
			if _, err := os.Stat(tt.htmlFile); os.IsNotExist(err) {
				t.Skipf("HTML file not found: %s (run 'just generate' first)", tt.htmlFile)
			}

			content, err := os.ReadFile(tt.htmlFile)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", tt.htmlFile, err)
			}

			html := string(content)

			// Check for inline <style> tags
			if strings.Contains(html, "<style") {
				t.Errorf("%s contains inline <style> tag - violates CSP style-src 'self'", tt.htmlFile)
			}

			// Check for inline style attributes
			if strings.Contains(html, " style=\"") || strings.Contains(html, " style='") {
				t.Errorf("%s contains inline style attribute - violates CSP style-src 'self'", tt.htmlFile)
			}

			// Verify external CSS is referenced
			if !strings.Contains(html, "<link rel=\"stylesheet\" href=\"/assets/") {
				t.Errorf("%s does not reference external CSS - should use <link rel=\"stylesheet\">", tt.htmlFile)
			}
		})
	}
}
