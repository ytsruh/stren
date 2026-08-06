package models

import (
	"testing"
	"time"
)

// TestGoal_IsComplete asserts the boolean helper used by every view that
// switches between "Mark Complete" and "Reopen" actions.
func TestGoal_IsComplete(t *testing.T) {
	now := time.Now()
	t.Run("no completed_at", func(t *testing.T) {
		g := Goal{}
		if g.IsComplete() {
			t.Error("expected IsComplete()=false for goal with nil CompletedAt")
		}
	})
	t.Run("with completed_at", func(t *testing.T) {
		g := Goal{CompletedAt: &now}
		if !g.IsComplete() {
			t.Error("expected IsComplete()=true for goal with non-nil CompletedAt")
		}
	})
}

// TestGoal_FormattedCompletedDate covers the long-format date used on
// the completed-goal card. We assert the exact format the user sees
// so any change is a deliberate decision.
func TestGoal_FormattedCompletedDate(t *testing.T) {
	t.Run("nil CompletedAt", func(t *testing.T) {
		g := Goal{}
		if got := g.FormattedCompletedDate(); got != "" {
			t.Errorf("expected empty string for nil CompletedAt, got %q", got)
		}
	})
	t.Run("with CompletedAt", func(t *testing.T) {
		when := time.Date(2026, 1, 9, 8, 0, 0, 0, time.UTC)
		g := Goal{CompletedAt: &when}
		if got, want := g.FormattedCompletedDate(), "09 Jan 2026"; got != want {
			t.Errorf("FormattedCompletedDate = %q, want %q", got, want)
		}
	})
}

// TestFormatGoalDate locks the long-format date helper. The view uses
// this to render every optional date chip (start / target / end) so
// the format must stay consistent across the card.
func TestFormatGoalDate(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if got := FormatGoalDate(nil); got != "" {
			t.Errorf("expected empty string for nil, got %q", got)
		}
	})
	t.Run("with date", func(t *testing.T) {
		when := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		if got, want := FormatGoalDate(&when), "01 Jul 2026"; got != want {
			t.Errorf("FormatGoalDate = %q, want %q", got, want)
		}
	})
}

// TestGoal_DaysUntilTarget covers the boundary conditions of the
// "due in N days" chip. Past and same-day targets return nil so the
// view can suppress the chip entirely; future targets return the
// whole-day delta. We also verify the day-boundary truncation
// (00:00:00 on both sides) so a target later in the same day is
// reported as "today" (0 days), not "1 day".
func TestGoal_DaysUntilTarget(t *testing.T) {
	mustDate := func(d int) time.Time {
		// Use midday UTC so we're well away from any timezone boundary.
		return time.Date(2026, 6, d, 12, 0, 0, 0, time.UTC)
	}

	t.Run("no target", func(t *testing.T) {
		g := Goal{}
		if got := g.DaysUntilTarget(mustDate(15)); got != nil {
			t.Errorf("expected nil for goal with nil TargetDate, got %v", got)
		}
	})
	t.Run("target today (later in the day)", func(t *testing.T) {
		now := time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC)
		target := time.Date(2026, 6, 15, 23, 0, 0, 0, time.UTC)
		g := Goal{TargetDate: &target}
		if got := g.DaysUntilTarget(now); got == nil || *got != 0 {
			t.Errorf("expected 0 (today), got %v", got)
		}
	})
	t.Run("target in 5 days", func(t *testing.T) {
		now := mustDate(15)
		target := mustDate(20)
		g := Goal{TargetDate: &target}
		if got := g.DaysUntilTarget(now); got == nil || *got != 5 {
			t.Errorf("expected 5, got %v", got)
		}
	})
	t.Run("target yesterday returns nil", func(t *testing.T) {
		now := mustDate(15)
		target := mustDate(14)
		g := Goal{TargetDate: &target}
		if got := g.DaysUntilTarget(now); got != nil {
			t.Errorf("expected nil for past target, got %v", got)
		}
	})
	t.Run("target far in the past", func(t *testing.T) {
		now := mustDate(15)
		target := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		g := Goal{TargetDate: &target}
		if got := g.DaysUntilTarget(now); got != nil {
			t.Errorf("expected nil for past target, got %v", got)
		}
	})
}
