// Package controllers provides business logic for the strength tracker application.
package controllers

import (
	"github.com/labstack/echo/v4"

	"stren/internal/views"
)

// TimerController handles timer-related business logic.
type TimerController struct{}

// NewTimerController creates a new TimerController instance.
func NewTimerController() *TimerController {
	return &TimerController{}
}

// TimerPage renders the timer page.
func (tc *TimerController) TimerPage(c echo.Context, isAdmin bool) error {
	data := views.PageData{
		Title:           "Timer",
		IsAuthenticated: true,
		IsAdmin:         isAdmin,
	}
	return views.TimerPage(data).Render(c.Request().Context(), c.Response())
}