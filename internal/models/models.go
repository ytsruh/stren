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

// FormattedDate returns a human-readable date
func (e *ExerciseEntry) FormattedDate() string {
	return e.CreatedAt.Format("Jan 02, 2006")
}

// FormattedTime returns a human-readable time
func (e *ExerciseEntry) FormattedTime() string {
	return e.CreatedAt.Format("3:04 PM")
}