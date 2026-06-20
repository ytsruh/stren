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

// ValidWeightUnits enumerates the allowed values for User.WeightUnit.
var ValidWeightUnits = []string{"kg", "lbs"}
