package push

import (
	"context"
	"sync"
)

// SubscriptionStore is the minimum data-access surface Service needs.
// Defined here (not in models/) so the push package can be tested with
// a simple in-memory fake, and so the model layer can satisfy it
// without circular imports. The interface mirrors the methods of
// models.SubscriptionRepo that Service actually calls.
type SubscriptionStore interface {
	// ListAll returns every subscription in the system, regardless of
	// which user owns it. Used by the admin broadcast.
	ListAll(ctx context.Context) ([]Subscription, error)
	// DeleteByEndpoint removes a single subscription by its push
	// service endpoint URL. Called when the push service reports the
	// device as gone (404/410). It is a no-op if the endpoint does
	// not exist.
	DeleteByEndpoint(ctx context.Context, endpoint string) error
}

// Sender is the narrow contract Service depends on for delivering
// one message to one subscription. Defining it as an interface
// (rather than depending on *Client directly) lets the service be
// unit-tested without a real VAPID keypair or a fake HTTP client —
// callers can implement Send with whatever the test needs.
type Sender interface {
	Send(ctx context.Context, sub Subscription, msg Message) SendOutcome
}

// Service fans out a Message to every subscription in the system,
// counts results, and prunes subscriptions that the push service has
// rejected as gone. It is the only place that orchestrates concurrency
// for outbound pushes.
//
// Concurrency model: a single producer goroutine dispatches one
// subscription per worker via a bounded channel. Workers are limited
// to maxWorkers (default 10) so a sudden spike of subscriptions cannot
// exhaust file descriptors or starve other handlers. A WaitGroup
// ensures the function only returns after every send has finished.
type Service struct {
	sender     Sender
	store      SubscriptionStore
	maxWorkers int
}

// ServiceConfig groups the optional knobs on Service. Zero values
// fall back to package defaults.
type ServiceConfig struct {
	// MaxWorkers bounds the number of concurrent send goroutines.
	// Default: 10.
	MaxWorkers int
}

const defaultMaxWorkers = 10

// NewService returns a Service ready to broadcast.
func NewService(sender Sender, store SubscriptionStore, cfg ServiceConfig) *Service {
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = defaultMaxWorkers
	}
	return &Service{
		sender:     sender,
		store:      store,
		maxWorkers: cfg.MaxWorkers,
	}
}

// BroadcastResult is the aggregated result of a Broadcast call. It is
// returned to the admin controller for rendering the result toast.
type BroadcastResult struct {
	// Sent is the number of subscriptions the push service accepted
	// (any 2xx response).
	Sent int
	// Deleted is the number of subscriptions pruned because the push
	// service reported them as gone (404/410). These are the only
	// "good news" failures — the admin does not need to act on them.
	Deleted int
	// Failed is the number of subscriptions where the send did not
	// succeed and the row is still in the table. Each one is paired
	// with an entry in Errors (truncated to the first few to keep
	// the toast small).
	Failed int
	// Total is the number of subscriptions we attempted to reach.
	Total int
	// Errors contains a short summary of up to maxErrors individual
	// failure messages. Useful for the admin toast; full details
	// should be in the server log.
	Errors []string
}

const maxErrors = 5

// Broadcast sends msg to every subscription in the store. It always
// returns a non-nil result; transport-level failures do not abort the
// fan-out. The returned error is non-nil only when the store itself
// could not be read (e.g. database down) — in that case nothing was
// sent and the admin should be told to retry.
//
// ctx controls only the store calls (ListAll, DeleteByEndpoint) and is
// passed through to each per-subscription send. Cancellation stops new
// sends and prunes no further subscriptions.
func (s *Service) Broadcast(ctx context.Context, msg Message) (BroadcastResult, error) {
	subs, err := s.store.ListAll(ctx)
	if err != nil {
		return BroadcastResult{}, err
	}

	result := BroadcastResult{Total: len(subs)}
	if len(subs) == 0 {
		return result, nil
	}

	workers := s.maxWorkers
	if workers > len(subs) {
		workers = len(subs)
	}

	jobs := make(chan Subscription, len(subs))
	outcomes := make(chan SendOutcome, len(subs))

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for sub := range jobs {
				if ctx.Err() != nil {
					// Caller cancelled; report the rest as
					// failures so the admin sees the count.
					outcomes <- SendOutcome{
						Subscription: sub,
						Error:        ctx.Err(),
					}
					continue
				}
				outcomes <- s.sender.Send(ctx, sub, msg)
			}
		}()
	}

	for _, sub := range subs {
		jobs <- sub
	}
	close(jobs)

	// Close outcomes only after all workers have finished writing to
	// it, otherwise the receive loop will race the producers.
	go func() {
		wg.Wait()
		close(outcomes)
	}()

	var errMu sync.Mutex
	addError := func(msg string) {
		errMu.Lock()
		defer errMu.Unlock()
		if len(result.Errors) < maxErrors {
			result.Errors = append(result.Errors, msg)
		}
	}

	for o := range outcomes {
		switch {
		case o.Error == nil:
			result.Sent++
		case o.Deleted:
			result.Deleted++
			// Best-effort prune. A failure to delete is logged
			// in the Errors summary but does not change the
			// Deleted counter — the push service still told us
			// the device is gone, and the next broadcast will
			// retry the DELETE.
			if err := s.store.DeleteByEndpoint(ctx, o.Subscription.Endpoint); err != nil {
				addError("prune " + truncate(o.Subscription.Endpoint, 40) + ": " + err.Error())
			}
		default:
			result.Failed++
			addError(truncate(o.Subscription.Endpoint, 40) + ": " + o.Error.Error())
		}
	}

	return result, nil
}

// truncate clips a string to max runes with an ellipsis suffix. Used
// to keep the admin toast short. Returns the input unchanged if it
// already fits.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
