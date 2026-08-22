package reminders

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"stren/internal/models"
)

// --- Test doubles for the orchestrator's dependencies ---

// fakeReminderRepo is a small in-memory ReminderRepo for tests.
// It implements both the list-due-users query and the
// mark-fired advancement, so the orchestrator's full path is
// exercised without a real database. Thread-safe so the
// orchestrator's concurrent sends do not race the test's
// setup.
type fakeReminderRepo struct {
	mu          sync.Mutex
	due         []models.User
	listErr     error
	firedLog    []firedRecord
	markErr     error
	nextFireFor map[string]time.Time
}

type firedRecord struct {
	UserID    string
	FiredAt   time.Time
	NextFire  time.Time
}

func (f *fakeReminderRepo) ListUsersDueForReminder(_ context.Context, now time.Time) ([]models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]models.User, len(f.due))
	copy(out, f.due)
	return out, nil
}

func (f *fakeReminderRepo) MarkUserReminderFired(_ context.Context, userID string, firedAt, nextFire time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.markErr != nil {
		return f.markErr
	}
	f.firedLog = append(f.firedLog, firedRecord{UserID: userID, FiredAt: firedAt, NextFire: nextFire})
	if f.nextFireFor == nil {
		f.nextFireFor = map[string]time.Time{}
	}
	f.nextFireFor[userID] = nextFire
	return nil
}

func (f *fakeReminderRepo) firedRecords() []firedRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]firedRecord, len(f.firedLog))
	copy(out, f.firedLog)
	return out
}

// fakeEmailSender records every per-user send the orchestrator
// asks for. errAll makes every send fail; errFor makes a
// specific user fail (e.g. to test the "email failed" path).
type fakeEmailSender struct {
	mu     sync.Mutex
	calls  []string
	errAll error
	errFor map[string]error
}

func (f *fakeEmailSender) SendWeightReminder(_ context.Context, u *models.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, u.Email)
	if f.errAll != nil {
		return f.errAll
	}
	if f.errFor != nil {
		if e, ok := f.errFor[u.Email]; ok {
			return e
		}
	}
	return nil
}

func (f *fakeEmailSender) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// fixedClock returns a constant time. The orchestrator's tests
// pin "now" so the reminder stride (24h / 7d / 14d) is
// deterministic.
type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

// newTestReminder wires the fakes into a UserReminder.
// Tests override the fakes before calling Run or SendToUser.
func newTestReminder(t *testing.T) (*UserReminder, *fakeReminderRepo, *fakeEmailSender) {
	t.Helper()
	repo := &fakeReminderRepo{}
	emails := &fakeEmailSender{}
	r, err := NewUserReminder(repo, emails, UserReminderConfig{MaxEmailWorkers: 4})
	if err != nil {
		t.Fatalf("NewUserReminder: %v", err)
	}
	// Pin the clock to a known instant so the "next fire"
	// math is deterministic. The tests assert against this
	// value when they care.
	r.clock = fixedClock{t: time.Date(2026, 8, 9, 9, 0, 0, 0, time.UTC)}
	return r, repo, emails
}

// --- NewUserReminder construction ---

func TestNewUserReminder_RejectsNilDependencies(t *testing.T) {
	if _, err := NewUserReminder(nil, &fakeEmailSender{}, UserReminderConfig{}); err == nil {
		t.Error("expected error for nil repo")
	}
	if _, err := NewUserReminder(&fakeReminderRepo{}, nil, UserReminderConfig{}); err == nil {
		t.Error("expected error for nil emails")
	}
}

// --- Run: end-to-end tick ---

func TestUserReminder_Run_NoUsersDue(t *testing.T) {
	// No due users → no email sends and no
	// mark-fired calls. The TickResult must reflect the
	// zero-attempt case (Attempted: true, Users: 0) so the
	// admin UI can distinguish it from a list error.
	r, repo, emails := newTestReminder(t)
	res, attempted := r.Run(context.Background())
	if !attempted {
		t.Error("attempted = false, want true (list succeeded)")
	}
	if res.Users != 0 {
		t.Errorf("Users = %d, want 0", res.Users)
	}
	if got := emails.callCount(); got != 0 {
		t.Errorf("email calls = %d, want 0", got)
	}
	if got := len(repo.firedRecords()); got != 0 {
		t.Errorf("mark-fired calls = %d, want 0", got)
	}
}

func TestUserReminder_Run_FiresOneEmailPerUser(t *testing.T) {
	// Each user must get exactly one email send. The
	// orchestrator's per-user send is sequential, so the
	// counts line up one-to-one.
	r, repo, emails := newTestReminder(t)
	day := 0
	repo.due = []models.User{
		{ID: "u1", Name: "Alice", Email: "alice@example.com",
			ReminderEnabled: true, ReminderFrequency: models.ReminderWeekly,
			ReminderDayOfWeek: &day, ReminderTime: "09:00",
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "u2", Name: "Bob", Email: "bob@example.com",
			ReminderEnabled: true, ReminderFrequency: models.ReminderWeekly,
			ReminderDayOfWeek: &day, ReminderTime: "09:00",
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	res, attempted := r.Run(context.Background())
	if !attempted {
		t.Fatal("attempted = false, want true")
	}
	if res.Users != 2 {
		t.Errorf("Users = %d, want 2", res.Users)
	}
	if got := emails.callCount(); got != 2 {
		t.Errorf("email calls = %d, want 2", got)
	}
	// And each user must have had mark-fired called once
	// with the right firedAt (the pinned clock) and an
	// advanced next_fire_at (one week later for weekly).
	fired := repo.firedRecords()
	if len(fired) != 2 {
		t.Fatalf("mark-fired calls = %d, want 2", len(fired))
	}
	for _, rec := range fired {
		if !rec.FiredAt.Equal(r.clock.Now()) {
			t.Errorf("user %s: FiredAt = %v, want %v", rec.UserID, rec.FiredAt, r.clock.Now())
		}
		if rec.NextFire.Before(rec.FiredAt) {
			t.Errorf("user %s: NextFire %v is before FiredAt %v", rec.UserID, rec.NextFire, rec.FiredAt)
		}
	}
}

func TestUserReminder_Run_AdvancesByCadenceStride(t *testing.T) {
	// After firing, the next_fire_at must be exactly the
	// cadence's stride later: 24h for daily, 7d for weekly,
	// 14d for biweekly. This is the tripwire for any future
	// refactor that mis-computes the advance.
r, repo, _ := newTestReminder(t)
	day := 0
	now := r.clock.Now()
	// Pre-compute the "next candidate" for each frequency so
	// the orchestrator's advance puts the next_fire_at on
	// the expected cadence boundary.
	mk := func(id, name, email string, freq models.ReminderFrequency) models.User {
		u := models.User{
			ID: id, Name: name, Email: email,
			ReminderEnabled: true, ReminderFrequency: freq,
			ReminderDayOfWeek: &day, ReminderTime: "09:00",
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		// Pre-fire candidate is "today at 09:00" for the
		// user — we set nextFire to now so the orchestrator
		// picks them up.
		next := now
		u.ReminderNextFireAt = &next
		return u
	}
	repo.due = []models.User{
		mk("daily", "Daily", "d@x.com", models.ReminderDaily),
		mk("weekly", "Weekly", "w@x.com", models.ReminderWeekly),
		mk("biweekly", "Biweekly", "b@x.com", models.ReminderBiweekly),
	}

	r.Run(context.Background())

	fired := repo.firedRecords()
	if len(fired) != 3 {
		t.Fatalf("mark-fired calls = %d, want 3", len(fired))
	}
	for _, rec := range fired {
		var want time.Duration
		switch rec.UserID {
		case "daily":
			want = 24 * time.Hour
		case "weekly":
			want = 7 * 24 * time.Hour
		case "biweekly":
			want = 14 * 24 * time.Hour
		default:
			t.Errorf("unexpected user %q", rec.UserID)
			continue
		}
		// The orchestrator advances by the stride and then
		// re-computes the next candidate. The new next_fire_at
		// should be the previous next_fire_at + stride (since
		// we set the row's next_fire_at to now=09:00 today).
		got := rec.NextFire.Sub(now)
		if got != want {
			t.Errorf("user %s: next fire advance = %v, want %v", rec.UserID, got, want)
		}
	}
}

func TestUserReminder_Run_FailingEmailDoesNotPoisonBatch(t *testing.T) {
	// A single SMTP failure on one user must not stop the
	// other users from receiving their email, and the
	// failing user must still have their next_fire_at
	// advanced (we don't want them stuck in a retry loop).
	r, repo, emails := newTestReminder(t)
	day := 0
	repo.due = []models.User{
		{ID: "u1", Name: "Alice", Email: "alice@example.com",
			ReminderEnabled: true, ReminderFrequency: models.ReminderWeekly,
			ReminderDayOfWeek: &day, ReminderTime: "09:00",
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "u2", Name: "Bob", Email: "bob@example.com",
			ReminderEnabled: true, ReminderFrequency: models.ReminderWeekly,
			ReminderDayOfWeek: &day, ReminderTime: "09:00",
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	emails.errFor = map[string]error{
		"bob@example.com": errors.New("smtp 421"),
	}

	res, attempted := r.Run(context.Background())
	if !attempted {
		t.Fatal("attempted = false, want true")
	}
	if got := emails.callCount(); got != 2 {
		t.Errorf("email calls = %d, want 2 (every user attempted)", got)
	}
	// Find the per-user outcomes.
	var alice, bob *UserReminderResult
	for i, r := range res.Results {
		if r.UserEmail == "alice@example.com" {
			alice = &res.Results[i]
		}
		if r.UserEmail == "bob@example.com" {
			bob = &res.Results[i]
		}
	}
	if alice == nil || !alice.EmailSent {
		t.Error("expected alice EmailSent = true")
	}
	if bob == nil || !bob.EmailFailed || bob.EmailSent {
		t.Errorf("expected bob EmailFailed = true; got %+v", bob)
	}
	// And both must have had mark-fired called so they
	// don't retry immediately on the next tick.
	if got := len(repo.firedRecords()); got != 2 {
		t.Errorf("mark-fired calls = %d, want 2 (both advanced)", got)
	}
}

func TestUserReminder_Run_ListErrorAborts(t *testing.T) {
	// A failure to read the due-user list is the only
	// error path that short-circuits Run. Nothing was
	// attempted yet, so there is nothing to fall through
	// to. The orchestrator logs and returns; the test
	// asserts that no downstream work happened, and that
	// the admin UI can distinguish "list failed" from
	// "list succeeded with zero users" via the bool.
	r, repo, emails := newTestReminder(t)
	repo.listErr = errors.New("db down")

	res, attempted := r.Run(context.Background())
	if attempted {
		t.Error("attempted = true, want false (list failed)")
	}
	if res.ListError == "" {
		t.Error("result.ListError is empty, want a non-empty error string")
	}
	if got := emails.callCount(); got != 0 {
		t.Errorf("email calls = %d, want 0 when list failed", got)
	}
}

// --- SendToUser: per-user channel skip rules ---

func TestUserReminder_SendToUser_EmptyEmailIsSkipped(t *testing.T) {
	// The email address is the only per-user skip condition left
	// (defensive — every user should have one, but the
	// orchestrator does not assume the user repo is perfect).
	// A skipped user must still advance so the tick does not
	// pick them up every hour.
	r, repo, emails := newTestReminder(t)
	day := 0
	u := &models.User{
		ID: "u1", Name: "Alice", Email: "",
		ReminderEnabled: true, ReminderFrequency: models.ReminderWeekly,
		ReminderDayOfWeek: &day, ReminderTime: "09:00",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	res := r.SendToUser(context.Background(), u, r.clock.Now())
	if !res.EmailSkipped {
		t.Error("EmailSkipped = false, want true (empty email)")
	}
	if res.EmailSent || res.EmailFailed {
		t.Errorf("expected EmailSent=EmailFailed=false; got %+v", res)
	}
	if got := emails.callCount(); got != 0 {
		t.Errorf("email calls = %d, want 0 when email is empty", got)
	}
	if got := len(repo.firedRecords()); got != 1 {
		t.Errorf("mark-fired calls = %d, want 1 (user advanced even when skipped)", got)
	}
}

func TestUserReminder_SendToUser_AdvancesNextFireByCadence(t *testing.T) {
	// SendToUser must advance the user's next_fire_at by
	// the cadence's stride so the next tick skips them
	// until the new time. A regression that forgot the
	// advance would have the tick pick the user up every
	// hour, which the seed users are configured to
	// tolerate but the operator would notice.
r, repo, _ := newTestReminder(t)
	day := 0
	for _, freq := range []models.ReminderFrequency{
		models.ReminderDaily, models.ReminderWeekly, models.ReminderBiweekly,
	} {
		repo.mu.Lock()
		repo.firedLog = nil
		repo.mu.Unlock()
		u := &models.User{
			ID: "u1", Name: "Alice", Email: "a@x.com",
			ReminderEnabled: true, ReminderFrequency: freq,
			ReminderDayOfWeek: &day, ReminderTime: "09:00",
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		now := r.clock.Now()
		r.SendToUser(context.Background(), u, now)
		fired := repo.firedRecords()
		if len(fired) != 1 {
			t.Fatalf("%s: mark-fired calls = %d, want 1", freq, len(fired))
		}
		var want time.Duration
		switch freq {
		case models.ReminderDaily:
			want = 24 * time.Hour
		case models.ReminderWeekly:
			want = 7 * 24 * time.Hour
		case models.ReminderBiweekly:
			want = 14 * 24 * time.Hour
		}
		// We pass `now + stride` into ComputeNextFire (via
		// reminderStride), so the new next_fire_at is the
		// "next candidate at-or-after now+stride". For the
		// canonical case of weekly Sunday 09:00 with
		// now=Sun 09:00, +7d is also a Sunday 09:00, so the
		// next_fire_at lands exactly 7d later.
		got := fired[0].NextFire.Sub(now)
		// Allow a small fudge for the day-of-week math
		// (sub-day rolls on biweekly across DST etc).
		if got < want || got > want+24*time.Hour {
			t.Errorf("%s: advance = %v, want ~%v", freq, got, want)
		}
	}
}

func TestUserReminder_SendToUser_MarkFiredErrorDoesNotPanic(t *testing.T) {
	// A failure to write the next_fire_at back to the DB
	// must not panic. The orchestrator records the error
	// in the per-user result and moves on; the next tick
	// will re-attempt (the row is still "due" because the
	// write failed). This is the same recovery story as
	// the all-user email failure.
r, repo, _ := newTestReminder(t)
	repo.markErr = errors.New("db write failed")
	day := 0
	u := &models.User{
		ID: "u1", Name: "Alice", Email: "a@x.com",
		ReminderEnabled: true, ReminderFrequency: models.ReminderWeekly,
		ReminderDayOfWeek: &day, ReminderTime: "09:00",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	res := r.SendToUser(context.Background(), u, r.clock.Now())
	if res.Error == "" {
		t.Error("expected non-empty Error on mark-fired failure")
	}
}
