package controllers

import (
	"context"
	"errors"
	"strings"

	"stren/internal/models"
)

// Push-related errors. They are returned as plain sentinels so the
// route handler can switch on them and pick the right HTMX response
// (toast, error card, etc.).
var (
	ErrPushSubscriptionMissingFields = errors.New("subscription is missing required fields")
	ErrPushEndpointMissing           = errors.New("endpoint is required")
)

// PushController handles the authenticated user's push subscription
// state. It exposes three operations:
//
//   - Subscribe:    store the browser's PushSubscription JSON
//   - Unsubscribe:  delete it
//   - HasSubscription: cheap count for the profile banner
//
// Subscribe/unsubscribe return nil on success or a validation error
// the route handler can map to a toast.
type PushController struct {
	repo models.PushSubscriptionRepo
}

// NewPushController returns a PushController bound to repo.
func NewPushController(repo models.PushSubscriptionRepo) *PushController {
	return &PushController{repo: repo}
}

// SubscribeInput is the parsed form/JSON body of a subscribe request.
// The browser sends the full PushSubscription object — we only need
// the three wire-level fields, so the route handler strips the rest
// before calling Subscribe.
type SubscribeInput struct {
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

// Subscribe stores a subscription for userID. It is idempotent:
// re-subscribing the same device overwrites the keys and bumps
// last_seen_at but does not create a duplicate row.
func (pc *PushController) Subscribe(ctx context.Context, userID string, in SubscribeInput) error {
	in.Endpoint = strings.TrimSpace(in.Endpoint)
	in.P256dh = strings.TrimSpace(in.P256dh)
	in.Auth = strings.TrimSpace(in.Auth)
	if in.Endpoint == "" || in.P256dh == "" || in.Auth == "" {
		return ErrPushSubscriptionMissingFields
	}
	_, err := pc.repo.UpsertForUser(ctx, userID, models.PushSubscription{
		Endpoint: in.Endpoint,
		P256dh:   in.P256dh,
		Auth:     in.Auth,
	})
	return err
}

// Unsubscribe removes the subscription for endpoint. Idempotent:
// removing a non-existent row is not an error.
func (pc *PushController) Unsubscribe(ctx context.Context, endpoint string) error {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ErrPushEndpointMissing
	}
	// The push protocol guarantees endpoint uniqueness across the
	// table, so the global delete is equivalent to a user-scoped
	// delete in practice.
	return pc.repo.DeleteByEndpoint(ctx, endpoint)
}

// HasSubscription reports whether the user has at least one
// subscription row. Used to drive the initial banner state.
func (pc *PushController) HasSubscription(ctx context.Context, userID string) (bool, error) {
	n, err := pc.repo.CountForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
