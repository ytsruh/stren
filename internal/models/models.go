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

// HistoryStats summarises an exercise's training history for the header stat cards.
// LastSet is a zero-value ExerciseEntry when the user has no entries for the exercise.
type HistoryStats struct {
	MaxWeight float64
	LastSet   ExerciseEntry
}

// ExerciseHistoryPage bundles a single page of history entries with the stats
// needed to render the page header and the pagination state.
type ExerciseHistoryPage struct {
	Entries []ExerciseEntry
	Stats   HistoryStats
	Page    int
	HasPrev bool
	HasNext bool
}

// FormattedWeight returns the weight labelled with the given unit.
// No conversion happens — the value is rendered as "%.1f <unit>" so
// the number is displayed using whatever unit the user prefers.
func (e *ExerciseEntry) FormattedWeight(unit string) string {
	return FormatWeight(e.Weight, unit)
}

// FormatWeight returns a human-readable weight string using the
// given unit. No conversion happens — the value is labelled with
// whatever unit the caller passes. Use everywhere a weight is
// shown to a user (display, chart labels, form labels) or written
// to a CSV.
func FormatWeight(value float64, unit string) string {
	return fmt.Sprintf("%.1f %s", value, unit)
}

// NormalizeWeightUnit returns a clean "kg" or "lbs", or "kg" if
// the input is empty or unrecognised. Use at trust boundaries
// (e.g. reading a value from a form or the DB) so downstream code
// can rely on a normalised value.
func NormalizeWeightUnit(unit string) string {
	switch unit {
	case "kg", "lbs":
		return unit
	default:
		return "kg"
	}
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