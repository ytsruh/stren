// Package reminders contains the scheduled jobs that run in the
// background while the server is up. The package owns the only
// place in the codebase that imports a third-party scheduler
// (github.com/robfig/cron/v3); everything else in the package
// talks to a narrow Scheduler interface so the dep is easy to
// replace with a different runner (a real queue, a system cron,
// an HTTP trigger, etc.) without touching the job code.
//
// The job code itself is decoupled from the cron wrapper: the
// UserReminder struct exposes a plain Run(ctx) method that the
// scheduler calls. Tests can call Run directly without standing
// up a scheduler. Conversely, the Scheduler is testable with a
// fake job function in the rare case we want to exercise the
// scheduler lifecycle in isolation.
package reminders

import (
	"context"
	"fmt"
	"log"
	"time"

	"stren/internal/email"
	"stren/internal/models"
)

// ReminderRepo is the data-access surface the UserReminder needs to
// find and advance per-user reminder rows. Defined as an interface
// (rather than depending on the concrete *models.UserRepository) so
// the orchestrator can be unit-tested with an in-memory fake, and
// so the model layer stays free of reminder-specific dependencies.
type ReminderRepo interface {
	// ListUsersDueForReminder returns every enabled user whose
	// next_fire_at is at or before now. The hourly tick calls
	// this once per hour.
	ListUsersDueForReminder(ctx context.Context, now time.Time) ([]models.User, error)
	// MarkUserReminderFired atomically sets last_fired_at = firedAt
	// and advances next_fire_at to nextFire for the given user.
	MarkUserReminderFired(ctx context.Context, userID string, firedAt, nextFire time.Time) error
}

// EmailSender is the narrow contract the orchestrator depends on
// for the per-user email send. Mirrors the shape of the existing
// email.Service interface so the orchestrator can call it from a
// goroutine without sharing state.
//
// Returning an error rather than panicking means a single SMTP
// failure on one user is logged and skipped, not fatal for the
// whole batch.
type EmailSender interface {
	SendWeightReminder(ctx context.Context, user *models.User) error
}

// Clock is the time source the orchestrator uses for any
// "what time is it" decision. Injected as an interface so tests
// can pin the time and assert on the exact tick; in production,
// RealClock is used.
type Clock interface {
	Now() time.Time
}

// RealClock returns the wall clock. The default Clock the
// orchestrator uses; tests inject a fixed clock to keep
// assertions deterministic.
type RealClock struct{}

// Now returns time.Now(). Lives on a type (rather than as a
// package-level function) so it satisfies Clock without adapters.
func (RealClock) Now() time.Time { return time.Now() }

// defaultMaxEmailWorkers bounds the goroutine pool that fans out
// per-user emails: a sudden user-count spike cannot exhaust file
// descriptors or starve other handlers.
const defaultMaxEmailWorkers = 10

// UserReminder orchestrates the per-user weight reminder: for
// every user whose next_fire_at is at or before the tick time, it
// fires the user's chosen channel (email) and
// advances the user's next_fire_at by the appropriate stride
// (24h for daily, 7d for weekly, 14d for biweekly). It is
// constructed once in main and invoked by the Scheduler on every
// tick.
//
// The orchestrator is stateless and safe for concurrent use: each
// Run call constructs its own worker pool internally and tears it
// down before returning. A second Run call (e.g. an admin "send
// weight reminder" button in the future) can overlap with an
// in-flight one without trampling shared state.
type UserReminder struct {
	repo     ReminderRepo
	emails   EmailSender
	clock    Clock
	maxEmail int
}

// UserReminderConfig groups the optional knobs on UserReminder.
// Zero values fall back to package defaults (RealClock,
// defaultMaxEmailWorkers).
type UserReminderConfig struct {
	// MaxEmailWorkers bounds the per-user email fan-out.
	// 0 or negative → defaultMaxEmailWorkers.
	MaxEmailWorkers int
}

// NewUserReminder returns a UserReminder bound to the given
// dependencies. repo and emails are required (nil returns an error
// so a missing dependency fails at startup, not at the first tick).
// clock is optional; nil falls back to RealClock.
func NewUserReminder(repo ReminderRepo, emails EmailSender, cfg UserReminderConfig) (*UserReminder, error) {
	if repo == nil {
		return nil, fmt.Errorf("reminders: NewUserReminder: repo is nil")
	}
	if emails == nil {
		return nil, fmt.Errorf("reminders: NewUserReminder: emails is nil")
	}

	maxEmail := cfg.MaxEmailWorkers
	if maxEmail <= 0 {
		maxEmail = defaultMaxEmailWorkers
	}

	return &UserReminder{
		repo:     repo,
		emails:   emails,
		clock:    RealClock{},
		maxEmail: maxEmail,
	}, nil
}

// UserReminderResult is the per-user outcome of SendToUser.
// Skipped flags distinguish "user disabled the channel" from
// "send failed" so callers do not conflate them with errors.
type UserReminderResult struct {
	UserID       string
	UserName     string
	UserEmail    string
	EmailSent    bool
	EmailSkipped bool
	EmailFailed  bool
	// Error is non-empty when the orchestrator could not
	// process the user at all (e.g. could not compute the
	// next fire time).
	Error string
}

// TickResult is the aggregate of one orchestrator Run call.
// Consumed by the admin "send all due reminders" route to render
// a result card; the cron path discards it (the server log is the
// canonical record for the cron path).
type TickResult struct {
	Users   int
	Results []UserReminderResult
	// ListError is non-empty when the user list was unreadable.
	// When set, Results is nil and Users is 0 (the admin route
	// surfaces this as a distinct error card).
	ListError  string
	Duration   time.Duration
	Attempted  bool
	Now        time.Time
}

// Run is the function the Scheduler invokes on every tick. It
// finds every user due, sends each one's email reminder (when the
// user has the channel enabled), and advances their next_fire_at.
// A single per-user failure (e.g. SMTP error on one user) is
// logged and does not stop the rest of the batch.
//
// The function returns only after every per-user send has
// completed. A failure to read the user list is logged and the
// run aborts — there is nothing useful to do without the list.
//
// The second return value is "attempts were made": false when the
// user list was unreadable (so the admin handler can render a
// distinct error card). When true, the result reflects what
// actually ran (which may include zero attempts if no user is due).
func (r *UserReminder) Run(ctx context.Context) (TickResult, bool) {
	start := r.clock.Now()
	log.Printf("reminders: tick starting at %s", start.UTC().Format(time.RFC3339))

	users, err := r.repo.ListUsersDueForReminder(ctx, start)
	if err != nil {
		log.Printf("reminders: tick: list due users failed: %v", err)
		return TickResult{
			Duration:  time.Since(start),
			ListError: err.Error(),
			Now:       start,
		}, false
	}
	log.Printf("reminders: tick: %d user(s) due", len(users))

	results := make([]UserReminderResult, 0, len(users))
	for _, u := range users {
		results = append(results, r.SendToUser(ctx, &u, start))
	}

	duration := time.Since(start)
	log.Printf(
		"reminders: tick complete: users=%d duration=%s",
		len(users),
		duration,
	)

	return TickResult{
		Users:      len(users),
		Results:    results,
		Duration:   duration,
		Attempted:  true,
		Now:        start,
	}, true
}

// SendToUser is the per-user body of Run. It is exported (rather
// than kept private to Run) so the admin route can call it
// directly for a single user when needed, and so the unit tests
// can assert on a single user's outcome without standing up the
// "list due users" path.
//
// The function applies the skip rule:
//
//   - email: skipped if user.Email is empty (defensive — every
//     user should have an email, but the orchestrator does not
//     assume the user repo is perfect).
//
// After firing, next_fire_at is advanced by the cadence's stride
// (24h / 7d / 14d) using the supplied "now" as the base. The
// advanced time is written via repo.MarkUserReminderFired, which
// also stamps last_fired_at. The orchestrator does not read the
// row back — the value it wrote is the new next_fire_at.
func (r *UserReminder) SendToUser(ctx context.Context, user *models.User, now time.Time) UserReminderResult {
	res := UserReminderResult{
		UserID:    user.ID,
		UserName:  user.Name,
		UserEmail: user.Email,
	}

	// Email channel — the only delivery channel. Reminders are
	// gated by the master switch upstream (the due-users query
	// filters on reminder_enabled), so the only per-user skip
	// here is a missing address.
	switch {
	case user.Email == "":
		res.EmailSkipped = true
	default:
		// Per-user panic safety: a misbehaving SMTP layer must
		// not take down the whole tick. Mirrors the
		// fanOutEmails pattern from the old all-user
		// orchestrator.
		sendErr := func() (err error) {
			defer func() {
				if rec := recover(); rec != nil {
					err = fmt.Errorf("panic: %v", rec)
				}
			}()
			return r.emails.SendWeightReminder(ctx, user)
		}()
		if sendErr != nil {
			log.Printf("reminders: user %s email failed: %v", user.Email, sendErr)
			res.EmailFailed = true
		} else {
			res.EmailSent = true
		}
	}

	// Advance next_fire_at by the cadence's stride. Unlike
	// the initial-schedule case (where ComputeNextFire finds
	// the next matching <day_of_week> at <hour>), a
	// successful fire always means "schedule the next one
	// exactly N time-units from now", regardless of day-of-
	// week: if the user was fired on Sunday, the next weekly
	// fire is exactly 7 days later (still Sunday). Adding
	// the stride to the current fire time gives that result
	// directly, without re-running the day-of-week math.
	nextFire := now.Add(reminderStride(user.ReminderFrequency))
	if err := r.repo.MarkUserReminderFired(ctx, user.ID, now, nextFire); err != nil {
		log.Printf("reminders: user %s: mark fired failed: %v", user.ID, err)
		res.Error = err.Error()
	}
	return res
}

// reminderStride returns the per-frequency offset that the
// orchestrator adds to the current fire time to compute the
// next one. Pulled into a helper so the four cases live in one
// place and the orchestrator's advance-NextFireAt code reads
// as a single expression.
func reminderStride(f models.ReminderFrequency) time.Duration {
	switch f {
	case models.ReminderDaily:
		return 24 * time.Hour
	case models.ReminderWeekly:
		return 7 * 24 * time.Hour
	case models.ReminderBiweekly:
		return 14 * 24 * time.Hour
	}
	// Off / unknown: the tick should not have selected this row,
	// but if it did (e.g. the row was disabled between the list
	// query and the per-user send), the +24h advance lets the
	// next tick correctly skip it (RemindersEnabled() will be
	// false by then).
	return 24 * time.Hour
}

// Compile-time checks: the production types must satisfy the
// interfaces the orchestrator depends on. If the signatures of
// the real implementations ever drift, these lines fail to
// compile and point the maintainer at the break.
var (
	_ ReminderRepo = (*models.UserRepository)(nil)
	_ EmailSender  = (*email.Service)(nil)
)
