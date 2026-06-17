// Package routes: push.go wires the user-facing push subscription
// endpoints. They are HTMX-friendly: the form on the profile page
// POSTs the browser's PushSubscription JSON, the server validates
// and stores it, and the response is a small toast (rendered via
// the views.PushSubscribeSuccess/Error templates) that the existing
// #toaster picks up.
package routes

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/views"
)

// Subscribe handles POST /api/push/subscribe. Body is a JSON object
// with endpoint, p256dh, and auth — the only fields the browser's
// PushSubscription serialises that we care about. Authenticated
// users only.
func (h *Handler) PushSubscribe(c echo.Context) error {
	claims := GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
	}

	var in controllers.SubscribeInput
	if err := json.NewDecoder(c.Request().Body).Decode(&in); err != nil {
		return render(c, views.PushSubscribeError("Invalid subscription payload"))
	}

	if err := h.pushCtrl.Subscribe(c.Request().Context(), claims.UserID, in); err != nil {
		return render(c, views.PushSubscribeError(err.Error()))
	}
	return render(c, views.PushSubscribeSuccess())
}

// Unsubscribe handles DELETE /api/push/unsubscribe. Body is a JSON
// object with a single `endpoint` field. Authenticated users only.
func (h *Handler) PushUnsubscribe(c echo.Context) error {
	claims := GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
	}

	var in struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&in); err != nil {
		return render(c, views.PushUnsubscribeSuccess())
	}

	if err := h.pushCtrl.Unsubscribe(c.Request().Context(), in.Endpoint); err != nil {
		return render(c, views.PushSubscribeError(err.Error()))
	}
	return render(c, views.PushUnsubscribeSuccess())
}
