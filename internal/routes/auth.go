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

// loginInput represents the validated login form data.
type loginInput struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required"`
}

// Login handles login form submission.
func (h *Handler) Login(c echo.Context) error {
	input := loginInput{
		Email:    c.FormValue("email"),
		Password: c.FormValue("password"),
	}

	if err := h.validator.ValidateStruct(&input); err != nil {
		return render(c, views.LoginPage(err.Error()))
	}

	_, token, err := h.authCtrl.Login(input.Email, input.Password)
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

// registerInput represents the validated registration form data.
type registerInput struct {
	Name     string `validate:"required,min=1,max=100"`
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=6"`
}

// Register handles registration form submission.
func (h *Handler) Register(c echo.Context) error {
	input := registerInput{
		Name:     c.FormValue("name"),
		Email:    c.FormValue("email"),
		Password: c.FormValue("password"),
	}

	if err := h.validator.ValidateStruct(&input); err != nil {
		return render(c, views.RegisterPage(err.Error()))
	}

	_, token, err := h.authCtrl.Register(input.Name, input.Email, input.Password)
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
