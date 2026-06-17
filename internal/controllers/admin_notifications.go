package controllers

import (
	"context"
	"errors"
	"strings"

	"stren/internal/push"
)

// Constraints on the user-supplied notification fields. Kept in one
// place so the form validation and the templ's maxlength attribute
// cannot drift apart.
const (
	MinNotificationTitleLength = 1
	MaxNotificationTitleLength = 80
	MinNotificationBodyLength  = 1
	MaxNotificationBodyLength  = 500
	MaxNotificationURLLength   = 2000
)

// Admin notification errors. They are returned as plain sentinels so
// the route handler can switch on them and pick the right HTMX
// response (toast, error card, etc.).
var (
	ErrNotificationTitleEmpty = errors.New("title is required")
	ErrNotificationTitleLong  = errors.New("title must be at most 80 characters")
	ErrNotificationBodyEmpty  = errors.New("message is required")
	ErrNotificationBodyLong   = errors.New("message must be at most 500 characters")
	ErrNotificationURLLong    = errors.New("URL must be at most 2000 characters")
	ErrPushNotConfigured      = errors.New("push notifications are not configured on this server")
)

// AdminNotificationsController handles the admin-side broadcast form.
// It wraps the push.Service and applies simple input validation. The
// fan-out itself (concurrency, retries, dead-subscription pruning) is
// the service's job — the controller's only responsibility is parsing
// the form and translating the result into something the route can
// render.
type AdminNotificationsController struct {
	service *push.Service
}

// NewAdminNotificationsController returns a controller bound to the
// given push service. service may be nil if the server is running
// without VAPID keys (e.g. unit tests) — the controller checks for
// nil before use and returns a friendly error.
func NewAdminNotificationsController(service *push.Service) *AdminNotificationsController {
	return &AdminNotificationsController{service: service}
}

// BroadcastInput is the parsed body of the admin send form.
type BroadcastInput struct {
	Title string
	Body  string
	URL   string
}

// Validate checks the input against the length limits. Returns the
// first violation so the form can show a single, focused error.
func (in BroadcastInput) Validate() error {
	title := strings.TrimSpace(in.Title)
	body := strings.TrimSpace(in.Body)
	url := strings.TrimSpace(in.URL)

	if len(title) < MinNotificationTitleLength {
		return ErrNotificationTitleEmpty
	}
	if len(title) > MaxNotificationTitleLength {
		return ErrNotificationTitleLong
	}
	if len(body) < MinNotificationBodyLength {
		return ErrNotificationBodyEmpty
	}
	if len(body) > MaxNotificationBodyLength {
		return ErrNotificationBodyLong
	}
	if len(url) > MaxNotificationURLLength {
		return ErrNotificationURLLong
	}
	return nil
}

// Broadcast runs a fan-out to every subscription. Returns the
// aggregated result the route can render in a result card, or a
// validation error.
func (c *AdminNotificationsController) Broadcast(ctx context.Context, in BroadcastInput) (push.BroadcastResult, error) {
	if c.service == nil {
		return push.BroadcastResult{}, ErrPushNotConfigured
	}
	if err := in.Validate(); err != nil {
		return push.BroadcastResult{}, err
	}
	return c.service.Broadcast(ctx, push.Message{
		Title: strings.TrimSpace(in.Title),
		Body:  strings.TrimSpace(in.Body),
		URL:   strings.TrimSpace(in.URL),
	})
}
