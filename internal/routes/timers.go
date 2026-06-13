// Package routes provides HTTP route handlers for the strength tracker application.
package routes

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"stren/internal/views"
)

// TimerPage renders the timer tab.
func (h *Handler) TimerPage(c echo.Context) error {
	claims := GetClaims(c)
	isAdmin := claims != nil && claims.IsAdmin
	return h.timersCtrl.TimerPage(c, isAdmin)
}

// TimerValidationError returns a toast for invalid timer input.
func (h *Handler) TimerValidationError(c echo.Context) error {
	return render(c, views.Toast("error", "Invalid Duration", "Timer must be between 1 and 300 seconds"))
}

// EMOMPage renders the EMOM timer tab.
func (h *Handler) EMOMPage(c echo.Context) error {
	claims := GetClaims(c)
	isAdmin := claims != nil && claims.IsAdmin
	return h.timersCtrl.EMOMPage(c, isAdmin)
}

// EMOMValidationError returns a toast for invalid EMOM rounds input.
func (h *Handler) EMOMValidationError(c echo.Context) error {
	return render(c, views.Toast("error", "Invalid Rounds", "Rounds must be between 1 and 15"))
}

// EMOMRoundToast returns a toast notifying the user that a round is complete.
func (h *Handler) EMOMRoundToast(c echo.Context) error {
	round := c.FormValue("round")
	r, err := strconv.Atoi(round)
	if err != nil || r < 1 {
		r = 1
	}
	return render(c, views.Toast("info", "Round "+round+" Complete", ""))
}
