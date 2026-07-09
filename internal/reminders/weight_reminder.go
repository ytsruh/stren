package reminders

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"stren/internal/email"
	"stren/internal/models"
	"stren/internal/push"
)

// UserLister is the data-access surface the WeightReminder
// needs to discover which users to email. Defined as an
// interface (rather than depending on the concrete
// *models.UserAdminRepository) so the orchestrator can be
// unit-tested with an in-memory fake, and so the model
// layer stays free of reminder-specific dependencies.
type UserLister interface {
	ListUsers(ctx context.Context) ([]models.User, error)
}

// EmailSender is the narrow contract the orchestrator
// depends on for the per-user email send. Mirrors the
// shape of the push.Sender interface: a single method
// scoped to one message, so the orchestrator can fire
// it from a goroutine without sharing state.
//
// Returning an error rather than panicking means a single
// SMTP failure on one user is logged and skipped, not
// fatal for the whole batch.
type EmailSender interface {
	SendWeightReminder(ctx context.Context, user *models.User) error
}

// PushBroadcaster is the narrow contract the orchestrator
// depends on for the push fan-out. Reuses the existing
// push.Service.Broadcast shape (one call sends to every
// subscription in the system) so the reminder does not
// need any push-protocol knowledge of its own.
type PushBroadcaster interface {
	Broadcast(ctx context.Context, msg push.Message) (push.BroadcastResult, error)
}

// Clock is the time source the orchestrator uses for any
// "what time is it" decision. Injected as an interface so
// tests can pin the time and assert on the exact cron
// tick; in production, *RealClock is used.
type Clock interface {
	Now() time.Time
}

// RealClock returns the wall clock. The default Clock the
// orchestrator uses; tests inject a fixed clock to keep
// assertions deterministic.
type RealClock struct{}

// Now returns time.Now(). Lives on a type (rather than as
// a package-level function) so it satisfies Clock without
// adapters.
func (RealClock) Now() time.Time { return time.Now() }

// defaultMaxEmailWorkers bounds the goroutine pool that
// fans out per-user emails. The value matches the
// push.Service default so the two pipelines behave
// similarly under load: a sudden user-count spike cannot
// exhaust file descriptors or starve other handlers.
const defaultMaxEmailWorkers = 10

// WeightReminder orchestrates the weekly "log your
// weight" reminder: it fans out one email per user
// (via EmailSender) and a single push broadcast to every
// subscribed device (via PushBroadcaster). It is
// constructed once in main and invoked by the Scheduler
// on its cron tick.
//
// The orchestrator is stateless and safe for concurrent
// use: each Run call constructs its own worker pool
// internally and tears it down before returning. A
// second Run call (e.g. an admin "send now" button in
// the future) can overlap with an in-flight one without
// trampling shared state.
type WeightReminder struct {
	users    UserLister
	emails   EmailSender
	push     PushBroadcaster
	clock    Clock
	maxEmail int
}

// WeightReminderConfig groups the optional knobs on
// WeightReminder. Zero values fall back to package
// defaults (RealClock, defaultMaxEmailWorkers).
type WeightReminderConfig struct {
	// MaxEmailWorkers bounds the per-user email fan-out.
	// 0 or negative → defaultMaxEmailWorkers.
	MaxEmailWorkers int
}

// NewWeightReminder returns a WeightReminder bound to
// the given dependencies. users, emails, and push are
// required (nil returns an error so a missing
// dependency fails at startup, not at the first tick).
// clock is optional; nil falls back to RealClock.
func NewWeightReminder(users UserLister, emails EmailSender, push PushBroadcaster, cfg WeightReminderConfig) (*WeightReminder, error) {
	if users == nil {
		return nil, fmt.Errorf("reminders: NewWeightReminder: users is nil")
	}
	if emails == nil {
		return nil, fmt.Errorf("reminders: NewWeightReminder: emails is nil")
	}
	if push == nil {
		return nil, fmt.Errorf("reminders: NewWeightReminder: push is nil")
	}

	max := cfg.MaxEmailWorkers
	if max <= 0 {
		max = defaultMaxEmailWorkers
	}

	return &WeightReminder{
		users:    users,
		emails:   emails,
		push:     push,
		clock:    RealClock{},
		maxEmail: max,
	}, nil
}

// RunResult is the structured outcome of a single
// WeightReminder.Run call. It is consumed by the admin
// "send weight reminder" route to render a result card;
// the cron path discards it. The orchestrator also
// writes a single matching log line so the server log
// remains the canonical record of every run.
//
// Field semantics:
//   - Users:                total number of users the
//                            orchestrator attempted to
//                            email. 0 when the user
//                            list was unreadable
//                            (ListError set, attempted=false).
//   - EmailsSent:           emails that returned nil
//                            from the EmailSender.
//   - EmailsFailed:         emails that returned a
//                            non-nil error or panicked.
//   - EmailsFailedAddresses: the email addresses of the
//                            failed sends, in the order
//                            they were observed. Useful
//                            for showing the operator
//                            exactly who did not get
//                            the email.
//   - PushSent / PushDeleted / PushFailed: copied
//                            verbatim from
//                            push.BroadcastResult.
//   - PushError:            non-empty when the push
//                            broadcast itself errored
//                            (the store was unreadable).
//                            Per-subscription failures
//                            are not surfaced here — they
//                            are in PushFailed and
//                            logged by push.Service.
//   - ListError:            non-empty when the user
//                            list was unreadable.
//   - Duration:             wall-clock time spent in
//                            Run. Reported for parity
//                            with the server log line.
type RunResult struct {
	Users                 int
	EmailsSent            int
	EmailsFailed          int
	EmailsFailedAddresses []string
	PushSent              int
	PushDeleted           int
	PushFailed            int
	PushError             string
	ListError             string
	Duration              time.Duration
}

// fanOutSummary is the internal return value of
// fanOutEmails. Held in a small struct so the public
// RunResult stays focused on what the admin UI cares
// about and the worker-internal details (per-user
// success bool) stay out of the public API.
type fanOutSummary struct {
	sent      int
	failed    int
	failAddrs []string
}

// Run is the function the Scheduler invokes on each
// tick (the cron job path discards the result) and
// what the admin "send weight reminder" route
// invokes on demand. It fans out the email and push
// for the current week and logs a single summary
// line so the operator can see the run's outcome
// in the server log without any admin UI.
//
// The function returns only after both the email
// pool and the push broadcast have completed. A push
// failure does not undo emails already sent — the
// two pipelines are independent. Email failures on
// individual users are logged inside the worker
// but do not stop the rest of the batch. A failure
// to read the user list is logged and the run
// aborts — there is nothing useful to do without
// the user list.
//
// The second return value is "attempts were made":
// false when the user list was unreadable (so the
// admin handler can render a distinct error card).
// When true, the result reflects what actually ran
// (which may include zero attempts if the user
// table is empty).
func (r *WeightReminder) Run(ctx context.Context) (RunResult, bool) {
	start := r.clock.Now()
	log.Printf("reminders: weight reminder run starting at %s", start.UTC().Format(time.RFC3339))

	allUsers, err := r.users.ListUsers(ctx)
	if err != nil {
		log.Printf("reminders: weight reminder: list users failed: %v", err)
		return RunResult{
			Duration:  time.Since(start),
			ListError: err.Error(),
		}, false
	}
	log.Printf("reminders: weight reminder: %d user(s) to email", len(allUsers))

	emailSummary := r.fanOutEmails(ctx, allUsers)

	pushResult, pushErr := r.push.Broadcast(ctx, buildWeightReminderPushMessage())
	var pushErrStr string
	if pushErr != nil {
		// The push service is the only place that returns a
		// non-nil error for "the store itself was unreadable".
		// Per-subscription failures are reported in
		// BroadcastResult and logged by push.Service, not by
		// us.
		log.Printf("reminders: weight reminder push broadcast error: %v", pushErr)
		pushErrStr = pushErr.Error()
	}

	duration := time.Since(start)
	log.Printf(
		"reminders: weight reminder complete: users=%d emails_failed=%d push: sent=%d deleted=%d failed=%d duration=%s",
		len(allUsers),
		emailSummary.failed,
		pushResult.Sent,
		pushResult.Deleted,
		pushResult.Failed,
		duration,
	)

	return RunResult{
		Users:                 len(allUsers),
		EmailsSent:            emailSummary.sent,
		EmailsFailed:          emailSummary.failed,
		EmailsFailedAddresses: emailSummary.failAddrs,
		PushSent:              pushResult.Sent,
		PushDeleted:           pushResult.Deleted,
		PushFailed:            pushResult.Failed,
		PushError:             pushErrStr,
		Duration:              duration,
	}, true
}

// fanOutEmails sends the weight-reminder email to every
// user in users, bounded by r.maxEmail concurrent
// workers. Returns a fanOutSummary the caller folds
// into the public RunResult.
//
// Each per-user send runs in its own goroutine with a
// recover: a panic in the SMTP path (e.g. a nil
// dereference in a future refactor) is logged and
// counted as a failure rather than crashing the
// process. The same pattern is used by the welcome
// email fire-and-forget in controllers.AuthController.
//
// Failed addresses are collected in the order the
// workers observed them (which is non-deterministic
// under load, but the list is just for the operator
// to spot patterns). The slice is built without
// locking because only the orchestrator's single
// consumer reads it after the channel is closed.
func (r *WeightReminder) fanOutEmails(ctx context.Context, users []models.User) fanOutSummary {
	if len(users) == 0 {
		return fanOutSummary{}
	}

	workers := r.maxEmail
	if workers > len(users) {
		workers = len(users)
	}

	type result struct {
		email string
		err   error
	}

	jobs := make(chan models.User, len(users))
	results := make(chan result, len(users))

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for u := range jobs {
				if ctx.Err() != nil {
					results <- result{email: u.Email, err: ctx.Err()}
					continue
				}
				sendErr := func() (err error) {
					defer func() {
						if r := recover(); r != nil {
							err = fmt.Errorf("panic: %v", r)
						}
					}()
					return r.emails.SendWeightReminder(ctx, &u)
				}()
				if sendErr != nil {
					log.Printf("reminders: weight reminder email to %s failed: %v", u.Email, sendErr)
				}
				results <- result{email: u.Email, err: sendErr}
			}
		}()
	}

	for _, u := range users {
		jobs <- u
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var summary fanOutSummary
	for res := range results {
		if res.err == nil {
			summary.sent++
			continue
		}
		summary.failed++
		summary.failAddrs = append(summary.failAddrs, res.email)
	}
	return summary
}

// buildWeightReminderPushMessage returns the fixed push
// payload sent alongside the emails. Title and body are
// short on purpose — the notification is a tap-to-act
// prompt, not a marketing message. URL is /weight/new
// so a tap lands directly on the new-entry form (where
// the photo upload already lives).
//
// Kept as a tiny helper so the copy lives in one place
// and is easy to tweak alongside the email copy.
func buildWeightReminderPushMessage() push.Message {
	return push.Message{
		Title: "Sunday weigh-in",
		Body:  "Tap to log your weight for the week.",
		URL:   "/weight/new",
	}
}

// Compile-time checks: the production types must satisfy
// the interfaces the orchestrator depends on. If the
// signatures of the real implementations ever drift,
// these lines fail to compile and point the maintainer
// at the break.
var (
	_ UserLister      = (*models.UserAdminRepository)(nil)
	_ EmailSender     = (*email.Service)(nil)
	_ PushBroadcaster = (*push.Service)(nil)
)
