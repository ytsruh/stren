package controllers

import (
	"context"
	"errors"
	"sync"
	"testing"

	"stren/internal/models"
	"stren/internal/push"
)

// mockPushRepo is a hand-rolled fake of the PushSubscriptionRepo
// interface. We keep it in this file (not exported) because the
// controller tests are the only consumer.
type mockPushRepo struct {
	mu      sync.Mutex
	rows    map[string]models.PushSubscription // keyed by endpoint
	listErr error
}

func newMockPushRepo() *mockPushRepo {
	return &mockPushRepo{rows: map[string]models.PushSubscription{}}
}

func (m *mockPushRepo) UpsertForUser(_ context.Context, userID string, sub models.PushSubscription) (*models.PushSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.rows[sub.Endpoint]; ok {
		existing.P256dh = sub.P256dh
		existing.Auth = sub.Auth
		m.rows[sub.Endpoint] = existing
		out := existing
		return &out, nil
	}
	row := sub
	row.UserID = userID
	m.rows[sub.Endpoint] = row
	out := row
	return &out, nil
}

func (m *mockPushRepo) ListForUser(_ context.Context, userID string) ([]models.PushSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []models.PushSubscription{}
	for _, r := range m.rows {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockPushRepo) ListAll(_ context.Context) ([]models.PushSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]models.PushSubscription, 0, len(m.rows))
	for _, r := range m.rows {
		out = append(out, r)
	}
	return out, nil
}

func (m *mockPushRepo) DeleteByEndpoint(_ context.Context, endpoint string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rows, endpoint)
	return nil
}

func (m *mockPushRepo) CountForUser(_ context.Context, userID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for _, r := range m.rows {
		if r.UserID == userID {
			n++
		}
	}
	return n, nil
}

// Compile-time check.
var _ models.PushSubscriptionRepo = (*mockPushRepo)(nil)

func TestPushController_Subscribe_HappyPath(t *testing.T) {
	repo := newMockPushRepo()
	pc := NewPushController(repo)
	in := SubscribeInput{
		Endpoint: "https://push.example/abc",
		P256dh:   "x", Auth: "y",
	}
	if err := pc.Subscribe(context.Background(), "user-1", in); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if _, ok := repo.rows[in.Endpoint]; !ok {
		t.Fatal("expected row to be stored")
	}
}

func TestPushController_Subscribe_RejectsEmptyFields(t *testing.T) {
	pc := NewPushController(newMockPushRepo())
	if err := pc.Subscribe(context.Background(), "u", SubscribeInput{Endpoint: "", P256dh: "x", Auth: "y"}); !errors.Is(err, ErrPushSubscriptionMissingFields) {
		t.Fatalf("expected ErrPushSubscriptionMissingFields, got %v", err)
	}
}

func TestPushController_Unsubscribe_HappyPath(t *testing.T) {
	repo := newMockPushRepo()
	pc := NewPushController(repo)
	// Seed the row directly.
	repo.rows["https://push.example/abc"] = models.PushSubscription{Endpoint: "https://push.example/abc", UserID: "u"}
	if err := pc.Unsubscribe(context.Background(), "https://push.example/abc"); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}
	if _, ok := repo.rows["https://push.example/abc"]; ok {
		t.Fatal("expected row to be removed")
	}
}

func TestPushController_Unsubscribe_RejectsEmptyEndpoint(t *testing.T) {
	pc := NewPushController(newMockPushRepo())
	if err := pc.Unsubscribe(context.Background(), ""); !errors.Is(err, ErrPushEndpointMissing) {
		t.Fatalf("expected ErrPushEndpointMissing, got %v", err)
	}
}

func TestPushController_HasSubscription(t *testing.T) {
	repo := newMockPushRepo()
	pc := NewPushController(repo)

	// No subscriptions yet.
	has, err := pc.HasSubscription(context.Background(), "u")
	if err != nil {
		t.Fatalf("HasSubscription: %v", err)
	}
	if has {
		t.Fatal("expected has=false")
	}

	// Add one and re-check.
	repo.rows["e"] = models.PushSubscription{Endpoint: "e", UserID: "u"}
	has, err = pc.HasSubscription(context.Background(), "u")
	if err != nil {
		t.Fatalf("HasSubscription: %v", err)
	}
	if !has {
		t.Fatal("expected has=true")
	}
}

func TestAdminNotifications_BroadcastInput_Validate(t *testing.T) {
	cases := []struct {
		name string
		in   BroadcastInput
		want error
	}{
		{"ok", BroadcastInput{Title: "Hello", Body: "World"}, nil},
		{"empty title", BroadcastInput{Title: "", Body: "World"}, ErrNotificationTitleEmpty},
		{"whitespace title", BroadcastInput{Title: "   ", Body: "World"}, ErrNotificationTitleEmpty},
		{"title too long", BroadcastInput{Title: longString(MaxNotificationTitleLength + 1), Body: "World"}, ErrNotificationTitleLong},
		{"empty body", BroadcastInput{Title: "Hello", Body: ""}, ErrNotificationBodyEmpty},
		{"body too long", BroadcastInput{Title: "Hello", Body: longString(MaxNotificationBodyLength + 1)}, ErrNotificationBodyLong},
		{"url too long", BroadcastInput{Title: "Hello", Body: "World", URL: longString(MaxNotificationURLLength + 1)}, ErrNotificationURLLong},
		{"url ok", BroadcastInput{Title: "Hello", Body: "World", URL: "/foo"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.Validate()
			if !errors.Is(got, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestAdminNotifications_Broadcast_NilService(t *testing.T) {
	c := NewAdminNotificationsController(nil, nil)
	_, err := c.Broadcast(context.Background(), BroadcastInput{Title: "x", Body: "y"})
	if !errors.Is(err, ErrPushNotConfigured) {
		t.Fatalf("expected ErrPushNotConfigured, got %v", err)
	}
}

func TestAdminNotifications_SendWeightReminder_NilReminder(t *testing.T) {
	// A nil orchestrator (e.g. a server started without
	// the reminders package) must produce a clean error so
	// the route handler can render a friendly error card.
	// Returning a typed sentinel — rather than panicking
	// on a nil dereference — keeps the failure mode
	// observable in the admin UI.
	c := NewAdminNotificationsController(nil, nil)
	_, attempted, err := c.SendAllDueReminders(context.Background())
	if !errors.Is(err, ErrWeightReminderNotConfigured) {
		t.Fatalf("expected ErrWeightReminderNotConfigured, got %v", err)
	}
	if attempted {
		t.Error("attempted = true, want false when reminder is nil")
	}
}

func longString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

// keep push import alive in case the broadcast tests above are
// removed in a future refactor; the symbol is referenced indirectly
// through the push package types in the test file.
var _ = push.Subscription{}
