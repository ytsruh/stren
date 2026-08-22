package export

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

// renderToString renders a templ component to a string for assertions.
func renderToString(t *testing.T, component templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("failed to render component: %v", err)
	}
	return buf.String()
}

// TestDataExportPage_RendersBothExports confirms the page surfaces
// both download links — weight entries and exercise entries — since
// a missing anchor here silently removes a data-portability option.
func TestDataExportPage_RendersBothExports(t *testing.T) {
	html := renderToString(t, DataExportPage("Test User", false))
	for _, want := range []string{
		"Data Export",
		"Weight export",
		"Exercise entries export",
		`href="/weight/export"`,
		`href="/exercises/export"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected rendered page to contain %q", want)
		}
	}
}
