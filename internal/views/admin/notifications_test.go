package admin

import (
	"strings"
	"testing"
	"time"

	"stren/internal/reminders"
)

// renderToString is the templ test helper used by this file. It
// lives in views_test.go (same package) and renders a templ
// component into a string the test can substring-check.

func TestAdminWeightReminderResult_ShowsCounts(t *testing.T) {
	// Happy path: 5 users due, all 5 emails sent, push
	// reached 4 devices across the per-user broadcasts.
	// The card must surface the aggregate counts so the
	// admin can confirm the run looked right at a glance.
	html := renderToString(t, AdminWeightReminderResult(reminders.TickResult{
		Users:    5,
		Duration: 1234 * time.Millisecond,
		Results: []reminders.UserReminderResult{
			{UserName: "Alice", UserEmail: "alice@example.com", EmailSent: true, PushSent: 1},
			{UserName: "Bob", UserEmail: "bob@example.com", EmailSent: true, PushSent: 1},
			{UserName: "Carol", UserEmail: "carol@example.com", EmailSent: true, PushSent: 1},
			{UserName: "Dave", UserEmail: "dave@example.com", EmailSent: true, PushSent: 1},
			{UserName: "Eve", UserEmail: "eve@example.com", EmailSent: true},
		},
	}))

	for _, want := range []string{
		"Weight reminders fired",
		">5</dd>",     // Users due
		"Emails sent", // the "Emails sent" label
		"Push sent",
		"Duration:",
		"1.2s", // 1234ms rounded to 1.2s
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in result card", want)
		}
	}
}

func TestAdminWeightReminderResult_ShowsPerUserOutcomes(t *testing.T) {
	// Partial failure: 3 of 5 users got the email, one
	// had a transport error, one had push disabled. The
	// card must break each user's outcome out into the
	// three states (sent / skipped / failed) so the
	// operator can spot patterns without grepping logs.
	html := renderToString(t, AdminWeightReminderResult(reminders.TickResult{
		Users:    3,
		Duration: 200 * time.Millisecond,
		Results: []reminders.UserReminderResult{
			{UserName: "Alice", UserEmail: "alice@example.com", EmailSent: true, PushSkipped: true, PushSkipReason: "user has push disabled"},
			{UserName: "Bob", UserEmail: "bob@example.com", EmailFailed: true},
			{UserName: "Carol", UserEmail: "carol@example.com", EmailSent: true, PushSent: 1},
		},
	}))

	if !strings.Contains(html, "Emails failed:") {
		t.Error("expected 'Emails failed:' label")
	}
	if !strings.Contains(html, "Per-user outcomes (3)") {
		t.Error("expected per-user outcomes header with count")
	}
	if !strings.Contains(html, "alice@example.com") {
		t.Error("expected alice@example.com in per-user list")
	}
	if !strings.Contains(html, "push skipped") {
		t.Error("expected 'push skipped' outcome for Alice")
	}
	if !strings.Contains(html, "email failed") {
		t.Error("expected 'email failed' outcome for Bob")
	}
}

func TestAdminWeightReminderResult_ShowsPushAggregateFailure(t *testing.T) {
	// A per-user push broadcast returned a transport
	// failure (e.g. the user-repo read errored). The card
	// must surface the per-user push failure so the
	// operator can see which device fan-outs failed.
	html := renderToString(t, AdminWeightReminderResult(reminders.TickResult{
		Users:    2,
		Duration: 100 * time.Millisecond,
		Results: []reminders.UserReminderResult{
			{UserName: "Alice", UserEmail: "alice@example.com", EmailSent: true, PushFailed: 1, PushSkipReason: "db down"},
			{UserName: "Bob", UserEmail: "bob@example.com", EmailSent: true, PushSkipped: true, PushSkipReason: "no active push subscriptions"},
		},
	}))

	if !strings.Contains(html, "Push failed:") {
		t.Error("expected 'Push failed:' label")
	}
	if !strings.Contains(html, "db down") {
		t.Error("expected per-user push failure reason")
	}
}

func TestAdminWeightReminderResult_CapsPerUserList(t *testing.T) {
	// When there are more per-user rows than the in-card
	// cap, the list is collapsed and a "Showing all N"
	// hint keeps the operator from misreading the
	// truncated list. Mirrors the previous design's
	// "Showing all N addresses" hint.
	results := make([]reminders.UserReminderResult, 30)
	for i := range results {
		results[i] = reminders.UserReminderResult{
			UserName: "User", UserEmail: "u@example.com", EmailSent: true,
		}
	}
	html := renderToString(t, AdminWeightReminderResult(reminders.TickResult{
		Users:    30,
		Results:  results,
		Duration: 500 * time.Millisecond,
	}))

	if !strings.Contains(html, "Showing all 30 users") {
		t.Error("expected 'Showing all 30 users' hint when above cap")
	}
}

func TestAdminWeightReminderResult_DurationFormatsSubSecond(t *testing.T) {
	// A fast run (under a second) should display in ms,
	// not as "0.0s" — the operator cares whether the
	// run was instant or a few hundred ms.
	html := renderToString(t, AdminWeightReminderResult(reminders.TickResult{
		Users:    0,
		Duration: 247 * time.Millisecond,
	}))
	if !strings.Contains(html, "247ms") {
		t.Errorf("expected '247ms' in result card, got: %q", html)
	}
}

func TestAdminWeightReminderResult_NoUsersRendersQuietly(t *testing.T) {
	// When the due-user list is empty (the common case
	// when nobody has scheduled a reminder for this hour),
	// the card must still render but with no per-user
	// section so the admin does not see an empty header.
	html := renderToString(t, AdminWeightReminderResult(reminders.TickResult{
		Users:    0,
		Duration: 5 * time.Millisecond,
	}))
	if !strings.Contains(html, "Weight reminders fired") {
		t.Error("expected title in result card")
	}
	if strings.Contains(html, "Per-user outcomes") {
		t.Error("did not expect per-user section when no users were due")
	}
}

func TestAdminNotificationsPage_ContainsWeightReminderButton(t *testing.T) {
	// The admin page must render the new button with the
	// right hx-post so the click fires the new route. This
	// is the tripwire: a future refactor that drops the
	// button from the layout fails the test.
	html := renderToString(t, AdminNotificationsPage("Admin", true))

	for _, want := range []string{
		`hx-post="/admin/notifications/send-weight-reminder"`,
		`id="weight-reminder-button"`,
		`id="weight-reminder-spinner"`,
		`Send all due reminders now`,
		`hx-target="#send-result"`,
		`hx-disabled-elt="this"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in admin notifications page", want)
		}
	}
}
