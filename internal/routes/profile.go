// Package routes provides HTTP route handlers for the strength tracker application.
package routes

import (
	"github.com/labstack/echo/v4"

	"stren/internal/views"
)

// Profile renders the authenticated user's profile page.
func (h *Handler) Profile(c echo.Context) error {
	claims := GetClaims(c)
	return render(c, views.ProfilePage(claims.Name, claims.Email, claims.IsAdmin))
}
