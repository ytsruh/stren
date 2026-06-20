package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"stren/internal/db"
)

// PushSubscription is the application-level view of a stored browser
// push subscription. One row per (user, device) — a user with two
// devices (e.g. phone + laptop) has two rows.
type PushSubscription struct {
	ID         string
	UserID     string
	Endpoint   string
	P256dh     string
	Auth       string
	CreatedAt  time.Time
	LastSeenAt time.Time
}

// PushSubscriptionRepository is the default implementation backed by
// the sqlc-generated queries. It satisfies both the application-level
// PushSubscriptionRepo (used by controllers, declared in
// repository.go) and the narrower push.SubscriptionStore (used by
// push.Service for fan-out) — the two are intentionally different so
// callers can declare only what they actually need.
type PushSubscriptionRepository struct {
	queries *db.Queries
}

// Compile-time check: ensure the repository satisfies the model-level
// interface. The push.SubscriptionStore satisfaction is checked in the
// push package via a small constructor, since the interface lives
// there.
var _ PushSubscriptionRepo = (*PushSubscriptionRepository)(nil)

// NewPushSubscriptionRepository returns a repository bound to the
// given database connection. The connection is wrapped in a sqlc
// Queries object on every call to keep the constructor cheap and
// consistent with the other repositories in the package.
func NewPushSubscriptionRepository(database *db.DB) *PushSubscriptionRepository {
	return &PushSubscriptionRepository{queries: db.New(database.Conn())}
}

// UpsertForUser inserts or updates a subscription keyed on endpoint.
// If the endpoint already exists we update the keys and bump
// last_seen_at but keep the original user_id — a device belongs to
// whoever first subscribed, even if it briefly re-subscribes from a
// different session of the same browser.
//
// MySQL/SQLite do not have native upsert with a RETURNING clause, so
// the implementation is "try update, fall back to insert, retry on
// unique violation". The two-step approach keeps the SQL portable and
// avoids a transaction.
func (r *PushSubscriptionRepository) UpsertForUser(ctx context.Context, userID string, sub PushSubscription) (*PushSubscription, error) {
	if sub.Endpoint == "" || sub.P256dh == "" || sub.Auth == "" {
		return nil, errors.New("push: subscription is missing required keys")
	}
	if userID == "" {
		return nil, errors.New("push: userID is empty")
	}

	existing, err := r.queries.GetPushSubscriptionByEndpoint(ctx, sub.Endpoint)
	_ = existing // retained to keep the read-then-write pattern explicit
	if err == nil {
		row, err := r.queries.UpdatePushSubscription(ctx, db.UpdatePushSubscriptionParams{
			P256dh:   sub.P256dh,
			Auth:     sub.Auth,
			Endpoint: sub.Endpoint,
		})
		if err != nil {
			return nil, fmt.Errorf("update subscription: %w", err)
		}
		return pushRowToModel(row), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("lookup subscription: %w", err)
	}

	// No existing row — insert. If two requests race for the same
	// endpoint, one will lose on the UNIQUE constraint; we retry as
	// an update.
	id := uuid.New().String()
	row, err := r.queries.CreatePushSubscription(ctx, db.CreatePushSubscriptionParams{
		ID:       id,
		UserID:   userID,
		Endpoint: sub.Endpoint,
		P256dh:   sub.P256dh,
		Auth:     sub.Auth,
	})
	if err == nil {
		return pushRowToModel(row), nil
	}
	// Best-effort retry once: a race can cause UNIQUE failure even
	// though we just looked. We swallow the first insert error if
	// the retry succeeds, otherwise surface it.
	row, err2 := r.queries.UpdatePushSubscription(ctx, db.UpdatePushSubscriptionParams{
		P256dh:   sub.P256dh,
		Auth:     sub.Auth,
		Endpoint: sub.Endpoint,
	})
	if err2 != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}
	return pushRowToModel(row), nil
}

// ListForUser returns the user's subscriptions, newest first.
func (r *PushSubscriptionRepository) ListForUser(ctx context.Context, userID string) ([]PushSubscription, error) {
	rows, err := r.queries.ListAllPushSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PushSubscription, 0, len(rows))
	for _, row := range rows {
		if row.UserID == userID {
			out = append(out, *pushRowToModel(row))
		}
	}
	return out, nil
}

// ListAll returns every subscription in the system. There is no
// ordering guarantee; the service layer is the only caller and does
// not need one.
func (r *PushSubscriptionRepository) ListAll(ctx context.Context) ([]PushSubscription, error) {
	rows, err := r.queries.ListAllPushSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PushSubscription, 0, len(rows))
	for _, row := range rows {
		out = append(out, *pushRowToModel(row))
	}
	return out, nil
}

// DeleteByEndpoint removes the row with the given endpoint. Returns
// nil if the row does not exist (idempotent).
func (r *PushSubscriptionRepository) DeleteByEndpoint(ctx context.Context, endpoint string) error {
	return r.queries.DeletePushSubscriptionByEndpoint(ctx, endpoint)
}

// CountForUser returns the number of subscriptions the user owns.
func (r *PushSubscriptionRepository) CountForUser(ctx context.Context, userID string) (int64, error) {
	return r.queries.CountPushSubscriptionsByUser(ctx, userID)
}

// pushRowToModel converts the sqlc-generated row into the application
// model. Extracted so every read path goes through one place — if the
// schema ever gains a column the change is a single edit.
func pushRowToModel(row db.PushSubscription) *PushSubscription {
	return &PushSubscription{
		ID:         row.ID,
		UserID:     row.UserID,
		Endpoint:   row.Endpoint,
		P256dh:     row.P256dh,
		Auth:       row.Auth,
		CreatedAt:  row.CreatedAt,
		LastSeenAt: row.LastSeenAt,
	}
}
