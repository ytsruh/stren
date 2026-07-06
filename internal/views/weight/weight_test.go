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
func TestWeightProgress(t *testing.T) {
	tests := []struct {
		name    string
		current float64
		target  float64
		wantPct float64
	}{
		{
			name:    "current below target (gaining goal)",
			current: 67.6, target: 70,
			wantPct: 96.57142857142857,
		},
		{
			name:    "current above target (cutting goal)",
			current: 95, target: 90,
			wantPct: 100, // 95/90 = 105.5%, clamped
		},
		{
			name:    "exactly at goal",
			current: 80, target: 80,
			wantPct: 100,
		},
		{
			name:    "current is 90% of target",
			current: 90, target: 100,
			wantPct: 90,
		},
		{
			name:    "current is 50% of target",
			current: 50, target: 100,
			wantPct: 50,
		},
		{
			name:    "target is zero (no goal)",
			current: 90, target: 0,
			wantPct: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pct := weightProgress(tt.current, tt.target)
			if pct != tt.wantPct {
				t.Errorf("weightProgress(%v, %v) = %v, want %v", tt.current, tt.target, pct, tt.wantPct)
			}
		})
	}
}

// TestLatestWeight asserts the helper returns the weight from the most
// recent entry, regardless of the input order. The chart sorts
// defensively too, so this guarantees both consumers pick the same
// "current" reading.
func TestLatestWeight(t *testing.T) {
	day1 := time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC)
	day2 := time.Date(2024, 6, 8, 8, 0, 0, 0, time.UTC)
	day3 := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)

	t.Run("ascending order", func(t *testing.T) {
		entries := []models.WeightEntry{
			{ID: "w1", Weight: 80, CreatedAt: day1},
			{ID: "w2", Weight: 78, CreatedAt: day2},
			{ID: "w3", Weight: 76, CreatedAt: day3},
		}
		if got, want := latestWeight(entries), 76.0; got != want {
			t.Errorf("latestWeight = %v, want %v", got, want)
		}
	})

	t.Run("reverse-chronological order (as the repo returns)", func(t *testing.T) {
		entries := []models.WeightEntry{
			{ID: "w3", Weight: 30, CreatedAt: day3},
			{ID: "w2", Weight: 35, CreatedAt: day2},
			{ID: "w1", Weight: 40, CreatedAt: day1},
		}
		if got, want := latestWeight(entries), 30.0; got != want {
			t.Errorf("latestWeight = %v, want %v", got, want)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		if got := latestWeight(nil); got != 0 {
			t.Errorf("latestWeight(nil) = %v, want 0", got)
		}
	})
}

// TestWeightProgressCard_RendersKeyStats asserts the card surfaces the
// current/target values and the percent progress to the user.
func TestWeightProgressCard_RendersKeyStats(t *testing.T) {
	html := renderToString(t, WeightProgressCard(50, 100, 50, "kg"))

	// Headline percent.
	if !strings.Contains(html, "50%") {
		t.Error("expected 50% progress in rendered card")
	}
	// Current and target values appear in the stats grid.
	for _, want := range []string{"50.0 kg", "100.0 kg"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in rendered card", want)
		}
	}
}

// TestWeightProgressCard_RendersSuppliedValues asserts the card is a
// pure render: whatever current/target/pct the caller passes in are
// what the user sees. (The "latest entry" sort and the
// current/target percent are the caller's responsibility — covered
// separately by TestLatestWeight and TestWeightProgress.)
func TestWeightProgressCard_RendersSuppliedValues(t *testing.T) {
	html := renderToString(t, WeightProgressCard(30, 50, 60, "kg"))

	// current 30, target 50, pct 60.
	if !strings.Contains(html, "60%") {
		t.Errorf("expected 60%% progress, got: %s", html)
	}
	if !strings.Contains(html, "30.0 kg") {
		t.Errorf("expected current 30.0 kg, got: %s", html)
	}
	if !strings.Contains(html, "50.0 kg") {
		t.Errorf("expected target 50.0 kg, got: %s", html)
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
		html := renderToString(t, WeightPage(entries, "Test User", true, false, nil, "kg"))
		// Chart wrapper must NOT carry sm:w-3/4 when there's no goal.
		if strings.Contains(html, "sm:w-3/4") {
			t.Error("expected chart to be full width on sm+ when no target weight is set")
		}
		// Progress card must not be rendered at all.
		if strings.Contains(html, "Progress") {
			t.Error("expected progress card to be omitted when no target weight is set")
		}
	})

	t.Run("with goal", func(t *testing.T) {
		target := 80.0
		html := renderToString(t, WeightPage(entries, "Test User", true, false, &target, "kg"))
		// Chart wrapper DOES carry sm:w-3/4 when a goal is set.
		if !strings.Contains(html, "sm:w-3/4") {
			t.Error("expected chart to be 3/4 width on sm+ when a target weight is set")
		}
		// Progress card IS rendered.
		if !strings.Contains(html, "Progress") {
			t.Error("expected progress card to render when a target weight is set")
		}
	})
}

// TestWeightPage_ProgressCardHiddenWithoutEntries locks in the panic
// guard: a user who has set a goal but logged no entries must not see
// the progress card (which would index into an empty slice).
func TestWeightPage_ProgressCardHiddenWithoutEntries(t *testing.T) {
	target := 80.0
	html := renderToString(t, WeightPage(nil, "Test User", true, false, &target, "kg"))
	if strings.Contains(html, "Progress") {
		t.Error("expected progress card to be omitted when there are no entries")
	}
	// The chart still takes the full row — same layout as the no-goal case.
	if strings.Contains(html, "sm:w-3/4") {
		t.Error("expected chart to be full width on sm+ when no entries exist")
	}
}

// TestWeightRow_RendersCompareCheckboxWhenPhoto ensures entries that
// have a photo expose a checkbox so the user can pick them for the
// image-comparison feature.
func TestWeightRow_RendersCompareCheckboxWhenPhoto(t *testing.T) {
	day := time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC)
	entry := models.WeightEntry{
		ID:        "w-1",
		Weight:    80,
		PhotoKey:  "weight/u1/abc.jpg",
		CreatedAt: day,
	}
	html := renderToString(t, WeightRow(entry, "kg"))
	if !strings.Contains(html, `data-compare-pick`) {
		t.Error("expected compare checkbox on a photo-bearing row")
	}
	if !strings.Contains(html, `data-entry-id="w-1"`) {
		t.Errorf("expected checkbox to carry the entry id, got: %s", html)
	}
}

// TestWeightRow_HidesCompareCheckboxWhenNoPhoto ensures entries
// without a photo do not expose a checkbox — the user can only pick
// entries that can actually be compared.
func TestWeightRow_HidesCompareCheckboxWhenNoPhoto(t *testing.T) {
	day := time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC)
	entry := models.WeightEntry{
		ID:        "w-2",
		Weight:    80,
		PhotoKey:  "",
		CreatedAt: day,
	}
	html := renderToString(t, WeightRow(entry, "kg"))
	if strings.Contains(html, `data-compare-pick`) {
		t.Errorf("expected no compare checkbox on a photo-less row, got: %s", html)
	}
}

// TestWeightPage_RendersCompareBarAndModalContainer ensures the page
// hosts the sticky compare bar and the empty modal container that
// htmx swaps the comparison dialog into.
func TestWeightPage_RendersCompareBarAndModalContainer(t *testing.T) {
	day1 := time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC)
	day2 := time.Date(2024, 6, 8, 8, 0, 0, 0, time.UTC)
	entries := []models.WeightEntry{
		{ID: "w1", Weight: 80, PhotoKey: "weight/u1/a.jpg", CreatedAt: day1},
		{ID: "w2", Weight: 79, PhotoKey: "weight/u1/b.jpg", CreatedAt: day2},
	}
	html := renderToString(t, WeightPage(entries, "Test User", true, false, nil, "kg"))
	if !strings.Contains(html, `id="weight-compare-bar"`) {
		t.Error("expected compare bar in page")
	}
	if !strings.Contains(html, `id="weight-compare-modal-container"`) {
		t.Error("expected modal container in page")
	}
	if !strings.Contains(html, `data-compare-pick`) {
		t.Error("expected at least one compare checkbox for photo-bearing entries")
	}
}

// TestCompareModal_RendersImagesAndLabels ensures the modal fragment
// surfaces both images, the labels, and the delta line.
func TestCompareModal_RendersImagesAndLabels(t *testing.T) {
	html := renderToString(t, CompareModal(
		"01 Jan 2026", "82.4 kg", "https://cdn.example.com/before.jpg",
		"15 Jun 2026", "78.1 kg", "https://cdn.example.com/after.jpg",
		"−4.3 kg",
	))
	if !strings.Contains(html, `https://cdn.example.com/before.jpg`) {
		t.Error("expected before image URL in modal")
	}
	if !strings.Contains(html, `https://cdn.example.com/after.jpg`) {
		t.Error("expected after image URL in modal")
	}
	if !strings.Contains(html, `01 Jan 2026`) || !strings.Contains(html, `15 Jun 2026`) {
		t.Error("expected both dates in modal")
	}
	if !strings.Contains(html, `82.4 kg`) || !strings.Contains(html, `78.1 kg`) {
		t.Error("expected both weights in modal")
	}
	if !strings.Contains(html, `id="weight-compare-slider"`) {
		t.Error("expected slider container id")
	}
	if !strings.Contains(html, `class="image-compare`) {
		t.Error("expected image-compare class on slider container")
	}
	if !strings.Contains(html, `data-photo-a=`) || !strings.Contains(html, `data-photo-b=`) {
		t.Error("expected data attributes carrying photo URLs on dialog")
	}
	if !strings.Contains(html, `data-label-a=`) || !strings.Contains(html, `data-label-b=`) {
		t.Error("expected data attributes carrying labels on dialog")
	}
}

// TestCompareModal_LabelsUseDateAndWeight ensures the labels match
// the agreed format "DD Mon YYYY · X.X kg" by checking the data-label
// attributes that the inline script uses to configure the library.
func TestCompareModal_LabelsUseDateAndWeight(t *testing.T) {
	html := renderToString(t, CompareModal(
		"01 Jan 2026", "82.4 kg", "https://cdn.example.com/before.jpg",
		"15 Jun 2026", "78.1 kg", "https://cdn.example.com/after.jpg",
		"−4.3 kg",
	))
	if !strings.Contains(html, `data-label-a="01 Jan 2026 · 82.4 kg"`) {
		t.Errorf("expected before label to combine date and weight, got: %s", html)
	}
	if !strings.Contains(html, `data-label-b="15 Jun 2026 · 78.1 kg"`) {
		t.Errorf("expected after label to combine date and weight, got: %s", html)
	}
}

// TestCompareModal_SingleCenteredChild locks in the structural fix for
// the centering bug. Basecoat's `.dialog > *` rule applies
// `position: fixed; top: 50%; left: 50%; transform: translate(-50%,-50%)`
// to every direct child of the dialog, plus the card chrome (border,
// padding, shadow, max-width). If the dialog has more than one direct
// child, each becomes its own independently-positioned panel and they
// stack on top of each other at the center, which manifests as the
// modal "not being centered" and the close buttons drifting off-screen.
//
// The fix is to wrap the inner content in a single <div> so the
// `> *` selector has exactly one target. This test asserts the dialog
// contains a single wrapper div holding the slider, and that the
// dialog itself does not carry any width/height constraints that
// would each get independently fixed-positioned.
func TestCompareModal_SingleCenteredChild(t *testing.T) {
	html := renderToString(t, CompareModal(
		"01 Jan 2026", "82.4 kg", "https://cdn.example.com/before.jpg",
		"15 Jun 2026", "78.1 kg", "https://cdn.example.com/after.jpg",
		"−4.3 kg",
	))

	// The dialog must NOT carry the size/height classes itself any
	// more — those are now on the wrapper child (which is the element
	// the basecoat `> *` rule actually styles).
	if strings.Contains(html, `class="dialog w-full sm:max-w-3xl`) {
		t.Error("dialog element should not carry width classes; they belong on the wrapper child")
	}

	// The slider is wrapped in a single div, which is the sole
	// direct child of the dialog. Verify the slider element is
	// present inside the dialog so the image-compare library has
	// something to mount.
	if !strings.Contains(html, `id="weight-compare-slider"`) {
		t.Error("expected the slider container to be rendered inside the dialog")
	}

	// The slider is rendered with the image-compare class so the
	// library picks it up on mount.
	if !strings.Contains(html, `class="image-compare"`) {
		t.Error("expected the slider container to carry the image-compare class")
	}

	// The X close button is no longer rendered as an aria-label in
	// the dialog header — the dialog closes via the backdrop click
	// handler instead.
	if strings.Contains(html, `aria-label="Close"`) {
		t.Error("X close button should be removed from the header")
	}
}

// TestCompareModal_OmitsDeltaWhenEmpty ensures the "no change" label
// is not rendered when the weight delta is zero. An empty deltaText
// should produce a header that ends with the after-date, not a trailing
// "· no change" suffix.
func TestCompareModal_OmitsDeltaWhenEmpty(t *testing.T) {
	html := renderToString(t, CompareModal(
		"01 Jan 2026", "80.0 kg", "https://cdn.example.com/before.jpg",
		"15 Jun 2026", "80.0 kg", "https://cdn.example.com/after.jpg",
		"", // no delta
	))
	if strings.Contains(html, "no change") {
		t.Error("expected no 'no change' text in modal")
	}
}

// TestCompareBar_HiddenByDefault ensures the bar starts hidden so the
// table is unobstructed on initial page load.
func TestCompareBar_HiddenByDefault(t *testing.T) {
	html := renderToString(t, CompareBar())
	if !strings.Contains(html, `id="weight-compare-bar"`) {
		t.Fatal("expected compare bar in DOM")
	}
	if !strings.Contains(html, `class="hidden card p-3 mb-4`) {
		t.Error("expected compare bar to start hidden and rendered as an inline card")
	}
	if strings.Contains(html, `fixed bottom-0`) {
		t.Error("expected compare bar to no longer be position:fixed")
	}
	if !strings.Contains(html, `id="weight-compare-go"`) {
		t.Error("expected compare button in bar")
	}
	if !strings.Contains(html, `disabled`) {
		t.Error("expected compare button to start disabled")
	}
	if !strings.Contains(html, `id="weight-compare-id-a"`) {
		t.Error("expected hidden input for the first id in bar")
	}
	if !strings.Contains(html, `id="weight-compare-id-b"`) {
		t.Error("expected hidden input for the second id in bar")
	}
	if !strings.Contains(html, `hx-get="/weight/compare-modal"`) {
		t.Error("expected compare button to target compare-modal endpoint")
	}
}

// TestWeightEntry_FormattedDateLong covers the helper used to format
// the long date in the comparison modal label.
func TestWeightEntry_FormattedDateLong(t *testing.T) {
	day := time.Date(2026, 1, 9, 8, 0, 0, 0, time.UTC)
	e := models.WeightEntry{CreatedAt: day}
	if got, want := e.FormattedDateLong(), "09 Jan 2026"; got != want {
		t.Errorf("FormattedDateLong = %q, want %q", got, want)
	}
}

// TestWeightPage_RendersExportLinkWithEntries asserts that the export
// link is present in the page header when the user has at least one
// entry. The link points at /weight/export and carries id="weight-export-link"
// so the inline script can wire up the confirm dialog.
func TestWeightPage_RendersExportLinkWithEntries(t *testing.T) {
	day := time.Date(2026, 1, 9, 8, 0, 0, 0, time.UTC)
	entries := []models.WeightEntry{
		{ID: "w1", Weight: 80, CreatedAt: day},
	}
	html := renderToString(t, WeightPage(entries, "Test User", true, false, nil, "kg"))

	if !strings.Contains(html, `id="weight-export-link"`) {
		t.Error("expected export link in header")
	}
	if !strings.Contains(html, `href="/weight/export"`) {
		t.Error("expected export link to point at /weight/export")
	}
	if !strings.Contains(html, `download`) {
		t.Error("expected export link to carry the download attribute")
	}
}

// TestWeightPage_HidesExportLinkWhenEmpty ensures users with no
// entries don't see an export button — exporting an empty zip would
// just be a confusing no-op.
func TestWeightPage_HidesExportLinkWhenEmpty(t *testing.T) {
	html := renderToString(t, WeightPage(nil, "Test User", true, false, nil, "kg"))
	if strings.Contains(html, `id="weight-export-link"`) {
		t.Error("expected no export link when the user has no entries")
	}
	// The page is in the empty-state branch, so the regular
	// table/CompareBar/WeightExportScript are not rendered.
	if strings.Contains(html, `id="weight-compare-bar"`) {
		t.Error("expected no compare bar in empty state")
	}
}
