package admin

import (
	"strings"
	"testing"
	"time"

	"stren/internal/reminders"
)

func TestAdminWeightReminderResult_ShowsCounts(t *testing.T) {
	// Happy path: 5 users, 5 emails sent, 0 failed, push
	// reached 4 devices. The card must surface every
	// important number so the admin can confirm the run
	// looked right at a glance.
	html := renderToString(t, AdminWeightReminderResult(reminders.RunResult{
		Users:      5,
		EmailsSent: 5,
		PushSent:   4,
		Duration:   1234 * time.Millisecond,
	}))

	for _, want := range []string{
		"Weekly weight reminder sent",
		">5</dd>",     // Users
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

func TestAdminWeightReminderResult_ShowsFailedAddresses(t *testing.T) {
	// Partial failure: 3 of 5 users got the email; the
	// card must list the failed addresses inline so the
	// operator can spot a pattern.
	html := renderToString(t, AdminWeightReminderResult(reminders.RunResult{
		Users:                 5,
		EmailsSent:            3,
		EmailsFailed:          2,
		EmailsFailedAddresses: []string{"alice@example.com", "bob@example.com"},
	}))

	if !strings.Contains(html, "Emails failed:") {
		t.Error("expected 'Emails failed:' label")
	}
	if !strings.Contains(html, "Failed email addresses (2)") {
		t.Error("expected failed-addresses details header with count")
	}
	for _, addr := range []string{"alice@example.com", "bob@example.com"} {
		if !strings.Contains(html, addr) {
			t.Errorf("expected failed address %q in card", addr)
		}
	}
}

func TestAdminWeightReminderResult_ShowsPushError(t *testing.T) {
	// The push service itself errored (e.g. the store was
	// unreadable). The card must surface the error message
	// rather than just the broadcast counts.
	html := renderToString(t, AdminWeightReminderResult(reminders.RunResult{
		Users:     2,
		PushError: "push service unreachable",
	}))

	if !strings.Contains(html, "Push service error: push service unreachable") {
		t.Error("expected push service error message in card")
	}
}

func TestAdminWeightReminderResult_CapsFailedAddresses(t *testing.T) {
	// When there are more failed addresses than the
	// in-card cap, the list is still rendered (no data
	// loss) and a "Showing all N addresses" hint keeps
	// the operator from misreading the truncated list.
	html := renderToString(t, AdminWeightReminderResult(reminders.RunResult{
		Users:        30,
		EmailsSent:   5,
		EmailsFailed: 25,
		EmailsFailedAddresses: []string{
			"u1@example.com", "u2@example.com", "u3@example.com",
			"u4@example.com", "u5@example.com", "u6@example.com",
			"u7@example.com", "u8@example.com", "u9@example.com",
			"u10@example.com", "u11@example.com", "u12@example.com",
			"u13@example.com", "u14@example.com", "u15@example.com",
			"u16@example.com", "u17@example.com", "u18@example.com",
			"u19@example.com", "u20@example.com", "u21@example.com",
			"u22@example.com", "u23@example.com", "u24@example.com",
			"u25@example.com",
		},
	}))

	if !strings.Contains(html, "Showing all 25 addresses") {
		t.Error("expected 'Showing all 25 addresses' hint when above cap")
	}
}

func TestAdminWeightReminderResult_DurationFormatsSubSecond(t *testing.T) {
	// A fast run (under a second) should display in ms,
	// not as "0.0s" — the operator cares whether the
	// run was instant or a few hundred ms.
	html := renderToString(t, AdminWeightReminderResult(reminders.RunResult{
		Users:    0,
		Duration: 247 * time.Millisecond,
	}))
	if !strings.Contains(html, "247ms") {
		t.Errorf("expected '247ms' in result card, got: %q", html)
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
		`hx-confirm="Send the weekly weight reminder to every user now?`,
		`hx-target="#send-result"`,
		`hx-disabled-elt="this"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in admin notifications page", want)
		}
	}
}
