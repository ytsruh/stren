package models

import (
	"testing"
	"time"
)

// fixedClock returns a deterministic time for test fixtures. The
// orchestrator's tests use a 2026-08-05 Wednesday anchor so the
// arithmetic does not accidentally cross a DST or month boundary.
var computeFireTestNow = time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)

func intPtr(v int) *int { return &v }

func TestUser_RemindersEnabled(t *testing.T) {
	cases := []struct {
		name string
		user User
		want bool
	}{
		{name: "nil user is off", user: User{}, want: false},
		{name: "master switch off is off",
			user: User{ReminderEnabled: false, ReminderFrequency: ReminderWeekly}, want: false},
		{name: "frequency off is off even with master on",
			user: User{ReminderEnabled: true, ReminderFrequency: ReminderOff}, want: false},
		{name: "master on + daily is on",
			user: User{ReminderEnabled: true, ReminderFrequency: ReminderDaily}, want: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.RemindersEnabled(); got != tt.want {
				t.Errorf("RemindersEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReminderFrequency_IsValid(t *testing.T) {
	for _, f := range AllReminderFrequencies {
		if !f.IsValid() {
			t.Errorf("AllReminderFrequencies contains invalid value %q", f)
		}
	}
	if ReminderFrequency("yearly").IsValid() {
		t.Error("yearly should be invalid")
	}
	if ReminderFrequency("").IsValid() {
		t.Error("empty should be invalid")
	}
}

func TestReminderFrequency_NeedsDayOfWeek(t *testing.T) {
	cases := map[ReminderFrequency]bool{
		ReminderOff:      false,
		ReminderDaily:    false,
		ReminderWeekly:   true,
		ReminderBiweekly: true,
	}
	for f, want := range cases {
		if got := f.NeedsDayOfWeek(); got != want {
			t.Errorf("NeedsDayOfWeek(%q) = %v, want %v", f, got, want)
		}
	}
}

func TestUser_ReminderHour_DefaultOnMalformed(t *testing.T) {
	// A row with a corrupted reminder_time must not panic; the
	// default 9 keeps the tick alive instead of skipping the user.
	cases := []struct {
		name, time string
		want       int
	}{
		{"valid 18:00", "18:00", 18},
		{"valid 00:00", "00:00", 0},
		{"valid 09:00", "09:00", 9},
		{"garbage falls back to 9", "not-a-time", 9},
		{"empty falls back to 9", "", 9},
		{"minute precision is rejected (per hour-only design)", "09:30", 9},
		{"out of range falls back to 9", "25:00", 9},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := (&User{ReminderTime: tt.time}).ReminderHour(); got != tt.want {
				t.Errorf("ReminderHour() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseReminderTime(t *testing.T) {
	if h, m, err := parseReminderTime("07:00"); err != nil || h != 7 || m != 0 {
		t.Errorf("parseReminderTime(07:00) = %d, %d, %v", h, m, err)
	}
	if _, _, err := parseReminderTime("07:30"); err == nil {
		t.Error("parseReminderTime(07:30) should reject non-zero minute")
	}
	if _, _, err := parseReminderTime(""); err == nil {
		t.Error("parseReminderTime(\"\") should reject empty")
	}
}

func TestFormatReminderHour(t *testing.T) {
	cases := map[int]string{
		0:  "00:00",
		9:  "09:00",
		18: "18:00",
		23: "23:00",
		-1: "00:00", // clamped
		24: "23:00", // clamped
	}
	for in, want := range cases {
		if got := FormatReminderHour(in); got != want {
			t.Errorf("FormatReminderHour(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestUser_ComputeNextFire_OffReturnsFalse(t *testing.T) {
	// Master off or frequency off: never schedule.
	u := User{ReminderEnabled: false, ReminderFrequency: ReminderWeekly, ReminderTime: "10:00"}
	if _, ok := u.ComputeNextFire(computeFireTestNow); ok {
		t.Error("ComputeNextFire(off) returned a time, want false")
	}
	u = User{ReminderEnabled: true, ReminderFrequency: ReminderOff, ReminderTime: "10:00"}
	if _, ok := u.ComputeNextFire(computeFireTestNow); ok {
		t.Error("ComputeNextFire(frequency=off) returned a time, want false")
	}
}

func TestUser_ComputeNextFire_Daily(t *testing.T) {
	// Daily at 14:00 UTC, now=2026-08-05 10:00 UTC → today 14:00.
	u := User{
		ReminderEnabled:   true,
		ReminderFrequency: ReminderDaily,
		ReminderTime:      "14:00",
		CreatedAt:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	got, ok := u.ComputeNextFire(computeFireTestNow)
	if !ok {
		t.Fatal("ComputeNextFire(daily) returned false")
	}
	want := time.Date(2026, 8, 5, 14, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("daily next fire = %v, want %v", got, want)
	}

	// If now is past the hour, next is tomorrow at the same hour.
	later := time.Date(2026, 8, 5, 15, 0, 0, 0, time.UTC)
	got, _ = u.ComputeNextFire(later)
	want = time.Date(2026, 8, 6, 14, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("daily next fire (post-hour) = %v, want %v", got, want)
	}
}

func TestUser_ComputeNextFire_Weekly(t *testing.T) {
	// Weekly on Sunday (0). Now is Wednesday → next Sunday.
	sun := 0
	u := User{
		ReminderEnabled:   true,
		ReminderFrequency: ReminderWeekly,
		ReminderDayOfWeek: &sun,
		ReminderTime:      "09:00",
		CreatedAt:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	got, ok := u.ComputeNextFire(computeFireTestNow)
	if !ok {
		t.Fatal("ComputeNextFire(weekly) returned false")
	}
	want := time.Date(2026, 8, 9, 9, 0, 0, 0, time.UTC) // Sunday
	if !got.Equal(want) {
		t.Errorf("weekly (wed) next fire = %v, want %v", got, want)
	}

	// Now is Sunday at 08:00 → today's 09:00 is still in the future.
	sunMorning := time.Date(2026, 8, 9, 8, 0, 0, 0, time.UTC)
	got, _ = u.ComputeNextFire(sunMorning)
	if !got.Equal(want) {
		t.Errorf("weekly (sun 08:00) next fire = %v, want %v", got, want)
	}

	// Now is Sunday at 10:00 → next Sunday.
	sunLate := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	got, _ = u.ComputeNextFire(sunLate)
	want = time.Date(2026, 8, 16, 9, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("weekly (sun 10:00) next fire = %v, want %v", got, want)
	}
}

func TestUser_ComputeNextFire_Biweekly(t *testing.T) {
	// Biweekly on Sunday. The first fire is the very next
	// <Sunday> at HH:00 — exactly the same rule as weekly. The
	// biweekly stride (14d between subsequent fires) is applied
	// by the orchestrator after a successful fire, not here.
	sun := 0
	u := User{
		ReminderEnabled:   true,
		ReminderFrequency: ReminderBiweekly,
		ReminderDayOfWeek: &sun,
		ReminderTime:      "09:00",
		CreatedAt:         time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC),
	}
	got, ok := u.ComputeNextFire(computeFireTestNow)
	if !ok {
		t.Fatal("ComputeNextFire(biweekly) returned false")
	}
	want := time.Date(2026, 8, 9, 9, 0, 0, 0, time.UTC) // Sunday
	if !got.Equal(want) {
		t.Errorf("biweekly next fire = %v, want %v", got, want)
	}

	// A different created_at does not affect the first-fire
	// computation — the simpler "next candidate wins" model is
	// deliberate, so the user's signup history never delays the
	// first reminder.
	u.CreatedAt = time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)
	got, _ = u.ComputeNextFire(computeFireTestNow)
	if !got.Equal(want) {
		t.Errorf("biweekly (alt created_at) next fire = %v, want %v", got, want)
	}
}

func TestUser_ComputeNextFire_Biweekly_FutureSignup(t *testing.T) {
	// A user who signs up mid-week should still get the very next
	// <day_of_week> as their first fire — not a 2-week wait. This
	// is the case that motivated dropping the parity-based anchor.
	sun := 0
	u := User{
		ReminderEnabled:   true,
		ReminderFrequency: ReminderBiweekly,
		ReminderDayOfWeek: &sun,
		ReminderTime:      "09:00",
		CreatedAt:         time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC), // Wednesday
	}
	got, ok := u.ComputeNextFire(computeFireTestNow)
	if !ok {
		t.Fatal("ComputeNextFire(biweekly, mid-week signup) returned false")
	}
	want := time.Date(2026, 8, 9, 9, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("biweekly (mid-week) next fire = %v, want %v", got, want)
	}
}
