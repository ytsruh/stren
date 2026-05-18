// Package routes provides HTTP route handlers for the strength tracker application.
package routes

import (
	"github.com/labstack/echo/v4"

	"stren/internal/views"
)

// TimerPage renders the timer page.
func (h *Handler) TimerPage(c echo.Context) error {
	return h.timerCtrl.TimerPage(c)
}

// TimerValidationError returns a toast for invalid timer input.
func (h *Handler) TimerValidationError(c echo.Context) error {
	return render(c, views.Toast("error", "Invalid Duration", "Timer must be between 1 and 300 seconds"))
}