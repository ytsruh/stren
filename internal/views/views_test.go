package views

import (
	"bytes"
	"context"
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
	got := formatDateTimeLocal(tm)
	if got != "2024-06-15T14:30" {
		t.Errorf("formatDateTimeLocal = %q, want %q", got, "2024-06-15T14:30")
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

func TestMaxWeight(t *testing.T) {
	tests := []struct {
		name     string
		entries  []models.ExerciseEntry
		expected float64
	}{
		{
			name:     "empty",
			entries:  []models.ExerciseEntry{},
			expected: 0,
		},
		{
			name:     "single",
			entries:  []models.ExerciseEntry{{Weight: 100}},
			expected: 100,
		},
		{
			name:     "multiple",
			entries:  []models.ExerciseEntry{{Weight: 80}, {Weight: 120}, {Weight: 100}},
			expected: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maxWeight(tt.entries)
			if got != tt.expected {
				t.Errorf("maxWeight = %f, want %f", got, tt.expected)
			}
		})
	}
}

func TestGetLastSet(t *testing.T) {
	if got := getLastSet([]models.ExerciseEntry{}); got != "No entries" {
		t.Errorf("getLastSet(empty) = %q, want %q", got, "No entries")
	}
	entries := []models.ExerciseEntry{
		{Weight: 100, Reps: 5, CreatedAt: time.Now().Add(-1 * time.Hour)},
		{Weight: 80, Reps: 8, CreatedAt: time.Now().Add(-2 * time.Hour)},
	}
	got := getLastSet(entries)
	if got != "100.0 kg \u00d7 5" {
		t.Errorf("getLastSet = %q, want %q", got, "100.0 kg \u00d7 5")
	}
}

// === Component rendering tests ===

func TestToast_Error(t *testing.T) {
	html := renderToString(t, Toast("Bad input", true))
	if !strings.Contains(html, "Bad input") {
		t.Error("expected toast message in output")
	}
	if !strings.Contains(html, "alert-destructive") {
		t.Error("expected error CSS class in output")
	}
}

func TestToast_Success(t *testing.T) {
	html := renderToString(t, Toast("Saved!", false))
	if !strings.Contains(html, "Saved!") {
		t.Error("expected toast message in output")
	}
	if !strings.Contains(html, "alert") {
		t.Error("expected success CSS class in output")
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
		{ID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
	}
	html := renderToString(t, Dashboard(entries, "Test User", true, false))
	if !strings.Contains(html, "Workout History") {
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
		{ID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
		{ID: 2, ExerciseName: "Bench Press", Reps: 8, Weight: 80},
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
		ID:           1,
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
	if !strings.Contains(html, `id="entry-1"`) {
		t.Error("expected entry ID attribute")
	}
	if !strings.Contains(html, "/entries/1/edit") {
		t.Error("expected edit link")
	}
}

func TestDeletedRow(t *testing.T) {
	html := renderToString(t, DeletedRow())
	if !strings.Contains(html, `class="fade-out"`) {
		t.Error("expected fade-out class")
	}
}

func TestRecentEntries_WithData(t *testing.T) {
	entries := []models.ExerciseEntry{
		{ID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
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
	types := []models.ExerciseType{
		{ID: 1, Name: "Squat"},
		{ID: 2, Name: "Bench Press"},
	}
	html := renderToString(t, EntryForm(types, "Test User", true, false))
	if !strings.Contains(html, "New Entry") {
		t.Error("expected new entry title")
	}
	if !strings.Contains(html, `value=""`) && !strings.Contains(html, "Select an exercise") {
		t.Error("expected empty/default exercise selection")
	}
	if !strings.Contains(html, `/entries"`) {
		t.Error("expected form action to be /entries")
	}
}

func TestEditEntryForm(t *testing.T) {
	entry := &models.ExerciseEntry{
		ID:           1,
		ExerciseName: "Squat",
		Reps:         5,
		Weight:       100.0,
		Notes:        "good",
		CreatedAt:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	types := []models.ExerciseType{
		{ID: 1, Name: "Squat"},
		{ID: 2, Name: "Bench Press"},
	}
	html := renderToString(t, EditEntryForm(entry, types, "Test User", true, false))
	if !strings.Contains(html, "Edit Entry") {
		t.Error("expected edit title")
	}
	if !strings.Contains(html, "/entries/1") {
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
	if !strings.Contains(html, "reset()") {
		t.Error("expected form reset script for stay-on-page")
	}
}

func TestEntryFormSuccess_Redirect(t *testing.T) {
	html := renderToString(t, EntryFormSuccess("Saved!", false))
	if !strings.Contains(html, "Back to Dashboard") {
		t.Error("expected back link when not staying on page")
	}
	if strings.Contains(html, "reset()") {
		t.Error("did not expect form reset script for redirect")
	}
}

func TestEntryFormError(t *testing.T) {
	html := renderToString(t, EntryFormError("Invalid reps"))
	if !strings.Contains(html, "Invalid reps") {
		t.Error("expected error message")
	}
	if !strings.Contains(html, "alert-destructive") {
		t.Error("expected error CSS class")
	}
}

func TestExerciseHistory_WithEntries(t *testing.T) {
	entries := []models.ExerciseEntry{
		{ID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
	}
	html := renderToString(t, ExerciseHistory("Squat", entries, "Test User", true, false))
	if !strings.Contains(html, "Squat") {
		t.Error("expected exercise name heading")
	}
	if !strings.Contains(html, "Stats") {
		t.Error("expected stats section")
	}
	if !strings.Contains(html, "Personal Best") {
		t.Error("expected personal best stat")
	}
	if strings.Contains(html, "No entries yet") {
		t.Error("did not expect empty state when entries exist")
	}
}

func TestExerciseHistory_Empty(t *testing.T) {
	html := renderToString(t, ExerciseHistory("Squat", []models.ExerciseEntry{}, "Test User", true, false))
	if !strings.Contains(html, "No entries yet") {
		t.Error("expected empty state")
	}
	if strings.Contains(html, "Personal Best") {
		t.Error("did not expect stats when empty")
	}
}

func TestExerciseStats(t *testing.T) {
	entries := []models.ExerciseEntry{
		{Weight: 120, Reps: 3},
		{Weight: 100, Reps: 5},
	}
	html := renderToString(t, ExerciseStats(entries))
	if !strings.Contains(html, "120.0 kg") {
		t.Error("expected personal best weight")
	}
	if !strings.Contains(html, "120.0 kg \u00d7 3") {
		t.Errorf("expected last set string, got: %s", html)
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
