package models

import "time"

// User represents an authenticated user of the strength tracker.
type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	IsAdmin      bool
	// TargetWeight is the user's body-weight goal. nil means the user has not
	// set a goal; the weight page should hide the progress widget in that case.
	TargetWeight *float64
	// WeightUnit is the user's preferred body-weight unit ("kg" or "lbs").
	// Persisted now so a future per-user unit display can read it; the rest of
	// the app still renders all weights as "kg" until that wiring is in place.
	WeightUnit string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// HasWeightGoal reports whether the user has set a target weight.
func (u *User) HasWeightGoal() bool {
	return u.TargetWeight != nil
}

// WeightUnitDisplay returns the user's preferred weight unit,
// normalised to "kg" or "lbs". Falls back to "kg" when the user
// is nil or the stored unit is empty or unrecognised. Use this
// everywhere a weight is shown to the user (display, chart
// labels, form labels, CSV export) so the normalisation happens
// once, at the boundary, rather than at every call site.
func (u *User) WeightUnitDisplay() string {
	if u == nil {
		return "kg"
	}
	return NormalizeWeightUnit(u.WeightUnit)
}

// ValidWeightUnits enumerates the allowed values for User.WeightUnit.
var ValidWeightUnits = []string{"kg", "lbs"}
