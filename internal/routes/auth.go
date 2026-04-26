// Package routes provides HTTP route handlers for the strength tracker application.
package routes

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/views"
)

// --- Auth Handlers ---

// LoginForm renders the login page.
func (h *Handler) LoginForm(c echo.Context) error {
	return render(c, views.LoginPage(""))
}

// Login handles login form submission.
func (h *Handler) Login(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	_, token, err := h.authCtrl.Login(email, password)
	if err != nil {
		if errors.Is(err, controllers.ErrInvalidCredentials) {
			return render(c, views.LoginPage("Invalid email or password"))
		}
		return render(c, views.LoginPage("Failed to create session"))
	}

	setAuthCookie(c, token)
	return c.Redirect(http.StatusSeeOther, "/")
}

// RegisterForm renders the registration page.
func (h *Handler) RegisterForm(c echo.Context) error {
	return render(c, views.RegisterPage(""))
}

// Register handles registration form submission.
func (h *Handler) Register(c echo.Context) error {
	name := c.FormValue("name")
	email := c.FormValue("email")
	password := c.FormValue("password")

	_, token, err := h.authCtrl.Register(name, email, password)
	if err != nil {
		return render(c, views.RegisterPage(err.Error()))
	}

	setAuthCookie(c, token)
	return c.Redirect(http.StatusSeeOther, "/")
}

// Logout clears the auth cookie and redirects to login.
func (h *Handler) Logout(c echo.Context) error {
	clearAuthCookie(c)
	return c.Redirect(http.StatusSeeOther, "/login")
}
