package goals

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

// TestGoalCard_ActiveRendersTitleAndActions asserts the basic
// happy-path render of an active goal: title visible (no
// strikethrough), action buttons present, no completed chip, no
// delete button on the list.
func TestGoalCard_ActiveRendersTitleAndActions(t *testing.T) {
	g := models.Goal{
		ID:    "g1",
		Title: "Run a 5k",
	}
	html := renderToString(t, GoalCard(g, true, false))
	if !strings.Contains(html, "Run a 5k") {
		t.Error("expected title in card")
	}
	if strings.Contains(html, "line-through") {
		t.Error("active goal should not render line-through")
	}
	if !strings.Contains(html, "Mark Complete") && !strings.Contains(html, `aria-label="Mark Complete"`) {
		t.Error("expected Mark Complete button on active card")
	}
	if strings.Contains(html, "Reopen") {
		t.Error("Reopen should not be present on active card")
	}
	if !strings.Contains(html, `hx-post="/goals/g1/complete"`) {
		t.Error("expected Mark Complete to POST to /goals/g1/complete")
	}
	if !strings.Contains(html, `id="goal-g1"`) {
		t.Error("expected card id goal-g1 for OOB targeting")
	}
	if !strings.Contains(html, `hx-swap-oob="true"`) {
		t.Error("expected OOB swap attribute on the card")
	}
	// No delete button on the list view — delete only on the edit
	// form. The card uses hx-delete in one place: the embedded
	// Mark-Complete button would never trigger a delete, so the
	// only way "hx-delete" could leak into the card is via a
	// dedicated delete button.
	if strings.Contains(html, "hx-delete") {
		t.Error("list-view card should not render a delete button")
	}
}

// TestGoalCard_CompletedRendersStrikethroughAndReopen asserts the
// completed-state visuals: title gets a line-through, the card
// gets a dimmed appearance, and the action button switches to
// Reopen.
func TestGoalCard_CompletedRendersStrikethroughAndReopen(t *testing.T) {
	when := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	g := models.Goal{
		ID:          "g1",
		Title:       "Run a 5k",
		CompletedAt: &when,
	}
	html := renderToString(t, GoalCard(g, true, false))
	if !strings.Contains(html, "line-through") {
		t.Error("completed goal title should render line-through")
	}
	if !strings.Contains(html, `aria-label="Reopen"`) {
		t.Error("expected Reopen button on completed card")
	}
	if strings.Contains(html, "Mark Complete") {
		t.Error("Mark Complete should not be present on completed card")
	}
	if !strings.Contains(html, `hx-post="/goals/g1/reopen"`) {
		t.Error("expected Reopen to POST to /goals/g1/reopen")
	}
	if !strings.Contains(html, "opacity-60") {
		t.Error("expected dimmed appearance class on completed card")
	}
	// Completed chip surfaces the completed date so the user
	// doesn't have to remember when they ticked the box.
	if !strings.Contains(html, "Done 09 Jan 2026") {
		t.Error("expected 'Done 09 Jan 2026' chip on completed card")
	}
}

// TestGoalCard_InlineDateChips asserts the start / target / end
// dates render as inline text (not badges) so the row stays
// compact. Each chip is omitted when the date is nil.
func TestGoalCard_InlineDateChips(t *testing.T) {
	t.Run("no dates", func(t *testing.T) {
		g := models.Goal{ID: "g1", Title: "x"}
		html := renderToString(t, GoalCard(g, true, false))
		if strings.Contains(html, "Started") || strings.Contains(html, "Target") || strings.Contains(html, "Ended") {
			t.Error("expected no date chips when no dates are set")
		}
	})
	t.Run("all dates", func(t *testing.T) {
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		target := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
		g := models.Goal{ID: "g1", Title: "x", StartDate: &start, TargetDate: &target, EndDate: &end}
		html := renderToString(t, GoalCard(g, true, false))
		// shortGoalDate uses "02 Jan" format.
		for _, want := range []string{"Started 01 Jan", "Target 01 Jul", "Ended 31 Dec"} {
			if !strings.Contains(html, want) {
				t.Errorf("expected %q in card, got: %s", want, html)
			}
		}
	})
}

// TestGoalCard_DueInDaysHint asserts the "due in N days" hint
// renders on active goals with a future target. Past or nil
// targets do not render the hint.
func TestGoalCard_DueInDaysHint(t *testing.T) {
	t.Run("future target shows hint", func(t *testing.T) {
		now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
		future := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
		// Swap the package clock so the test is deterministic.
		orig := currentTime
		currentTime = func() timeT { return now }
		t.Cleanup(func() { currentTime = orig })

		g := models.Goal{ID: "g1", Title: "x", TargetDate: &future}
		html := renderToString(t, GoalCard(g, true, false))
		if !strings.Contains(html, "5d") {
			t.Errorf("expected '5d' hint, got: %s", html)
		}
	})
	t.Run("today target shows 'Today'", func(t *testing.T) {
		now := time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC)
		target := time.Date(2026, 6, 15, 23, 0, 0, 0, time.UTC)
		orig := currentTime
		currentTime = func() timeT { return now }
		t.Cleanup(func() { currentTime = orig })

		g := models.Goal{ID: "g1", Title: "x", TargetDate: &target}
		html := renderToString(t, GoalCard(g, true, false))
		if !strings.Contains(html, "Today") {
			t.Errorf("expected 'Today' hint, got: %s", html)
		}
	})
	t.Run("completed goal hides the hint", func(t *testing.T) {
		now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
		future := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
		orig := currentTime
		currentTime = func() timeT { return now }
		t.Cleanup(func() { currentTime = orig })

		completed := now
		g := models.Goal{ID: "g1", Title: "x", TargetDate: &future, CompletedAt: &completed}
		html := renderToString(t, GoalCard(g, true, false))
		if strings.Contains(html, "5d") {
			t.Errorf("completed goal should not show 'due in' hint, got: %s", html)
		}
	})
}

// TestGoalPage_EmptyState asserts the empty-state copy and CTA
// appear when the user has no goals.
func TestGoalPage_EmptyState(t *testing.T) {
	html := renderToString(t, GoalsPage(nil, "Test User", true, false))
	if !strings.Contains(html, "No goals yet") {
		t.Error("expected empty-state heading")
	}
	if !strings.Contains(html, `href="/goals/new"`) {
		t.Error("expected Add Your First Goal CTA")
	}
	// Empty state must not render any cards.
	if strings.Contains(html, `id="goal-`) {
		t.Error("empty state must not render any goal cards")
	}
}

// TestGoalPage_RendersActiveAndCompleted asserts the list page
// splits the goals into two sections (Active + Completed), renders
// each card under the right section, mounts the confetti +
// event-handler scripts, and wraps each section in an OOB-swappable
// container with the right id.
func TestGoalPage_RendersActiveAndCompleted(t *testing.T) {
	when := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	active := models.Goal{ID: "g-active", Title: "Active goal"}
	completed := models.Goal{ID: "g-done", Title: "Completed goal", CompletedAt: &when}
	html := renderToString(t, GoalsPage([]models.Goal{active, completed}, "Test User", true, false))

	// Both section wrappers present, OOB-swappable, and carry
	// the section title.
	if !strings.Contains(html, `id="active-goals-section"`) {
		t.Error("expected active-goals-section wrapper id")
	}
	if !strings.Contains(html, `id="completed-goals-section"`) {
		t.Error("expected completed-goals-section wrapper id")
	}
	if !strings.Contains(html, "Active goals") {
		t.Error("expected Active goals section header")
	}
	if !strings.Contains(html, "Completed goals") {
		t.Error("expected Completed goals section header")
	}
	// The OOB attribute is harmless on initial render; htmx
	// ignores it when there is no parent htmx request.
	if !strings.Contains(html, `hx-swap-oob="true"`) {
		t.Error("expected hx-swap-oob=\"true\" on the section wrappers")
	}

	// Both card ids present.
	if !strings.Contains(html, `id="goal-g-active"`) {
		t.Error("expected active card id")
	}
	if !strings.Contains(html, `id="goal-g-done"`) {
		t.Error("expected completed card id")
	}

	// The completed section uses Basecoat's accordion pattern
	// (native <details>) — assert the wrapper is present so the
	// Basecoat JS picks it up.
	if !strings.Contains(html, `class="accordion"`) {
		t.Error("expected completed section wrapped in Basecoat accordion")
	}
	if !strings.Contains(html, "<details>") {
		t.Error("expected completed section to use a <details> element")
	}

	// Completed accordion is closed by default (no `open`
	// attribute) so the page stays compact on first render.
	if strings.Contains(html, "<details open>") {
		t.Error("expected completed accordion to be closed by default")
	}

	// Confetti + event-handler scripts are still mounted.
	if !strings.Contains(html, "goalCompleted") {
		t.Error("expected confetti listener to be mounted on the page")
	}
	if !strings.Contains(html, "showGoalToast") {
		t.Error("expected goal event listener script")
	}
}

// TestGoalPage_OnlyActiveHidesCompleted asserts the completed
// section wrapper is still in the DOM (so the OOB swap target
// exists) but is hidden via class="hidden" when there are no
// completed goals. The user sees only the active section and
// the Add Goal button.
func TestGoalPage_OnlyActiveHidesCompleted(t *testing.T) {
	active := models.Goal{ID: "g-active", Title: "Run a 5k"}
	html := renderToString(t, GoalsPage([]models.Goal{active}, "Test User", true, false))

	if !strings.Contains(html, "Active goals") {
		t.Error("expected Active goals header when there is an active goal")
	}
	// The completed section is in the DOM (so OOB can swap into
	// it) but visually hidden. The attribute order is id +
	// hx-swap-oob + class, so we check the wrapper starts with
	// the right id and contains the hidden class.
	if !strings.Contains(html, `id="completed-goals-section"`) {
		t.Error("expected completed-goals-section wrapper to be in the DOM")
	}
	if !strings.Contains(html, `<div id="completed-goals-section"`) || !strings.Contains(html, `class="hidden"`) {
		t.Error("expected completed wrapper to carry class=\"hidden\" when empty")
	}
	if strings.Contains(html, "No active goals") {
		t.Error("expected no empty-active hint when there is an active goal")
	}
	// No completed card ids should be present.
	if strings.Contains(html, `id="goal-g-done"`) {
		t.Error("expected no completed card when there are no completed goals")
	}
}

// TestGoalPage_OnlyCompletedShowsEmptyActiveHint asserts the
// Active section still renders (with the in-section "no active
// goals" hint) when the user has completed goals but no active
// ones. Discoverability > hiding the section.
func TestGoalPage_OnlyCompletedShowsEmptyActiveHint(t *testing.T) {
	when := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	completed := models.Goal{ID: "g-done", Title: "Squat 100kg", CompletedAt: &when}
	html := renderToString(t, GoalsPage([]models.Goal{completed}, "Test User", true, false))

	// Active section is still present.
	if !strings.Contains(html, "Active goals") {
		t.Error("expected Active goals header even when empty")
	}
	// Empty-state hint surfaces the Add CTA.
	if !strings.Contains(html, "No active goals") {
		t.Error("expected empty-state hint inside the Active section")
	}
	if !strings.Contains(html, `href="/goals/new"`) {
		t.Error("expected Add CTA link inside the empty Active hint")
	}
	// No active cards rendered (the only `id="goal-` should be
	// the completed card).
	if strings.Contains(html, `id="goal-g-active"`) {
		t.Error("expected no active goal card id (g-active) when active is empty")
	}
	// Completed section is still rendered and visible (not hidden).
	if !strings.Contains(html, "Completed goals") {
		t.Error("expected Completed goals header")
	}
	if !strings.Contains(html, `id="goal-g-done"`) {
		t.Error("expected completed card id")
	}
	if strings.Contains(html, `<div id="completed-goals-section" hx-swap-oob="true" class="hidden"`) {
		t.Error("expected completed wrapper to NOT be hidden when there are completed goals")
	}
}

// TestGoalSectionHeader_RendersTitleAndCount asserts the
// shared section header renders the title, the count separator,
// and the count value, and that the divider line is below.
func TestGoalSectionHeader_RendersTitleAndCount(t *testing.T) {
	html := renderToString(t, GoalsSectionHeader("Active goals", 7))
	if !strings.Contains(html, "Active goals") {
		t.Error("expected section title")
	}
	if !strings.Contains(html, "7") {
		t.Error("expected section count")
	}
	if !strings.Contains(html, "·") {
		t.Error("expected count separator (middle dot)")
	}
	if !strings.Contains(html, "border-t") {
		t.Error("expected divider line beneath the header")
	}
}

// TestGoalSectionHeader_ZeroCount asserts the section header
// still renders with a 0 count (the page itself decides whether
// to skip the section when the count is 0; the header is
// unconditional once the caller chose to render it).
func TestGoalSectionHeader_ZeroCount(t *testing.T) {
	html := renderToString(t, GoalsSectionHeader("Active goals", 0))
	if !strings.Contains(html, ">0<") {
		t.Error("expected '0' count in header")
	}
}

// TestCompletedGoalsSection_UsesDetailsElement asserts the
// completed section wrapper carries the Basecoat
// `class="accordion"` marker (so Basecoat's JS picks it up),
// contains a native <details>, and that the summary text
// combines the title and the count. The goal cards must live
// INSIDE the <details> element so they're hidden when the
// accordion is collapsed.
func TestCompletedGoalsSection_UsesDetailsElement(t *testing.T) {
	when := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	completed := []models.Goal{
		{ID: "g-1", Title: "Squat 100kg", CompletedAt: &when},
		{ID: "g-2", Title: "Run a 5k", CompletedAt: &when},
	}
	html := renderToString(t, CompletedGoalsSection(completed, false))

	if !strings.Contains(html, `id="completed-goals-section"`) {
		t.Error("expected completed-goals-section wrapper id")
	}
	if !strings.Contains(html, `hx-swap-oob="true"`) {
		t.Error("expected hx-swap-oob=\"true\" on the wrapper")
	}
	if !strings.Contains(html, `class="accordion"`) {
		t.Error("expected Basecoat accordion class on the inner wrapper")
	}
	if !strings.Contains(html, "<details>") {
		t.Error("expected a <details> element")
	}
	if !strings.Contains(html, "<summary") {
		t.Error("expected a <summary> element")
	}
	// The accordion is closed by default — no `open` attribute.
	if strings.Contains(html, "<details open>") {
		t.Error("expected accordion to be closed by default")
	}
	// Summary text combines the title and the count.
	if !strings.Contains(html, "Completed goals") {
		t.Error("expected summary text 'Completed goals'")
	}
	if !strings.Contains(html, ">2<") {
		t.Error("expected count '2' in summary")
	}
	// Both card ids must appear inside the accordion (the
	// accordion content is what Basecoat expands).
	if !strings.Contains(html, `id="goal-g-1"`) {
		t.Error("expected first completed card id")
	}
	if !strings.Contains(html, `id="goal-g-2"`) {
		t.Error("expected second completed card id")
	}
}

// TestCompletedGoalsSection_HiddenWhenEmpty asserts the wrapper
// carries class="hidden" when there are no completed goals so it
// takes no visual space, but the wrapper itself is still in the
// DOM (so the OOB swap target exists for a future Mark Complete
// to unhide it).
func TestCompletedGoalsSection_HiddenWhenEmpty(t *testing.T) {
	html := renderToString(t, CompletedGoalsSection(nil, false))

	if !strings.Contains(html, `id="completed-goals-section"`) {
		t.Error("expected completed-goals-section wrapper id (must always be in DOM)")
	}
	if !strings.Contains(html, `class="hidden"`) {
		t.Error("expected class=\"hidden\" on the wrapper when empty")
	}
}

// TestGoalsSections_RendersBothSections asserts the OOB-swap
// response used by Mark Complete / Reopen renders BOTH section
// wrappers (active + completed), each with hx-swap-oob="true" so
// htmx can replace them in the DOM in a single round trip. This
// is the mechanism that moves a card from one section to the
// other without a page reload.
func TestGoalsSections_RendersBothSections(t *testing.T) {
	when := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	active := models.Goal{ID: "g-active", Title: "Active"}
	completed := models.Goal{ID: "g-done", Title: "Done", CompletedAt: &when}
	html := renderToString(t, GoalsSections([]models.Goal{active, completed}, false))

	// Both section wrappers present.
	if !strings.Contains(html, `id="active-goals-section"`) {
		t.Error("expected active-goals-section wrapper id in OOB response")
	}
	if !strings.Contains(html, `id="completed-goals-section"`) {
		t.Error("expected completed-goals-section wrapper id in OOB response")
	}
	// Both wrappers carry hx-swap-oob="true" so htmx can swap
	// them. The attribute must appear at least twice (once per
	// wrapper).
	if strings.Count(html, `hx-swap-oob="true"`) < 2 {
		t.Error("expected hx-swap-oob=\"true\" on both section wrappers")
	}
	// The active card lives inside the active wrapper, the
	// completed card lives inside the completed wrapper. We
	// can't assert DOM order from a string, but we can at least
	// assert both ids are present.
	if !strings.Contains(html, `id="goal-g-active"`) {
		t.Error("expected active card id in OOB response")
	}
	if !strings.Contains(html, `id="goal-g-done"`) {
		t.Error("expected completed card id in OOB response")
	}
}

// TestGoalsSections_PlacesCardInCorrectSection is the key
// behaviour test: when the goals list has a single completed
// goal (and no active ones), the GoalsSections response should
// put that goal card INSIDE the completed wrapper, not the
// active wrapper. We verify this by checking the byte position
// of the card id relative to each section wrapper's id.
func TestGoalsSections_PlacesCardInCorrectSection(t *testing.T) {
	when := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	completed := models.Goal{ID: "g-done", Title: "Done", CompletedAt: &when}
	html := renderToString(t, GoalsSections([]models.Goal{completed}, false))

	idxActive := strings.Index(html, `id="active-goals-section"`)
	idxCompleted := strings.Index(html, `id="completed-goals-section"`)
	idxCard := strings.Index(html, `id="goal-g-done"`)

	if idxActive < 0 || idxCompleted < 0 || idxCard < 0 {
		t.Fatalf("missing expected ids: active=%d completed=%d card=%d", idxActive, idxCompleted, idxCard)
	}
	// Active wrapper comes first (we render them in that order),
	// completed wrapper comes after, the card must be inside the
	// completed wrapper — i.e. its byte position must be after
	// the completed wrapper's opening id.
	if !(idxCompleted < idxCard) {
		t.Errorf("expected goal card to be inside the completed wrapper, got positions active=%d completed=%d card=%d",
			idxActive, idxCompleted, idxCard)
	}
}

// TestGoalsSections_TogglesCompletedVisibility asserts the OOB
// response includes the class="hidden" toggling on the completed
// wrapper: present when empty, absent when non-empty. The
// htmx-driven OOB swap is what actually changes the visibility
// at runtime; this test just pins the rendered class so a
// future refactor can't quietly drop the toggle.
func TestGoalsSections_TogglesCompletedVisibility(t *testing.T) {
	t.Run("non-empty completed", func(t *testing.T) {
		when := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
		completed := models.Goal{ID: "g-1", Title: "x", CompletedAt: &when}
		html := renderToString(t, GoalsSections([]models.Goal{completed}, false))
	if strings.Contains(html, `<div id="completed-goals-section" hx-swap-oob="true" class="hidden"`) {
		t.Error("completed wrapper must NOT be hidden when there are completed goals")
	}
	})
	t.Run("empty completed", func(t *testing.T) {
		html := renderToString(t, GoalsSections(nil, false))
	if !strings.Contains(html, `<div id="completed-goals-section" hx-swap-oob="true" class="hidden"`) {
		t.Error("completed wrapper must be hidden when empty")
	}
	})
}

// TestSplitGoals_PartitionsByCompletion locks in the helper
// behaviour: active goals first, completed goals after, both
// preserving their original internal order.
func TestSplitGoals_PartitionsByCompletion(t *testing.T) {
	when := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	in := []models.Goal{
		{ID: "a1", Title: "active 1"},
		{ID: "c1", Title: "completed 1", CompletedAt: &when},
		{ID: "a2", Title: "active 2"},
		{ID: "c2", Title: "completed 2", CompletedAt: &when},
	}
	active, completed := splitGoals(in)
	if len(active) != 2 || active[0].ID != "a1" || active[1].ID != "a2" {
		t.Errorf("active slice wrong: %+v", active)
	}
	if len(completed) != 2 || completed[0].ID != "c1" || completed[1].ID != "c2" {
		t.Errorf("completed slice wrong: %+v", completed)
	}
}

// TestGoalFormFields_RendersAllInputs asserts the shared form
// body renders the title, description, and three date inputs
// with the supplied values.
func TestGoalFormFields_RendersAllInputs(t *testing.T) {
	html := renderToString(t, GoalFormFields("My title", "My desc", "2026-01-01", "2026-07-01", "2026-12-31", false))
	for _, want := range []string{
		`name="title"`,
		`value="My title"`,
		`name="description"`,
		`>My desc<`,
		`name="start_date"`,
		`value="2026-01-01"`,
		`name="target_date"`,
		`value="2026-07-01"`,
		`name="end_date"`,
		`value="2026-12-31"`,
		`type="date"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in form, got: %s", want, html)
		}
	}
}

// TestGoalFormFields_EmptyDatesOmitValue asserts that empty
// date strings render as empty value attributes (so the input
// shows as blank in the browser).
func TestGoalFormFields_EmptyDatesOmitValue(t *testing.T) {
	html := renderToString(t, GoalFormFields("x", "", "", "", "", false))
	if strings.Contains(html, `value="2026`) {
		t.Error("did not expect a date value with empty inputs")
	}
	if !strings.Contains(html, `name="start_date"`) {
		t.Error("expected start_date input to be present")
	}
}

// TestFormatGoalDateInput_Unit covers the form-side date helper
// directly. Returns "" for nil and the canonical YYYY-MM-DD
// string for a non-nil time.
func TestFormatGoalDateInput_Unit(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if got := formatGoalDateInput(nil); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})
	t.Run("with date", func(t *testing.T) {
		when := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		if got, want := formatGoalDateInput(&when), "2026-07-01"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// TestShortGoalDate_Unit covers the inline date helper used on
// the list card. Returns "" for nil and "DD MMM" (e.g. "12 May")
// for a non-nil time.
func TestShortGoalDate_Unit(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if got := shortGoalDate(nil); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})
	t.Run("with date", func(t *testing.T) {
		when := time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC)
		if got, want := shortGoalDate(&when), "12 May"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
