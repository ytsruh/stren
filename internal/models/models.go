package models

import (
	"fmt"
	"time"
)

// Exercise represents a normalized exercise name
type Exercise struct {
	ID   string
	Name string
}

// ExerciseEntry represents a single set of an exercise
type ExerciseEntry struct {
	ID           string
	ExerciseID   string
	UserID       string
	ExerciseName string
	Reps         int
	Weight       float64
	Notes        string
	CreatedAt    time.Time
}

// FormattedWeight returns the weight with unit
func (e *ExerciseEntry) FormattedWeight() string {
	return fmt.Sprintf("%.1f kg", e.Weight)
}

// FormattedDate returns a human-readable date in UK short format
func (e *ExerciseEntry) FormattedDate() string {
	return e.CreatedAt.Format("02/01/06")
}