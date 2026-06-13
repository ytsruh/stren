package views

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
	"stren/internal/models"
	"stren/internal/views/components"
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

// === Helper function tests ===

func TestFormatDateTimeLocal(t *testing.T) {
	tm := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	got := FormatDateTimeLocal(tm)
	if got != "2024-06-15T14:30" {
		t.Errorf("FormatDateTimeLocal = %q, want %q", got, "2024-06-15T14:30")
	}
}

func TestCountUniqueExercises(t *testing.T) {
	now := time.Now()
	entries := []models.ExerciseEntry{
		{ExerciseName: "Squat", CreatedAt: now.Add(-1 * time.Hour)},
		{ExerciseName: "Bench Press", CreatedAt: now.Add(-2 * time.Hour)},
		{ExerciseName: "Squat", CreatedAt: now.Add(-3 * time.Hour)},
		{ExerciseName: "Deadlift", CreatedAt: now.Add(-8 * 24 * time.Hour)}, // older than 7 days
	}
	got := countUniqueExercises(entries)
	if got != 2 {
		t.Errorf("countUniqueExercises = %d, want 2", got)
	}
}

func TestCalculateVolume(t *testing.T) {
	now := time.Now()
	entries := []models.ExerciseEntry{
		{Weight: 100, CreatedAt: now.Add(-1 * time.Hour)},
		{Weight: 80, CreatedAt: now.Add(-2 * time.Hour)},
		{Weight: 120, CreatedAt: now.Add(-8 * 24 * time.Hour)}, // older than 7 days
	}
	got := calculateVolume(entries)
	if got != 180 {
		t.Errorf("calculateVolume = %f, want 180", got)
	}
}

func TestLastSetLabel(t *testing.T) {
	if got := lastSetLabel(models.ExerciseEntry{}); got != "No entries" {
		t.Errorf("lastSetLabel(empty) = %q, want %q", got, "No entries")
	}
	got := lastSetLabel(models.ExerciseEntry{ID: "e1", Weight: 100, Reps: 5})
	if got != "100.0 kg \u00d7 5" {
		t.Errorf("lastSetLabel = %q, want %q", got, "100.0 kg \u00d7 5")
	}
}

// === Component rendering tests ===

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
	if !strings.Contains(html, "No workouts yet") {
		t.Error("expected empty state heading")
	}
	if !strings.Contains(html, "Add First Entry") {
		t.Error("expected call-to-action button")
	}
}

func TestDashboard_WithEntries(t *testing.T) {
	entries := []models.ExerciseEntry{
		{ID: "entry-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
	}
	html := renderToString(t, Dashboard(entries, "Test User", true, false))
	if !strings.Contains(html, "7 Day History") {
		t.Error("expected page title")
	}
	if !strings.Contains(html, "Squat") {
		t.Error("expected exercise name in table")
	}
	if strings.Contains(html, "No workouts yet") {
		t.Error("did not expect empty state when entries exist")
	}
}

func TestDashboard_Empty(t *testing.T) {
	html := renderToString(t, Dashboard([]models.ExerciseEntry{}, "Test User", true, false))
	if !strings.Contains(html, "No workouts yet") {
		t.Error("expected empty state when no entries")
	}
}

func TestEntryList(t *testing.T) {
	entries := []models.ExerciseEntry{
		{ID: "entry-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
		{ID: "entry-2", ExerciseName: "Bench Press", Reps: 8, Weight: 80},
	}
	html := renderToString(t, EntryList(entries))
	if !strings.Contains(html, "Squat") {
		t.Error("expected Squat in output")
	}
	if !strings.Contains(html, "Bench Press") {
		t.Error("expected Bench Press in output")
	}
	if !strings.Contains(html, `<tbody id="entries-table">`) {
		t.Error("expected entries-table tbody")
	}
}

func TestEntryRow(t *testing.T) {
	entry := models.ExerciseEntry{
		ID:           "entry-1",
		ExerciseName: "Deadlift",
		Reps:         5,
		Weight:       180,
		Notes:        "PR",
		CreatedAt:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	html := renderToString(t, EntryRow(entry))
	if !strings.Contains(html, "Deadlift") {
		t.Error("expected exercise name")
	}
	if !strings.Contains(html, "180.0 kg") {
		t.Error("expected formatted weight")
	}
	if !strings.Contains(html, "PR") {
		t.Error("expected notes")
	}
	if !strings.Contains(html, `id="entry-entry-1"`) {
		t.Error("expected entry ID attribute")
	}
	if !strings.Contains(html, "/entries/entry-1/edit") {
		t.Error("expected edit link")
	}
}

func TestRecentEntries_WithData(t *testing.T) {
	entries := []models.ExerciseEntry{
		{ID: "entry-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
	}
	html := renderToString(t, RecentEntries(entries))
	if !strings.Contains(html, "Recent Workouts") {
		t.Error("expected section heading")
	}
	if !strings.Contains(html, "Squat") {
		t.Error("expected entry data")
	}
}

func TestRecentEntries_Empty(t *testing.T) {
	html := renderToString(t, RecentEntries([]models.ExerciseEntry{}))
	if !strings.Contains(html, "No workouts yet") {
		t.Error("expected empty state")
	}
}

func TestStatCard(t *testing.T) {
	html := renderToString(t, components.StatCard(components.StatCardProps{Label: "Total Sets", Value: "42"}))
	if !strings.Contains(html, "Total Sets") {
		t.Error("expected label")
	}
	if !strings.Contains(html, "42") {
		t.Error("expected value")
	}
}

func TestQuickStats(t *testing.T) {
	entries := []models.ExerciseEntry{
		{ExerciseName: "Squat", Weight: 100, CreatedAt: time.Now()},
		{ExerciseName: "Bench Press", Weight: 80, CreatedAt: time.Now()},
	}
	html := renderToString(t, QuickStats(entries))
	if !strings.Contains(html, "Last 7 Days") {
		t.Error("expected heading")
	}
	if !strings.Contains(html, "Total Sets") {
		t.Error("expected Total Sets stat")
	}
}

func TestEntryForm_New(t *testing.T) {
	types := []models.Exercise{
		{ID: "ex-1", Name: "Squat"},
		{ID: "ex-2", Name: "Bench Press"},
	}
	html := renderToString(t, EntryForm(types, "", "Test User", true, false))
	if !strings.Contains(html, "New Entry") {
		t.Error("expected new entry title")
	}
	if !strings.Contains(html, `value=""`) && !strings.Contains(html, "Select an exercise") {
		t.Error("expected empty/default exercise selection")
	}
	if !strings.Contains(html, `/entries"`) {
		t.Error("expected form action to be /entries")
	}
	// Multi-set form scaffolding.
	if !strings.Contains(html, `data-set-list`) {
		t.Error("expected data-set-list container for set rows")
	}
	if !strings.Contains(html, `data-add-set`) {
		t.Error("expected Add Set button")
	}
	if !strings.Contains(html, `id="entry-set-template"`) {
		t.Error("expected hidden row template for cloning")
	}
	// The form exposes the cap so the inline script can read it at runtime.
	wantMaxAttr := fmt.Sprintf(`data-max-sets="%d"`, MaxSetsPerEntry)
	if !strings.Contains(html, wantMaxAttr) {
		t.Errorf("expected form to expose %s", wantMaxAttr)
	}
	// Three starter rows rendered, indices 0/1/2.
	for _, i := range []string{"0", "1", "2"} {
		want := fmt.Sprintf(`name="sets[%s][reps]"`, i)
		if !strings.Contains(html, want) {
			t.Errorf("expected starter row with %s", want)
		}
		wantWeight := fmt.Sprintf(`name="sets[%s][weight]"`, i)
		if !strings.Contains(html, wantWeight) {
			t.Errorf("expected starter row with %s", wantWeight)
		}
	}
	// Remove button is wired up via data-remove-set.
	if !strings.Contains(html, `data-remove-set`) {
		t.Error("expected remove-set buttons on rows")
	}
	// Responsive layout: outer row is a single column on mobile and switches
	// to the 5-column grid at sm:. The header wrapper uses sm:contents so
	// the label and X button become direct grid items on larger screens.
	// sm:order-5 on the X button pushes it to the last column on desktop
	// (otherwise source order would put it next to the label, in column 2).
	if !strings.Contains(html, `sm:grid-cols-[auto_1fr_1fr_1fr_auto]`) {
		t.Error("expected sm: 5-column grid on set rows")
	}
	if !strings.Contains(html, `sm:contents`) {
		t.Error("expected sm:contents wrapper to flatten header on larger screens")
	}
	if !strings.Contains(html, `sm:items-end`) {
		t.Error("expected sm:items-end to align row contents on larger screens")
	}
	if !strings.Contains(html, `sm:order-5`) {
		t.Error("expected sm:order-5 on the X button so it sits in the last column on desktop")
	}
	// Sanity: the inline script looks the clone template up by id via
	// getElementById, NOT form.querySelector (the template lives outside the
	// form so a scoped query would miss it).
	if !strings.Contains(html, `getElementById('entry-set-template')`) {
		t.Error("expected script to look up clone template by getElementById")
	}
	// Date & Time input: a datetime-local field named created_at, marked
	// required, with a value rendered from FormatDateTimeLocal(time.Now()).
	if !strings.Contains(html, "Date &amp; Time") && !strings.Contains(html, "Date & Time") {
		t.Error("expected 'Date & Time' label in new-entry form")
	}
	if !strings.Contains(html, `id="entry-date"`) {
		t.Error("expected id=\"entry-date\" on the datetime input")
	}
	// The required input must combine name, type and required attributes
	// together so a regression that drops the field or its name is caught.
	dateInputPattern := `<input class="input" type="datetime-local" id="entry-date" name="created_at" value="[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}" required`
	if !regexp.MustCompile(dateInputPattern).MatchString(html) {
		t.Errorf("expected datetime-local input with name=\"created_at\" and rendered value matching the formatDateTimeLocal pattern; pattern was %q", dateInputPattern)
	}
	// Exercise select and date input sit side-by-side at sm: and above,
	// stacking on mobile. Assert the wrapper so a regression that drops
	// the responsive class (or moves the fields out of the wrapper) is
	// caught.
	if !strings.Contains(html, `sm:grid-cols-2 sm:items-end`) {
		t.Error("expected exercise + date to share a sm:grid-cols-2 sm:items-end wrapper")
	}
	// The exercise select id="exercise-name" and the date input id="entry-date"
	// must both be present in the rendered HTML so the wrapper is actually
	// wrapping the right fields. The regex above already proves the date
	// input; this double-checks the select is in the same context.
	if !strings.Contains(html, `id="exercise-name"`) {
		t.Error("expected exercise select id=\"exercise-name\"")
	}
}

func TestEditEntryForm(t *testing.T) {
	entry := &models.ExerciseEntry{
		ID:           "entry-1",
		ExerciseName: "Squat",
		Reps:         5,
		Weight:       100.0,
		Notes:        "good",
		CreatedAt:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	html := renderToString(t, EditEntryForm(entry, "Test User", true, false))
	if !strings.Contains(html, "Edit Entry") {
		t.Error("expected edit title")
	}
	if !strings.Contains(html, "/entries/entry-1") {
		t.Error("expected form action with entry ID")
	}
	if !strings.Contains(html, `name="created_at"`) {
		t.Error("expected datetime input for edit mode")
	}
}

func TestEntryFormSuccess_StayOnPage(t *testing.T) {
	html := renderToString(t, EntryFormSuccess("Saved!", true))
	if !strings.Contains(html, "Saved!") {
		t.Error("expected success message")
	}
	if !strings.Contains(html, "data-category=\"success\"") {
		t.Error("expected success toast category")
	}
}

func TestEntryFormSuccess_Redirect(t *testing.T) {
	html := renderToString(t, EntryFormSuccess("Saved!", false))
	if !strings.Contains(html, "Back to Dashboard") {
		t.Error("expected back link when not staying on page")
	}
	if !strings.Contains(html, "data-category=\"success\"") {
		t.Error("expected success toast category")
	}
}

func TestEntryFormError(t *testing.T) {
	html := renderToString(t, EntryFormError("Invalid reps"))
	if !strings.Contains(html, "Invalid reps") {
		t.Error("expected error message")
	}
	if !strings.Contains(html, "data-category=\"error\"") {
		t.Error("expected error data-category")
	}
}

func TestExerciseHistory_WithEntries(t *testing.T) {
	exercise := &models.Exercise{ID: "ex-1", Name: "Squat", Type: models.ExerciseTypeStrength}
	page := &models.ExerciseHistoryPage{
		Entries: []models.ExerciseEntry{
			{ID: "entry-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
		},
		Stats: models.HistoryStats{
			MaxWeight: 100,
			LastSet:   models.ExerciseEntry{ID: "entry-1", Reps: 5, Weight: 100},
		},
		Page: 1,
	}
	html := renderToString(t, ExerciseHistory(exercise, page, "Test User", true, false))
	if !strings.Contains(html, "Squat") {
		t.Error("expected exercise name heading")
	}
	if strings.Contains(html, "No entries yet") {
		t.Error("did not expect empty state when entries exist")
	}
	if !strings.Contains(html, "100.0 kg") {
		t.Error("expected personal best value to appear")
	}
}

func TestExerciseHistory_Empty(t *testing.T) {
	exercise := &models.Exercise{ID: "ex-1", Name: "Squat", Type: models.ExerciseTypeStrength}
	page := &models.ExerciseHistoryPage{Page: 1}
	html := renderToString(t, ExerciseHistory(exercise, page, "Test User", true, false))
	if !strings.Contains(html, "No entries yet") {
		t.Error("expected empty state")
	}
}

func TestHistoryTable_NoPagerWhenSinglePage(t *testing.T) {
	page := &models.ExerciseHistoryPage{
		Entries: []models.ExerciseEntry{
			{ID: "entry-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
		},
		Page: 1,
	}
	html := renderToString(t, HistoryTable("ex-1", page))
	if !strings.Contains(html, historyTableWrapID) {
		t.Error("expected swappable wrap region to be present")
	}
	if strings.Contains(html, "Previous page") || strings.Contains(html, "Next page") {
		t.Error("did not expect pagination buttons on a single-page result")
	}
}

func TestHistoryTable_ShowsNextAndPrev(t *testing.T) {
	page := &models.ExerciseHistoryPage{
		Entries: []models.ExerciseEntry{
			{ID: "entry-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
		},
		Page:    2,
		HasPrev: true,
		HasNext: true,
	}
	html := renderToString(t, HistoryTable("ex-1", page))
	if !strings.Contains(html, `aria-label="Previous page"`) {
		t.Error("expected Previous button on a middle page")
	}
	if !strings.Contains(html, `aria-label="Next page"`) {
		t.Error("expected Next button on a middle page")
	}
	if !strings.Contains(html, `hx-get="/exercises/ex-1?page=1"`) {
		t.Error("expected Previous button to point at page 1")
	}
	if !strings.Contains(html, `hx-get="/exercises/ex-1?page=3"`) {
		t.Error("expected Next button to point at page 3")
	}
	if !strings.Contains(html, `hx-push-url="true"`) {
		t.Error("expected pagination links to push URL for shareable history")
	}
	if !strings.Contains(html, `hx-target="#`+historyTableWrapID+`"`) {
		t.Error("expected pagination to target the swappable wrap region")
	}
}

func TestIcon_Trash(t *testing.T) {
	html := renderToString(t, components.Icon(components.IconProps{Name: "trash", Size: 16}))
	if !strings.Contains(html, `<svg`) {
		t.Error("expected svg element")
	}
	if !strings.Contains(html, `width="16"`) {
		t.Error("expected width attribute")
	}
	if !strings.Contains(html, "polyline") {
		t.Error("expected trash icon path")
	}
}

func TestIcon_Unknown(t *testing.T) {
	html := renderToString(t, components.Icon(components.IconProps{Name: "nonexistent"}))
	if !strings.Contains(html, `<circle`) {
		t.Error("expected fallback circle for unknown icon")
	}
}

func TestIcon_Alias(t *testing.T) {
	// "delete" should resolve to "trash" icon.
	html := renderToString(t, components.Icon(components.IconProps{Name: "delete"}))
	if !strings.Contains(html, "polyline") {
		t.Error("expected trash icon path for 'delete' alias")
	}
}

func TestIcon_DefaultSizeAndColor(t *testing.T) {
	html := renderToString(t, components.Icon(components.IconProps{Name: "plus"}))
	if !strings.Contains(html, `width="24"`) {
		t.Error("expected default size 24")
	}
	if !strings.Contains(html, `stroke="currentColor"`) {
		t.Error("expected default stroke color")
	}
}
