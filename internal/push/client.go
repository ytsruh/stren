package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// Default values for the outbound push. They are package-level (not
// exposed for runtime configuration) because they are a reasonable
// default for the project's "one-shot admin message" use case. If a
// future feature needs different values per-call, add an Options field
// to Client.Send.
const (
	defaultTTL      = 24 * time.Hour
	defaultUrgency  = webpush.UrgencyNormal
	defaultTopic    = "stren-admin-message"
	defaultSubject  = "mailto:chris@stren.ytsruh.com"
	defaultHTTPTime = 30 * time.Second
)

// maxErrorBody caps the size of the push service response body that
// the wrapper will copy into SendOutcome.Error. Apple, FCM, and
// Mozilla each return a short JSON body on failure (typically
// {"reason": "..."}); a few hundred bytes is plenty. Anything larger
// is almost certainly an HTML error page from a proxy, which is not
// useful in a log line.
const maxErrorBody = 256

// Subscription is the subset of a browser PushSubscription that the
// server needs to deliver a message. It is intentionally narrower than
// the full browser object (which also has `expirationTime` and options)
// so the type is easy to construct in tests and in the database layer.
type Subscription struct {
	// Endpoint is the push service URL the browser registered for.
	Endpoint string
	// P256dh is the base64url-encoded ECDH public key the browser
	// generated. Used to encrypt the payload (RFC 8291 §3.1).
	P256dh string
	// Auth is the base64url-encoded 16-byte authentication secret.
	// Combined with the ECDH shared secret to derive the content
	// encryption key (RFC 8291 §3.1).
	Auth string
}

// Message is the payload sent to the service worker. The service
// worker (public/sw.js) parses this and calls showNotification.
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
}

// SendOutcome reports the result of a single subscription send. It is
// the smallest amount of information the service needs to drive its
// counters and to decide which subscriptions to prune.
type SendOutcome struct {
	Subscription Subscription
	Status       int
	// Deleted is true when the push service rejected the subscription
	// with a status that means the device is gone (404 or 410). The
	// service layer should drop the row from the database.
	Deleted bool
	// Error is non-nil for transport-level failures or non-2xx status
	// codes that are not 404/410. The service layer logs and counts
	// these but does not delete the row.
	Error error
}

// Client wraps webpush-go and turns its single-subscription Send call
// into a small, easy-to-fake interface. The wrapper exists for three
// reasons:
//
//  1. Hide webpush-go behind our own types so the rest of the codebase
//     has no compile-time coupling to the library (per AGENTS.md:
//     "wrap dependencies in packages so they are isolated and easily
//     edited, removed, updated without codebase wide changes").
//  2. Translate HTTP status codes from the push service into a small
//     SendOutcome so the service layer can branch on Deleted/Error
//     without re-implementing the 404/410 contract.
//  3. Centralise defaults (TTL, urgency, topic, subject) so policy
//     changes are a single-file edit.
type Client struct {
	keys         *Keys
	httpClient   webpush.HTTPClient
	subscriber   string
	topic        string
	ttl          time.Duration
	urgency      webpush.Urgency
}

// ClientConfig holds the optional knobs on Client. Zero values fall
// back to package defaults, so callers only need to set what they care
// about.
type ClientConfig struct {
	// HTTPClient is the net/http client used to talk to the push
	// service. Defaults to a 30s-timeout client.
	HTTPClient webpush.HTTPClient
	// Subscriber is the VAPID `sub` claim: a mailto: URL or
	// https:// URL identifying the operator. Required by the spec.
	//
	// The value MUST resolve to a real contact. Apple
	// (web.push.apple.com) is the strictest of the major push
	// providers and returns HTTP 403 when the `sub` is a clearly
	// placeholder address (e.g. a `.local` hostname). FCM and
	// Mozilla autopush tend to accept placeholder values, which
	// means a bad default here shows up as iOS-only 403s in
	// production while every other platform appears to work.
	Subscriber string
	// Topic groups multiple messages so the push service can collapse
	// them on the device. Default: "stren-admin-message".
	Topic string
	// TTL is how long the push service may keep the message before
	// giving up. Default: 24h.
	TTL time.Duration
	// Urgency is the delivery priority hint. Default: normal.
	Urgency webpush.Urgency
	// Logger, if non-nil, is called once per outbound push with a
	// diagnostic line containing the VAPID Authorization header
	// (the JWT webpush-go built for this request, in the form
	// "vapid t=<JWT>,k=<pubkey>"). Intended for debugging
	// non-2xx responses like Apple's BadJwtToken, where the push
	// service's body alone doesn't tell us which JWT claim is
	// wrong. nil = silent. The default production configuration
	// leaves this nil so the wrapper pays zero cost.
	Logger func(format string, args ...any)
}

// NewClient returns a Client ready to send. The keys must already be
// loaded via LoadOrGenerate; we hold a pointer to keep the keypair
// alive for the VAPID JWT signing.
func NewClient(keys *Keys, cfg ClientConfig) *Client {
	if keys == nil {
		panic("push: NewClient called with nil keys")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: defaultHTTPTime}
	}
	if cfg.Subscriber == "" {
		cfg.Subscriber = defaultSubject
	}
	if cfg.Topic == "" {
		cfg.Topic = defaultTopic
	}
	if cfg.TTL == 0 {
		cfg.TTL = defaultTTL
	}
	if cfg.Urgency == "" {
		cfg.Urgency = defaultUrgency
	}
	if cfg.Logger != nil {
		cfg.HTTPClient = &loggingHTTPClient{
			inner:  cfg.HTTPClient,
			logger: cfg.Logger,
		}
	}
	return &Client{
		keys:       keys,
		httpClient: cfg.HTTPClient,
		subscriber: cfg.Subscriber,
		topic:      cfg.Topic,
		ttl:        cfg.TTL,
		urgency:    cfg.Urgency,
	}
}

// loggingHTTPClient is a thin webpush.HTTPClient wrapper that
// records the VAPID Authorization header on every outbound push.
// It is installed by NewClient only when ClientConfig.Logger is
// non-nil, so the default production config pays no cost. The
// request body is not logged — the JWT and the endpoint are what
// we need to diagnose a BadJwtToken-style rejection.
type loggingHTTPClient struct {
	inner  webpush.HTTPClient
	logger func(format string, args ...any)
}

func (l *loggingHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if auth := req.Header.Get("Authorization"); auth != "" {
		l.logger("push: %s %s auth=%q", req.Method, req.URL.String(), auth)
	}
	return l.inner.Do(req)
}

// Send delivers msg to a single subscription. It returns a SendOutcome
// that the service layer can use for accounting.
//
// The function never returns a non-nil error from the wrapper itself;
// transport-level failures and non-2xx HTTP codes are reported through
// SendOutcome.Error so a single broadcast can collect per-subscription
// results without short-circuiting on the first failure.
func (c *Client) Send(ctx context.Context, sub Subscription, msg Message) SendOutcome {
	if sub.Endpoint == "" || sub.P256dh == "" || sub.Auth == "" {
		return SendOutcome{
			Subscription: sub,
			Error:        errors.New("push: subscription is missing required keys"),
		}
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return SendOutcome{
			Subscription: sub,
			Error:        fmt.Errorf("push: marshal message: %w", err),
		}
	}

	opts := &webpush.Options{
		HTTPClient:      c.httpClient,
		Subscriber:      c.subscriber,
		Topic:           c.topic,
		TTL:             int(c.ttl.Seconds()),
		Urgency:         c.urgency,
		VAPIDPublicKey:  c.keys.PublicKeyString(),
		VAPIDPrivateKey: c.keys.PrivateKeyString(),
	}

	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}, opts)

	if err != nil {
		// Transport-level failure (DNS, TLS, timeout, etc.) — keep
		// the subscription, surface the error.
		return SendOutcome{
			Subscription: sub,
			Error:        err,
		}
	}
	defer resp.Body.Close()

	// Drain the body on every path so the underlying HTTP connection
	// can be reused. For 2xx the body is typically empty, but reading
	// is harmless. For non-2xx the body usually contains the push
	// service's reason (Apple: {"reason":"BadJwt"}, FCM: NOT_FOUND,
	// Mozilla: a plaintext code) and is the difference between
	// knowing and guessing the next time a 403 lands in production.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody+1))
	truncated := ""
	if len(body) > maxErrorBody {
		body = body[:maxErrorBody]
		truncated = "…"
	}
	bodyStr := strings.TrimSpace(string(body))

	out := SendOutcome{
		Subscription: sub,
		Status:       resp.StatusCode,
	}

	// 2xx is success. 4xx/5xx is a per-subscription failure. 404 and
	// 410 specifically mean the subscription is gone (RFC 8030 §7.3
	// and §7.4) and should be pruned from the database.
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return out
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		out.Deleted = true
		if bodyStr != "" {
			out.Error = fmt.Errorf("push: subscription gone (status %d): %s%s", resp.StatusCode, bodyStr, truncated)
		} else {
			out.Error = fmt.Errorf("push: subscription gone (status %d)", resp.StatusCode)
		}
		return out
	default:
		if bodyStr != "" {
			out.Error = fmt.Errorf("push: unexpected status %d from push service: %s%s", resp.StatusCode, bodyStr, truncated)
		} else {
			out.Error = fmt.Errorf("push: unexpected status %d from push service", resp.StatusCode)
		}
		return out
	}
}
