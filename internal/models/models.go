package models

import (
	"fmt"
	"net/url"
	"slices"
	"time"
)

// ExerciseType represents the type of exercise.
type ExerciseType string

const (
	ExerciseTypeStrength ExerciseType = "strength"
	ExerciseTypeCardio   ExerciseType = "cardio"
	ExerciseTypeOther    ExerciseType = "other"
)

// IsValid checks if the exercise type is a valid value.
func (et ExerciseType) IsValid() bool {
	return slices.Contains([]ExerciseType{ExerciseTypeStrength, ExerciseTypeCardio, ExerciseTypeOther}, et)
}

// Exercise represents a normalized exercise name with metadata.
type Exercise struct {
	ID          string
	Name        string
	Description string
	VideoURL    string
	ImgURL      string
	Type        ExerciseType
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
	RestTime     int
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

// ValidateURL checks if a string is a valid URL.
func ValidateURL(s string) bool {
	if s == "" {
		return true
	}
	u, err := url.Parse(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}