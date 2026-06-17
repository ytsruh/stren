// Package routes: admin_notifications.go wires the admin push
// broadcast form. Both routes are gated by AdminMiddleware() in
// routes.go's /admin group.
package routes

import (
	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/views"
)

// AdminNotificationsForm renders the admin "Notifications" page.
func (h *Handler) AdminNotificationsForm(c echo.Context) error {
	claims := GetClaims(c)
	return render(c, views.AdminNotificationsPage(claims.Name, claims.IsAdmin))
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
		return render(c, views.AdminNotificationError(err.Error()))
	}

	c.Response().Header().Set("HX-Trigger", adminNotificationSentTrigger)
	if result.Total == 0 {
		return render(c, views.AdminNotificationNoSubscribers())
	}
	return render(c, views.AdminNotificationResult(result))
}
