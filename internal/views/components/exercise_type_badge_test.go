package components

import (
	"strings"
	"testing"
)

// TestExerciseTypeBadge_CardioUsesSecondaryVariant locks in the
// type-colour distinction: cardio exercises render the secondary
// badge variant so they read differently from strength/other.
func TestExerciseTypeBadge_CardioUsesSecondaryVariant(t *testing.T) {
	html := renderToString(t, ExerciseTypeBadge("cardio"))
	if !strings.Contains(html, `data-variant="secondary"`) {
		t.Error("expected cardio badge to carry data-variant=secondary")
	}
	if !strings.Contains(html, ">cardio<") {
		t.Errorf("expected label text %q in %q", "cardio", html)
	}
}

// TestExerciseTypeBadge_OtherTypesUseDefault verifies strength and
// other keep the default badge styling (no variant attribute).
func TestExerciseTypeBadge_OtherTypesUseDefault(t *testing.T) {
	for _, typ := range []string{"strength", "other", ""} {
		html := renderToString(t, ExerciseTypeBadge(typ))
		if strings.Contains(html, "data-variant") {
			t.Errorf("type %q: did not expect a data-variant attribute", typ)
		}
		if !strings.Contains(html, `class="badge capitalize"`) {
			t.Errorf("type %q: expected base badge classes, got %q", typ, html)
		}
	}
}

// TestExerciseTypeBadge_ExtraClassesAppended verifies layout utility
// classes flow through after the base classes (used by the history
// page's hidden sm:flex responsive hiding).
func TestExerciseTypeBadge_ExtraClassesAppended(t *testing.T) {
	html := renderToString(t, ExerciseTypeBadge("strength", "hidden", "sm:flex"))
	if !strings.Contains(html, `class="badge capitalize hidden sm:flex"`) {
		t.Errorf("expected extra classes appended, got %q", html)
	}
}
