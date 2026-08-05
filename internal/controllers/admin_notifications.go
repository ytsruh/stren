package controllers

import (
	"context"
	"errors"
	"strings"

	"stren/internal/push"
	"stren/internal/reminders"
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
	ErrNotificationTitleEmpty    = errors.New("title is required")
	ErrNotificationTitleLong     = errors.New("title must be at most 80 characters")
	ErrNotificationBodyEmpty     = errors.New("message is required")
	ErrNotificationBodyLong      = errors.New("message must be at most 500 characters")
	ErrNotificationURLLong       = errors.New("URL must be at most 2000 characters")
	ErrPushNotConfigured         = errors.New("push notifications are not configured on this server")
	ErrWeightReminderNotConfigured = errors.New("weight reminder is not configured on this server")
)

// AdminNotificationsController handles the admin-side broadcast form
// and the admin "send all due reminders now" button. Both
// orchestrate through their respective services; the controller's
// only responsibility is parsing the form (for the broadcast) and
// translating the orchestrator's result into something the route
// can render.
type AdminNotificationsController struct {
	service        *push.Service
	weightReminder *reminders.UserReminder
}

// NewAdminNotificationsController returns a controller bound to the
// given push service and weight reminder orchestrator. service may
// be nil if the server is running without VAPID keys (e.g. unit
// tests) — the controller checks for nil before use and returns a
// friendly error. weightReminder may also be nil; the
// SendAllDueReminders method returns ErrWeightReminderNotConfigured
// in that case.
func NewAdminNotificationsController(service *push.Service, weightReminder *reminders.UserReminder) *AdminNotificationsController {
	return &AdminNotificationsController{service: service, weightReminder: weightReminder}
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

// SendAllDueReminders fires the same orchestrator the hourly tick
// uses, on demand. The admin can use this to rehearsal-test the
// full per-user pipeline end-to-end without waiting for the next
// tick. The orchestrator finds every user whose next_fire_at is
// at or before now and fires each user's chosen channels.
//
// Returns the orchestrator's TickResult so the route can render
// a result card showing per-user outcomes (email/push sent,
// skipped, or failed) plus any aggregate errors. The "attempted"
// return mirrors the orchestrator's contract: false when the
// user list was unreadable (ListError set on the result).
//
// Returns ErrWeightReminderNotConfigured when the orchestrator
// is nil (e.g. the reminders package failed to initialise at
// startup — a separate path from the orchestrator's own
// error returns).
func (c *AdminNotificationsController) SendAllDueReminders(ctx context.Context) (reminders.TickResult, bool, error) {
	if c.weightReminder == nil {
		return reminders.TickResult{}, false, ErrWeightReminderNotConfigured
	}
	result, attempted := c.weightReminder.Run(ctx)
	return result, attempted, nil
}
