package models

import (
	"fmt"
	"time"
)

// ExerciseType represents a normalized exercise name
type ExerciseType struct {
	ID   int64
	Name string
}

// ExerciseEntry represents a single set of an exercise
type ExerciseEntry struct {
	ID             int64
	ExerciseTypeID int64
	UserID         int64
	ExerciseName   string
	Reps           int
	Weight         float64
	Notes          string
	CreatedAt      time.Time
}

// FormattedWeight returns the weight with unit
func (e *ExerciseEntry) FormattedWeight() string {
	return fmt.Sprintf("%.1f kg", e.Weight)
}

// FormattedDate returns a human-readable date in UK short format
func (e *ExerciseEntry) FormattedDate() string {
	return e.CreatedAt.Format("02/01/06")
}