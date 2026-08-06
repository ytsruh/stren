package push

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"testing"
)

// fakeStore is a small in-memory SubscriptionStore for service tests.
// The three methods the service calls are the only ones we need.
type fakeStore struct {
	mu        sync.Mutex
	subs      []Subscription
	byUser    map[string][]Subscription
	deleted   []string
	listErr   error
	deleteErr error
}

func (s *fakeSender) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func (f *fakeStore) ListAll(_ context.Context) ([]Subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]Subscription, len(f.subs))
	copy(out, f.subs)
	return out, nil
}

func (f *fakeStore) ListForUser(_ context.Context, userID string) ([]Subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	src := f.byUser[userID]
	out := make([]Subscription, len(src))
	copy(out, src)
	return out, nil
}

func (f *fakeStore) DeleteByEndpoint(_ context.Context, endpoint string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, endpoint)
	for i, s := range f.subs {
		if s.Endpoint == endpoint {
			f.subs = append(f.subs[:i], f.subs[i+1:]...)
			break
		}
	}
	for uid, list := range f.byUser {
		for i, s := range list {
			if s.Endpoint == endpoint {
				f.byUser[uid] = append(list[:i], list[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (f *fakeStore) deletedEndpoints() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.deleted))
	copy(out, f.deleted)
	return out
}

// fakeSender is a tiny Sender implementation that returns outcomes
// from a per-endpoint status map. The service is the unit under test
// here; the sender is just plumbing. This keeps the service tests
// fast (no real crypto, no HTTP) and deterministic.
type fakeSender struct {
	mu         sync.Mutex
	statuses   map[string]int // keyed by endpoint
	errs       map[string]error
	defaultErr error
	calls      int
}

func (s *fakeSender) Send(_ context.Context, sub Subscription, _ Message) SendOutcome {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	out := SendOutcome{Subscription: sub}
	if e, ok := s.errs[sub.Endpoint]; ok {
		out.Error = e
		return out
	}
	if s.defaultErr != nil {
		out.Error = s.defaultErr
		return out
	}
	status, ok := s.statuses[sub.Endpoint]
	if !ok {
		status = http.StatusCreated
	}
	out.Status = status
	switch {
	case status >= 200 && status < 300:
		return out
	case status == http.StatusNotFound || status == http.StatusGone:
		out.Deleted = true
		out.Error = errFake("subscription gone")
		return out
	default:
		out.Error = errFake("unexpected status")
		return out
	}
}

// serviceHarness wires a fakeStore + fakeSender + Service for the
// tests below. Each test pre-fills the store/sender and asserts on
// the resulting BroadcastResult.
type serviceHarness struct {
	store   *fakeStore
	sender  *fakeSender
	service *Service
}

func newServiceHarness() *serviceHarness {
	store := &fakeStore{}
	sender := &fakeSender{
		statuses: map[string]int{},
		errs:     map[string]error{},
	}
	svc := NewService(sender, store, ServiceConfig{MaxWorkers: 4})
	return &serviceHarness{store: store, sender: sender, service: svc}
}

func (h *serviceHarness) addSub(endpoint string) {
	h.store.subs = append(h.store.subs, Subscription{
		Endpoint: endpoint,
		P256dh:   "x", Auth: "y",
	})
}

// addSubForUser seeds a subscription under a user id. The
// fakeStore's ListForUser reads from byUser; the fake
// ListAll still returns the full slice so admin-broadcast
// tests keep working unchanged.
func (h *serviceHarness) addSubForUser(userID, endpoint string) {
	if h.store.byUser == nil {
		h.store.byUser = map[string][]Subscription{}
	}
	h.store.byUser[userID] = append(h.store.byUser[userID], Subscription{
		Endpoint: endpoint,
		P256dh:   "x", Auth: "y",
	})
	// Keep the global subs list in sync so any test that
	// also calls Broadcast sees the same shape.
	h.store.subs = append(h.store.subs, Subscription{
		Endpoint: endpoint,
		P256dh:   "x", Auth: "y",
	})
}

func TestService_Broadcast_EmptyList(t *testing.T) {
	h := newServiceHarness()
	res, err := h.service.Broadcast(context.Background(), Message{Title: "x", Body: "y"})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if res.Total != 0 || res.Sent != 0 {
		t.Fatalf("expected zero result, got %+v", res)
	}
}

func TestService_Broadcast_AllSucceed(t *testing.T) {
	h := newServiceHarness()
	for _, e := range []string{"https://push.example/a", "https://push.example/b", "https://push.example/c"} {
		h.addSub(e)
	}
	res, err := h.service.Broadcast(context.Background(), Message{Title: "x", Body: "y"})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if res.Total != 3 || res.Sent != 3 || res.Failed != 0 {
		t.Fatalf("unexpected counts: %+v", res)
	}
}

func TestService_Broadcast_DeadSubscriptionsArePruned(t *testing.T) {
	h := newServiceHarness()
	h.addSub("https://push.example/good")
	h.addSub("https://push.example/gone")
	h.sender.statuses["https://push.example/gone"] = http.StatusGone

	res, err := h.service.Broadcast(context.Background(), Message{Title: "x", Body: "y"})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if res.Sent != 1 {
		t.Fatalf("expected 1 sent, got %d", res.Sent)
	}
	if res.Deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", res.Deleted)
	}
	deleted := h.store.deletedEndpoints()
	sort.Strings(deleted)
	if len(deleted) != 1 || deleted[0] != "https://push.example/gone" {
		t.Fatalf("expected only /gone to be deleted, got %v", deleted)
	}
}

func TestService_Broadcast_5xxCountsAsFailed(t *testing.T) {
	h := newServiceHarness()
	h.addSub("https://push.example/boom")
	h.sender.statuses["https://push.example/boom"] = http.StatusInternalServerError

	res, err := h.service.Broadcast(context.Background(), Message{Title: "x", Body: "y"})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if res.Failed != 1 || res.Sent != 0 {
		t.Fatalf("expected 1 failed, got %+v", res)
	}
	if res.Deleted != 0 {
		t.Fatalf("5xx should not prune, got %d deleted", res.Deleted)
	}
}

func TestService_Broadcast_StoreErrorIsPropagated(t *testing.T) {
	h := newServiceHarness()
	h.store.listErr = errFake("db down")
	if _, err := h.service.Broadcast(context.Background(), Message{}); err == nil {
		t.Fatal("expected error from store")
	}
}

func TestService_Broadcast_ContextCancelled(t *testing.T) {
	h := newServiceHarness()
	for i := 0; i < 50; i++ {
		h.addSub("https://push.example/" + string(rune('a'+i%26)))
	}
	// Sender that respects context cancellation.
	h.sender.defaultErr = nil
	cancellingSender := cancellingSenderFunc(func(sub Subscription, msg Message) SendOutcome {
		return SendOutcome{Subscription: sub, Error: context.Canceled}
	})
	svc := NewService(cancellingSender, h.store, ServiceConfig{MaxWorkers: 1})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = svc.Broadcast(ctx, Message{})
	// Smoke test — the test passes as long as nothing panics.
}

type cancellingSenderFunc func(Subscription, Message) SendOutcome

func (f cancellingSenderFunc) Send(_ context.Context, sub Subscription, msg Message) SendOutcome {
	return f(sub, msg)
}

func TestService_Broadcast_MixedOutcomes(t *testing.T) {
	h := newServiceHarness()
	h.addSub("https://push.example/ok1")
	h.addSub("https://push.example/ok2")
	h.addSub("https://push.example/410")
	h.addSub("https://push.example/500")
	h.sender.statuses["https://push.example/410"] = http.StatusGone
	h.sender.statuses["https://push.example/500"] = http.StatusInternalServerError

	res, err := h.service.Broadcast(context.Background(), Message{Title: "x", Body: "y"})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if res.Sent != 2 || res.Deleted != 1 || res.Failed != 1 {
		t.Fatalf("unexpected counts: %+v", res)
	}
	if res.Total != 4 {
		t.Fatalf("expected total=4, got %d", res.Total)
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected at least one error message")
	}
}

func TestService_Broadcast_DeleteErrorIsCountedButNotFatal(t *testing.T) {
	h := newServiceHarness()
	h.addSub("https://push.example/gone")
	h.sender.statuses["https://push.example/gone"] = http.StatusGone
	h.store.deleteErr = errFake("db error on delete")

	res, err := h.service.Broadcast(context.Background(), Message{Title: "x", Body: "y"})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	// The push service still told us the device is gone; the
	// service should still count it as Deleted even if the local
	// prune failed. The next broadcast will retry the delete.
	if res.Deleted != 1 {
		t.Fatalf("expected Deleted=1, got %d", res.Deleted)
	}
}

// --- BroadcastToUser: per-user fan-out for the reminder orchestrator ---

func TestService_BroadcastToUser_OnlyTargetsUserSubs(t *testing.T) {
	// Two users with two subscriptions each. The per-user
	// broadcast must only call the sender for the target
	// user's subscriptions — not every subscription in the
	// system (which is what the admin broadcast does).
	// This is the tripwire: a regression that swapped
	// ListForUser for ListAll would leak a different
	// user's reminders to the wrong device.
	h := newServiceHarness()
	h.addSubForUser("u1", "https://push.example/u1-a")
	h.addSubForUser("u1", "https://push.example/u1-b")
	h.addSubForUser("u2", "https://push.example/u2-a")
	h.addSubForUser("u2", "https://push.example/u2-b")

	res, err := h.service.BroadcastToUser(context.Background(), "u1", Message{Title: "x", Body: "y"})
	if err != nil {
		t.Fatalf("BroadcastToUser: %v", err)
	}
	if res.Total != 2 {
		t.Errorf("Total = %d, want 2 (only u1's subs)", res.Total)
	}
	if res.Sent != 2 {
		t.Errorf("Sent = %d, want 2", res.Sent)
	}
	if h.sender.callCount() != 2 {
		t.Errorf("sender calls = %d, want 2", h.sender.callCount())
	}
}

func TestService_BroadcastToUser_NoSubsReturnsEmptyResult(t *testing.T) {
	// A user with no subscriptions (or a wrong user id)
	// returns a zero result with no error. The orchestrator
	// translates this to "push skipped, no active
	// subscriptions" rather than "push failed".
	h := newServiceHarness()
	h.addSubForUser("u1", "https://push.example/u1-a")

	res, err := h.service.BroadcastToUser(context.Background(), "unknown", Message{Title: "x", Body: "y"})
	if err != nil {
		t.Fatalf("BroadcastToUser: %v", err)
	}
	if res.Total != 0 {
		t.Errorf("Total = %d, want 0", res.Total)
	}
	if h.sender.callCount() != 0 {
		t.Errorf("sender calls = %d, want 0 (no subs for unknown user)", h.sender.callCount())
	}
}

func TestService_BroadcastToUser_StoreErrorIsPropagated(t *testing.T) {
	// The store read failing is a transport-level error
	// the orchestrator must surface verbatim (so the
	// per-user result shows "push failed" instead of
	// "push skipped").
	h := newServiceHarness()
	h.store.listErr = errFake("db down")
	if _, err := h.service.BroadcastToUser(context.Background(), "u1", Message{}); err == nil {
		t.Fatal("expected error from store")
	}
}
