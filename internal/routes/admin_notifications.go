// Package routes: admin_notifications.go wires the admin push
// broadcast form. Both routes are gated by AdminMiddleware() in
// routes.go's /admin group.
package routes

import (
	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/views/admin"
)

// AdminNotificationsForm renders the admin "Notifications" page.
func (h *Handler) AdminNotificationsForm(c echo.Context) error {
	claims := GetClaims(c)
	return render(c, admin.AdminNotificationsPage(claims.Name, claims.IsAdmin))
}

// adminNotificationSentTrigger is the HTMX trigger name fired via the
// HX-Trigger response header when a broadcast (or a no-subscribers
// short-circuit) completes. The admin notifications form listens for
// it and clears its fields.
const adminNotificationSentTrigger = "admin-notification-sent"

// AdminNotificationsSend handles POST /admin/notifications/send. It
// validates the form, runs the fan-out, and returns an HTMX-swapped
// result card summarising the broadcast. Empty subscriber lists
// short-circuit to a friendly "no subscribers yet" card so the admin
// gets immediate feedback. On any successful run the HX-Trigger
// response header asks the form to reset itself; validation errors
// deliberately omit it so the admin can correct and retry without
// losing their draft.
func (h *Handler) AdminNotificationsSend(c echo.Context) error {
	in := controllers.BroadcastInput{
		Title: c.FormValue("title"),
		Body:  c.FormValue("body"),
		URL:   c.FormValue("url"),
	}

	result, err := h.adminNotificationsCtrl.Broadcast(c.Request().Context(), in)
	if err != nil {
		// Validation and configuration errors return an error
		// card that swaps into the same #send-result target. The
		// admin sees the message inline rather than as a
		// dismissable toast — a more visible cue for an action
		// that didn't go through.
		return render(c, admin.AdminNotificationError(err.Error()))
	}

	c.Response().Header().Set("HX-Trigger", adminNotificationSentTrigger)
	if result.Total == 0 {
		return render(c, admin.AdminNotificationNoSubscribers())
	}
	return render(c, admin.AdminNotificationResult(result))
}

// AdminNotificationsSendWeightReminder handles POST
// /admin/notifications/send-weight-reminder. It calls the same
// reminders.UserReminder the hourly tick uses — the route exists
// so the admin can rehearsal-test the full per-user pipeline
// end-to-end without waiting for the next tick.
//
// The orchestrator's TickResult is rendered into a result card
// that swaps into the same #send-result target as the broadcast
// form. A user-list error (e.g. database down) renders the
// shared AdminNotificationError card so the admin sees the
// failure inline rather than as a dismissable toast.
func (h *Handler) AdminNotificationsSendWeightReminder(c echo.Context) error {
	result, attempted, err := h.adminNotificationsCtrl.SendAllDueReminders(c.Request().Context())
	if err != nil {
		return render(c, admin.AdminNotificationError(err.Error()))
	}
	if !attempted {
		// The orchestrator did not even read the user list;
		// the result's ListError carries the underlying
		// cause. Render that into the same error card the
		// broadcast form uses.
		return render(c, admin.AdminNotificationError("Could not load user list: "+result.ListError))
	}
	return render(c, admin.AdminWeightReminderResult(result))
}
