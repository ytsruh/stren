package views

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

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

func TestFormatDateTimeLocal(t *testing.T) {
	tm := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	got := FormatDateTimeLocal(tm)
	if got != "2024-06-15T14:30" {
		t.Errorf("FormatDateTimeLocal = %q, want %q", got, "2024-06-15T14:30")
	}
}

func TestToast_Error(t *testing.T) {
	html := renderToString(t, Toast("error", "Error", "Bad input"))
	if !strings.Contains(html, "Bad input") {
		t.Error("expected toast message in output")
	}
	if !strings.Contains(html, "data-category=\"error\"") {
		t.Error("expected error data-category in output")
	}
}

func TestToast_Success(t *testing.T) {
	html := renderToString(t, Toast("success", "Success", "Saved!"))
	if !strings.Contains(html, "Saved!") {
		t.Error("expected toast message in output")
	}
	if !strings.Contains(html, "data-category=\"success\"") {
		t.Error("expected success data-category in output")
	}
}

func TestEmptyState(t *testing.T) {
	html := renderToString(t, EmptyState())
	if !strings.Contains(html, "No workouts in the last 7 days") {
		t.Error("expected empty state heading")
	}
	if !strings.Contains(html, "Add Entry") {
		t.Error("expected call-to-action button")
	}
}
