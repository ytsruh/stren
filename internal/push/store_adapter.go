// StoreAdapter is a thin converter from models.PushSubscription to
// push.Subscription, allowing the model layer to satisfy the narrow
// SubscriptionStore interface the push.Service depends on.
//
// Kept in its own file (rather than as a method on the repository) so
// the push package does not need to leak domain types into the model
// layer and vice versa. The conversion is trivial: only the
// wire-level fields are exposed to the push protocol.
package push

import (
	"context"

	"stren/internal/models"
)

// StoreAdapter wraps any type that can list and delete model-level
// subscriptions and exposes them to push.Service in the wire-level
// form. It is the adapter the wiring code in cmd/main.go uses.
type StoreAdapter struct {
	Repo interface {
		ListAll(ctx context.Context) ([]models.PushSubscription, error)
		DeleteByEndpoint(ctx context.Context, endpoint string) error
	}
}

// NewStoreAdapter wraps a repository so it can be passed to
// push.NewService.
func NewStoreAdapter(repo interface {
	ListAll(ctx context.Context) ([]models.PushSubscription, error)
	DeleteByEndpoint(ctx context.Context, endpoint string) error
}) *StoreAdapter {
	return &StoreAdapter{Repo: repo}
}

// ListAll converts the model's full row representation to the
// push.Subscription wire form.
func (a *StoreAdapter) ListAll(ctx context.Context) ([]Subscription, error) {
	rows, err := a.Repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Subscription, len(rows))
	for i, r := range rows {
		out[i] = Subscription{
			Endpoint: r.Endpoint,
			P256dh:   r.P256dh,
			Auth:     r.Auth,
		}
	}
	return out, nil
}

// DeleteByEndpoint delegates to the underlying repository. It is
// idempotent: a missing endpoint is not an error.
func (a *StoreAdapter) DeleteByEndpoint(ctx context.Context, endpoint string) error {
	return a.Repo.DeleteByEndpoint(ctx, endpoint)
}

// Compile-time check: ensure the adapter itself satisfies the
// interface the service depends on.
var _ SubscriptionStore = (*StoreAdapter)(nil)
