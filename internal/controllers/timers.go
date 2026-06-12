// Package controllers provides business logic for the strength tracker application.
package controllers

import (
	"github.com/labstack/echo/v4"

	"stren/internal/views"
	"stren/internal/views/timers"
)

// TimersController handles timer and EMOM business logic.
type TimersController struct{}

// NewTimersController creates a new TimersController instance.
func NewTimersController() *TimersController {
	return &TimersController{}
}

// TimerPage renders the timer tab.
func (tc *TimersController) TimerPage(c echo.Context, isAdmin bool) error {
	data := views.PageData{
		Title:           "Timer",
		IsAuthenticated: true,
		IsAdmin:         isAdmin,
		CurrentPath:     "/timer",
	}
	return timers.TimerPage(data).Render(c.Request().Context(), c.Response())
}

// EMOMPage renders the EMOM timer tab.
func (tc *TimersController) EMOMPage(c echo.Context, isAdmin bool) error {
	data := views.PageData{
		Title:           "EMOM Timer",
		IsAuthenticated: true,
		IsAdmin:         isAdmin,
		CurrentPath:     "/timer/emom",
	}
	return timers.EMOMPage(data).Render(c.Request().Context(), c.Response())
}
