package weight

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"stren/internal/models"

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

// TestWeightProgress covers the boundaries of the progress formula. The
// helper is unexported but lives in this package, so the tests can call it
// directly to assert exact behaviour. Higher-level rendering of the
// WeightProgressCard is covered by TestWeightProgressCard_*.
//
// The formula is simply current / target * 100, clamped to [0, 100].
// The start argument is no longer used by the formula but kept in the
// signature for backwards compatibility with existing test cases.
func TestWeightProgress(t *testing.T) {
	tests := []struct {
		name         string
		start, current, target float64
		wantPct      float64
		wantContains string // substring expected in the label
	}{
		{
			name:         "current below target (gaining goal)",
			start:        69, current: 67.6, target: 70,
			wantPct:      96.57142857142857,
			wantContains: "2.4 kg to go",
		},
		{
			name:         "current above target (cutting goal)",
			start:        90, current: 95, target: 90,
			wantPct:      100, // 95/90 = 105.5%, clamped
			wantContains: "5.0 kg to go",
		},
		{
			name:         "exactly at goal",
			start:        80, current: 80, target: 80,
			wantPct:      100,
			wantContains: "Goal reached",
		},
		{
			name:         "current is 90% of target",
			start:        100, current: 90, target: 100,
			wantPct:      90,
			wantContains: "10.0 kg to go",
		},
		{
			name:         "current is 50% of target",
			start:        100, current: 50, target: 100,
			wantPct:      50,
			wantContains: "50.0 kg to go",
		},
		{
			name:         "target is zero (no goal)",
			start:        100, current: 90, target: 0,
			wantPct:      0,
			wantContains: "Set a new target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pct, label := weightProgress(tt.start, tt.current, tt.target)
			if pct != tt.wantPct {
				t.Errorf("weightProgress(%v, %v, %v) pct = %v, want %v", tt.start, tt.current, tt.target, pct, tt.wantPct)
			}
			if !strings.Contains(label, tt.wantContains) {
				t.Errorf("weightProgress(%v, %v, %v) label = %q, want substring %q", tt.start, tt.current, tt.target, label, tt.wantContains)
			}
		})
	}
}

// TestWeightProgressCard_RendersKeyStats asserts the card surfaces the
// start/current/target values and the percent progress to the user.
func TestWeightProgressCard_RendersKeyStats(t *testing.T) {
	day1 := time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC)
	day2 := time.Date(2024, 6, 8, 8, 0, 0, 0, time.UTC)
	entries := []models.WeightEntry{
		{ID: "w1", Weight: 80, CreatedAt: day1},
		{ID: "w2", Weight: 50, CreatedAt: day2},
	}

	html := renderToString(t, WeightProgressCard(entries, 100))

	// Headline percent: current 50 / target 100 = 50%.
	if !strings.Contains(html, "50%") {
		t.Error("expected 50% progress in rendered card")
	}
	// Start, current, and target values appear in the stats grid.
	for _, want := range []string{"80.0 kg", "50.0 kg", "100.0 kg"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in rendered card", want)
		}
	}
	// Accessibility: a progressbar with aria-valuenow.
	if !strings.Contains(html, `role="progressbar"`) {
		t.Error("expected progressbar role")
	}
	if !strings.Contains(html, `aria-valuenow="50"`) {
		t.Errorf("expected aria-valuenow=50, got: %s", html)
	}
}

// TestWeightProgressCard_UsesSortedOrder asserts the card picks the
// earliest entry as "start" and the latest as "current", even when the
// caller passes them in descending order (the order the repo returns).
func TestWeightProgressCard_UsesSortedOrder(t *testing.T) {
	day1 := time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC)
	day2 := time.Date(2024, 6, 8, 8, 0, 0, 0, time.UTC)
	day3 := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)
	// Reverse-chronological order, as the weight repo returns.
	entries := []models.WeightEntry{
		{ID: "w3", Weight: 30, CreatedAt: day3},
		{ID: "w2", Weight: 35, CreatedAt: day2},
		{ID: "w1", Weight: 40, CreatedAt: day1},
	}

	html := renderToString(t, WeightProgressCard(entries, 50))

	// current 30 / target 50 = 60%.
	if !strings.Contains(html, "60%") {
		t.Errorf("expected 60%% progress (current=30, target=50), got: %s", html)
	}
}

// TestWeightPage_ChartIsFullWidthWithoutGoal locks in the responsive
// behaviour: when the user has no target weight, the chart wrapper
// takes the full row on sm+ screens (no half-empty space to the right).
// When a target is set, the chart drops to 3/4 and the progress card
// takes the remaining 1/4.
func TestWeightPage_ChartIsFullWidthWithoutGoal(t *testing.T) {
	day1 := time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC)
	day2 := time.Date(2024, 6, 8, 8, 0, 0, 0, time.UTC)
	entries := []models.WeightEntry{
		{ID: "w1", Weight: 100, CreatedAt: day1},
		{ID: "w2", Weight: 95, CreatedAt: day2},
	}

	t.Run("no goal", func(t *testing.T) {
		html := renderToString(t, WeightPage(entries, "Test User", true, false, nil))
		// Chart wrapper must NOT carry sm:w-3/4 when there's no goal.
		if strings.Contains(html, "sm:w-3/4") {
			t.Error("expected chart to be full width on sm+ when no target weight is set")
		}
		// Progress card must not be rendered at all.
		if strings.Contains(html, "Goal progress") {
			t.Error("expected progress card to be omitted when no target weight is set")
		}
	})

	t.Run("with goal", func(t *testing.T) {
		target := 80.0
		html := renderToString(t, WeightPage(entries, "Test User", true, false, &target))
		// Chart wrapper DOES carry sm:w-3/4 when a goal is set.
		if !strings.Contains(html, "sm:w-3/4") {
			t.Error("expected chart to be 3/4 width on sm+ when a target weight is set")
		}
		// Progress card IS rendered.
		if !strings.Contains(html, "Goal progress") {
			t.Error("expected progress card to render when a target weight is set")
		}
	})
}

// TestWeightPage_ProgressCardHiddenWithoutEntries locks in the panic
// guard: a user who has set a goal but logged no entries must not see
// the progress card (which would index into an empty slice).
func TestWeightPage_ProgressCardHiddenWithoutEntries(t *testing.T) {
	target := 80.0
	html := renderToString(t, WeightPage(nil, "Test User", true, false, &target))
	if strings.Contains(html, "Goal progress") {
		t.Error("expected progress card to be omitted when there are no entries")
	}
	// The chart still takes the full row — same layout as the no-goal case.
	if strings.Contains(html, "sm:w-3/4") {
		t.Error("expected chart to be full width on sm+ when no entries exist")
	}
}
