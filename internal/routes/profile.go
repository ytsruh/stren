// Package routes provides HTTP route handlers for the strength tracker application.
package routes

import (
	"github.com/labstack/echo/v4"

	"stren/internal/models"
	"stren/internal/views"
)

// profileInput represents the parsed and validated form data for profile updates.
type profileInput struct {
	Name string `validate:"required,min=2,max=100"`
}

// Profile renders the authenticated user's profile page.
func (h *Handler) Profile(c echo.Context) error {
	claims := GetClaims(c)

	// The push banner needs two pieces of information:
	//   1. Has the user already subscribed? Used for first-paint
	//      state; the JS will reconcile with the browser on load.
	//   2. The VAPID public key. Server-rendered into a data
	//      attribute so the JS can call pushManager.subscribe without
	//      a separate round trip.
	hasSubscription, err := h.pushCtrl.HasSubscription(c.Request().Context(), claims.UserID)
	if err != nil {
		// A failed DB read shouldn't take the profile page down.
		// We log via the Echo logger and fall through with the
		// "disabled" first-paint state — the JS will correct it
		// on load if the user is actually subscribed.
		c.Logger().Errorf("push subscription count failed: %v", err)
		hasSubscription = false
	}

	return render(c, views.ProfilePage(
		claims.Name,
		claims.Email,
		claims.IsAdmin,
		h.pushConfigured,
		hasSubscription,
		h.vapidPublicKey,
	))
}

// UpdateProfile handles profile update requests.
// It validates the input, updates the user's name, regenerates the JWT, and sets a new auth cookie.
func (h *Handler) UpdateProfile(c echo.Context) error {
	claims := GetClaims(c)

	input := profileInput{
		Name: c.FormValue("name"),
	}

	if err := h.validator.ValidateStruct(&input); err != nil {
		return render(c, views.ProfileUpdateError(friendlyValidationError(err)))
	}

	user := &models.User{
		ID:       claims.UserID,
		Name:     input.Name,
		Email:    claims.Email,
		IsAdmin:  claims.IsAdmin,
	}

	if err := h.userRepo.UpdateUser(user); err != nil {
		return render(c, views.ProfileUpdateError("Failed to update profile"))
	}

	token, err := h.jwtService.GenerateToken(user.ID, user.Email, user.Name, user.IsAdmin)
	if err != nil {
		return render(c, views.ProfileUpdateError("Failed to generate token"))
	}
	setAuthCookie(c, token)

	return render(c, views.ProfileUpdateSuccess())
}
