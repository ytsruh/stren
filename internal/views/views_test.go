package views

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
	// The web app is read-only for exercise entries (logging happens
	// in the iOS client), so the empty state must not offer a
	// server-side "Add Set" action.
	if strings.Contains(html, "Add Set") || strings.Contains(html, "/exercise-entries/new") {
		t.Error("empty state should not link to the removed set-logging form")
	}
}

func TestLayout_AuthenticatedSidebarLinks(t *testing.T) {
	html := renderToString(t, Layout(PageData{
		Title:           "Dashboard",
		UserName:        "Test User",
		IsAuthenticated: true,
	}))

	for _, want := range []string{
		`href="/profile"`,
		`href="/feedback"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected authenticated sidebar to contain %q", want)
		}
	}
}

func TestLayout_FeedbackNavMarksCurrentPage(t *testing.T) {
	// The sidebar marks the active page with aria-current="page"
	// so screen readers and styling can highlight it.
	html := renderToString(t, Layout(PageData{
		Title:           "Submit Feedback",
		UserName:        "Test User",
		IsAuthenticated: true,
		CurrentPath:     "/feedback",
	}))
	if !strings.Contains(html, `aria-current="page"`) {
		t.Error("expected aria-current on the active nav item")
	}
}

func TestLayout_UnauthenticatedHidesSidebar(t *testing.T) {
	html := renderToString(t, Layout(PageData{Title: "Login"}))
	if strings.Contains(html, `href="/feedback"`) || strings.Contains(html, `href="/profile"`) {
		t.Error("expected no sidebar links for unauthenticated visitors")
	}
}
