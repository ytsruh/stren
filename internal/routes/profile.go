// Package routes provides HTTP route handlers for the strength tracker application.
package routes

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"stren/internal/models"
	"stren/internal/views/profile"
)

// profileInput represents the parsed and validated form data for profile updates.
// TargetWeight is a pointer so an empty form value can be distinguished from a
// real "0" entry and stored as SQL NULL (i.e. "no goal").
type profileInput struct {
	Name         string   `validate:"required,min=2,max=100"`
	TargetWeight *float64 `validate:"omitempty,gte=0,lte=1000"`
	WeightUnit   string   `validate:"omitempty,oneof=kg lbs"`
}

// defaultWeightUnit is used when the form omits the weight_unit field entirely
// (e.g. on older forms). Matches the SQL DEFAULT in migration 00005.
const defaultWeightUnit = "kg"

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

	// GetUser caches the user on the request context, so this is a
	// single DB read. A missing user (shouldn't happen — claims came
	// from a verified JWT) is treated as a user with no goal and the
	// default unit.
	user := h.GetUser(c)
	var target *float64
	unit := defaultWeightUnit
	if user != nil {
		target = user.TargetWeight
		unit = user.WeightUnitDisplay()
	}

	return render(c, profile.ProfilePage(
		claims.Name,
		claims.Email,
		claims.IsAdmin,
		target,
		unit,
		h.pushConfigured,
		hasSubscription,
		h.vapidPublicKey,
	))
}

// UpdateProfile handles profile update requests.
// It validates the input, updates the user's name, target weight, and weight
// unit, regenerates the JWT so the new name is reflected in the cookie, and
// re-issues the auth cookie.
func (h *Handler) UpdateProfile(c echo.Context) error {
	claims := GetClaims(c)

	input := profileInput{
		Name:       c.FormValue("name"),
		WeightUnit: c.FormValue("weight_unit"),
	}
	if input.WeightUnit == "" {
		input.WeightUnit = defaultWeightUnit
	}

	// target_weight is optional. An empty form value clears the goal.
	if raw := c.FormValue("target_weight"); raw != "" {
		target, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return render(c, profile.ProfileUpdateError("Target weight must be a valid positive number"))
		}
		input.TargetWeight = &target
	}

	if err := h.validator.ValidateStruct(&input); err != nil {
		return render(c, profile.ProfileUpdateError(friendlyValidationError(err)))
	}

	user := &models.User{
		ID:           claims.UserID,
		Name:         input.Name,
		Email:        claims.Email,
		IsAdmin:      claims.IsAdmin,
		TargetWeight: input.TargetWeight,
		WeightUnit:   input.WeightUnit,
	}

	if err := h.userRepo.UpdateUser(user); err != nil {
		return render(c, profile.ProfileUpdateError("Failed to update profile"))
	}

	token, err := h.jwtService.GenerateToken(user.ID, user.Email, user.Name, user.IsAdmin)
	if err != nil {
		return render(c, profile.ProfileUpdateError("Failed to generate token"))
	}
	setAuthCookie(c, token)

	return render(c, profile.ProfileUpdateSuccess())
}
