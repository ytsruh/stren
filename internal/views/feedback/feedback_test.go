package feedback

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

func TestFeedbackPage_RendersForm(t *testing.T) {
	html := renderToString(t, FeedbackPage("Test User", true, false))

	for _, want := range []string{
		"Submit Feedback",
		`id="feedback-form"`,
		`hx-post="/feedback"`,
		`name="title"`,
		`name="message"`,
		`id="feedback-spinner"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected feedback page HTML to contain %q", want)
		}
	}
}

func TestFeedbackPage_SuccessEventWiring(t *testing.T) {
	// The page must listen for the same event the POST handler
	// triggers (HX-Trigger: feedbackSubmitted) so the form resets
	// after a successful submission.
	html := renderToString(t, FeedbackPage("Test User", true, false))
	if !strings.Contains(html, "feedbackSubmitted") {
		t.Error("expected page to listen for the feedbackSubmitted event")
	}
}

func TestFeedbackFormSuccess_IsSuccessToast(t *testing.T) {
	html := renderToString(t, FeedbackFormSuccess("Thanks for your feedback"))
	if !strings.Contains(html, `data-category="success"`) {
		t.Error("expected success toast markup")
	}
	if !strings.Contains(html, "Thanks for your feedback") {
		t.Error("expected toast message in output")
	}
}

func TestFeedbackFormError_IsErrorToast(t *testing.T) {
	html := renderToString(t, FeedbackFormError("Title must be at least 5 characters"))
	if !strings.Contains(html, `data-category="error"`) {
		t.Error("expected error toast markup")
	}
	if !strings.Contains(html, "Title must be at least 5 characters") {
		t.Error("expected error message in output")
	}
}
