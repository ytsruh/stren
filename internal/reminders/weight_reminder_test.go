package reminders

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"stren/internal/models"
	"stren/internal/push"
)

// --- Test doubles for the orchestrator's dependencies ---

// fakeUserLister is a small in-memory UserLister for tests.
// Thread-safe so the orchestrator's concurrent reads do not
// race the test's setup.
type fakeUserLister struct {
	mu    sync.Mutex
	users []models.User
	err   error
}

func (f *fakeUserLister) ListUsers(_ context.Context) ([]models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	out := make([]models.User, len(f.users))
	copy(out, f.users)
	return out, nil
}

// fakeEmailSender records every per-user send the orchestrator
// asks for. errFor lets the test make one specific user fail;
// errAll makes every send fail.
type fakeEmailSender struct {
	mu        sync.Mutex
	calls     []*models.User
	errAll    error
	errFor    map[string]error
	delayFunc func(*models.User)
}

func (f *fakeEmailSender) SendWeightReminder(_ context.Context, u *models.User) error {
	if f.delayFunc != nil {
		f.delayFunc(u)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, u)
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

// fakePushBroadcaster records every broadcast call and returns
// a pre-canned BroadcastResult / error so the test can simulate
// a successful run, a partially-failed run, or an unreadable
// store.
type fakePushBroadcaster struct {
	mu     sync.Mutex
	calls  []push.Message
	result push.BroadcastResult
	err    error
}

func (f *fakePushBroadcaster) Broadcast(_ context.Context, msg push.Message) (push.BroadcastResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, msg)
	return f.result, f.err
}

func (f *fakePushBroadcaster) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// fixedClock returns a constant time. Lets the test pin
// "now" without touching the system clock.
type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

// newTestReminder is a small constructor that wires the four
// fakes into a WeightReminder. Tests override the fakes
// before calling Run.
func newTestReminder(t *testing.T) (*WeightReminder, *fakeUserLister, *fakeEmailSender, *fakePushBroadcaster) {
	t.Helper()
	users := &fakeUserLister{}
	emails := &fakeEmailSender{}
	pushB := &fakePushBroadcaster{}
	r, err := NewWeightReminder(users, emails, pushB, WeightReminderConfig{MaxEmailWorkers: 4})
	if err != nil {
		t.Fatalf("NewWeightReminder: %v", err)
	}
	// Pin the clock to a Sunday morning so the run summary is
	// deterministic. The orchestrator does not currently branch
	// on day-of-week (the cron wrapper does), but the value
	// appears in the log line and tests can assert on the
	// formatted output if they want to.
	r.clock = fixedClock{t: time.Date(2026, 7, 5, 9, 0, 0, 0, time.UTC)}
	return r, users, emails, pushB
}

// --- Tests ---

func TestNewWeightReminder_RejectsNilDependencies(t *testing.T) {
	// nil dependencies must fail at construction, not at the
	// first tick. A nil user repo, for example, would NPE deep
	// inside the orchestrator.
	if _, err := NewWeightReminder(nil, &fakeEmailSender{}, &fakePushBroadcaster{}, WeightReminderConfig{}); err == nil {
		t.Error("expected error for nil users")
	}
	if _, err := NewWeightReminder(&fakeUserLister{}, nil, &fakePushBroadcaster{}, WeightReminderConfig{}); err == nil {
		t.Error("expected error for nil emails")
	}
	if _, err := NewWeightReminder(&fakeUserLister{}, &fakeEmailSender{}, nil, WeightReminderConfig{}); err == nil {
		t.Error("expected error for nil push")
	}
}

func TestWeightReminder_Run_EmptyUserList(t *testing.T) {
	// No users → no email sends. The push broadcast still
	// runs (it naturally short-circuits inside
	// push.Service.Broadcast when there are no subscriptions,
	// returning a zero result). That is by design: the push
	// pipeline is independent of the email pipeline, and a
	// future "log a user out" code path that purges
	// subscriptions but leaves the user row would still get
	// the reminder email. Keeping the push call unconditional
	// makes the orchestrator simpler and the two pipelines
	// symmetric.
	r, _, emails, pushB := newTestReminder(t)
	res, attempted := r.Run(context.Background())
	if !attempted {
		t.Error("attempted = false, want true (list succeeded)")
	}
	if res.Users != 0 {
		t.Errorf("result.Users = %d, want 0", res.Users)
	}
	if got := emails.callCount(); got != 0 {
		t.Errorf("email calls = %d, want 0", got)
	}
	if got := pushB.callCount(); got != 1 {
		t.Errorf("push broadcasts = %d, want 1 (unconditional)", got)
	}
}

func TestWeightReminder_Run_FansOutOneEmailPerUser(t *testing.T) {
	// Each user must get exactly one email send. The push
	// broadcast happens once (to all subscribers) regardless
	// of user count. The result struct must report the same
	// counts the admin UI will render.
	r, users, emails, pushB := newTestReminder(t)
	users.users = []models.User{
		{ID: "u1", Name: "Alice", Email: "alice@example.com"},
		{ID: "u2", Name: "Bob", Email: "bob@example.com"},
		{ID: "u3", Name: "Carol", Email: "carol@example.com"},
	}
	pushB.result = push.BroadcastResult{Sent: 7, Total: 7}

	res, attempted := r.Run(context.Background())
	if !attempted {
		t.Fatal("attempted = false, want true")
	}
	if got := emails.callCount(); got != 3 {
		t.Errorf("email calls = %d, want 3", got)
	}
	if got := pushB.callCount(); got != 1 {
		t.Errorf("push broadcasts = %d, want 1", got)
	}
	if res.Users != 3 {
		t.Errorf("result.Users = %d, want 3", res.Users)
	}
	if res.EmailsSent != 3 {
		t.Errorf("result.EmailsSent = %d, want 3", res.EmailsSent)
	}
	if res.EmailsFailed != 0 {
		t.Errorf("result.EmailsFailed = %d, want 0", res.EmailsFailed)
	}
	if res.PushSent != 7 {
		t.Errorf("result.PushSent = %d, want 7", res.PushSent)
	}
}

func TestWeightReminder_Run_PushMessageContents(t *testing.T) {
	// The push payload must be the agreed-on reminder copy
	// (title, body, URL). A future copy change is a one-line
	// edit; this test is the tripwire for accidental drift.
	r, users, _, pushB := newTestReminder(t)
	users.users = []models.User{{ID: "u1", Name: "Alice", Email: "alice@example.com"}}

	r.Run(context.Background())
	if got := pushB.callCount(); got != 1 {
		t.Fatalf("push broadcasts = %d, want 1", got)
	}
	got := pushB.calls[0]
	if got.Title != "Sunday weigh-in" {
		t.Errorf("push title = %q, want %q", got.Title, "Sunday weigh-in")
	}
	if got.URL != "/weight/new" {
		t.Errorf("push url = %q, want %q", got.URL, "/weight/new")
	}
	if got.Body == "" {
		t.Error("push body is empty")
	}
}

func TestWeightReminder_Run_FailingEmailDoesNotPoisonBatch(t *testing.T) {
	// A single SMTP failure on one user must not stop the
	// other users from receiving their emails. The failing
	// user's send is logged and the address surfaces in
	// RunResult.EmailsFailedAddresses so the admin UI can
	// show exactly who did not get the email.
	r, users, emails, _ := newTestReminder(t)
	users.users = []models.User{
		{ID: "u1", Name: "Alice", Email: "alice@example.com"},
		{ID: "u2", Name: "Bob", Email: "bob@example.com"},
		{ID: "u3", Name: "Carol", Email: "carol@example.com"},
	}
	emails.errFor = map[string]error{
		"bob@example.com": errors.New("smtp 421"),
	}

	res, attempted := r.Run(context.Background())
	if !attempted {
		t.Fatal("attempted = false, want true")
	}
	// All three users were attempted; only Bob's send
	// returned an error.
	if got := emails.callCount(); got != 3 {
		t.Errorf("email calls = %d, want 3 (every user attempted)", got)
	}
	if res.EmailsSent != 2 {
		t.Errorf("result.EmailsSent = %d, want 2", res.EmailsSent)
	}
	if res.EmailsFailed != 1 {
		t.Errorf("result.EmailsFailed = %d, want 1", res.EmailsFailed)
	}
	if len(res.EmailsFailedAddresses) != 1 || res.EmailsFailedAddresses[0] != "bob@example.com" {
		t.Errorf("result.EmailsFailedAddresses = %v, want [bob@example.com]", res.EmailsFailedAddresses)
	}
}

func TestWeightReminder_Run_ListErrorAborts(t *testing.T) {
	// A failure to read the user list is the only error
	// path that short-circuits Run. Nothing was attempted
	// yet, so there is nothing to fall through to. The
	// orchestrator logs and returns; the test asserts
	// that no downstream work happened, and that the
	// admin UI can distinguish "list failed" from
	// "list succeeded with zero users" via the bool.
	r, users, emails, pushB := newTestReminder(t)
	users.err = errors.New("db down")

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
	if got := pushB.callCount(); got != 0 {
		t.Errorf("push broadcasts = %d, want 0 when list failed", got)
	}
}

func TestWeightReminder_Run_PushBroadcastErrorIsLoggedNotFatal(t *testing.T) {
	// A push broadcast error is a transport-level problem;
	// the emails already sent cannot be un-sent. Run logs
	// the error and returns. The result's PushError field
	// surfaces the message so the admin UI can render it.
	r, users, _, pushB := newTestReminder(t)
	users.users = []models.User{{ID: "u1", Name: "Alice", Email: "alice@example.com"}}
	pushB.err = errors.New("push service unreachable")

	res, attempted := r.Run(context.Background())
	if !attempted {
		t.Fatal("attempted = false, want true")
	}
	if res.PushError != "push service unreachable" {
		t.Errorf("result.PushError = %q, want %q", res.PushError, "push service unreachable")
	}
}

func TestWeightReminder_Run_ContextCancellation(t *testing.T) {
	// A pre-cancelled context must not deadlock or panic.
	// Workers see ctx.Err() and report it back through the
	// same channel as a normal send failure; the
	// orchestrator counts it and moves on.
	r, users, _, _ := newTestReminder(t)
	users.users = []models.User{
		{ID: "u1", Name: "Alice", Email: "alice@example.com"},
		{ID: "u2", Name: "Bob", Email: "bob@example.com"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// The fan-out still attempts the jobs (the workers
	// read jobs before checking ctx), so callCount may be
	// 0..N depending on scheduling. The contract we care
	// about: Run does not deadlock and does not panic.
	r.Run(ctx)
}

func TestNewCronScheduler_RejectsBadSpec(t *testing.T) {
	// A typo in the cron spec must fail at startup, not
	// silently never fire. cmd/main.go passes the literal
	// "0 9 * * 0" — any regression that breaks parsing
	// would surface here before deploy.
	if _, err := NewCronScheduler("not a cron spec", time.UTC, func(_ context.Context) {}); err == nil {
		t.Error("expected error for unparseable spec")
	}
	if _, err := NewCronScheduler("0 9 * * 0", time.UTC, nil); err == nil {
		t.Error("expected error for nil job")
	}
}

func TestNewCronScheduler_HappyPathStartStop(t *testing.T) {
	// A valid spec + non-nil job must return a working
	// scheduler. Start/Stop are both safe to call and
	// idempotent — the production wrapper relies on this
	// for clean shutdown.
	called := make(chan struct{}, 1)
	sched, err := NewCronScheduler("0 9 * * 0", time.UTC, func(_ context.Context) {
		called <- struct{}{}
	})
	if err != nil {
		t.Fatalf("NewCronScheduler: %v", err)
	}
	sched.Start()
	// Stop without a fire: nothing in `called` is fine; we
	// are exercising the lifecycle, not the schedule.
	sched.Stop()
	// Idempotency: a second Stop is a no-op.
	sched.Stop()
	// And the channel was never written to (we are not
	// waiting for a tick).
	select {
	case <-called:
		t.Error("job ran on a never-tick'd schedule")
	default:
	}
}
