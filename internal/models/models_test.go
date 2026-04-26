package models

import (
	"testing"
	"time"
)

func TestExerciseEntry_FormattedWeight(t *testing.T) {
	tests := []struct {
		name     string
		weight   float64
		expected string
	}{
		{
			name:     "whole number",
			weight:   100,
			expected: "100.0 kg",
		},
		{
			name:     "decimal",
			weight:   67.5,
			expected: "67.5 kg",
		},
		{
			name:     "zero",
			weight:   0,
			expected: "0.0 kg",
		},
		{
			name:     "large number",
			weight:   999.9,
			expected: "999.9 kg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &ExerciseEntry{Weight: tt.weight}
			got := entry.FormattedWeight()
			if got != tt.expected {
				t.Errorf("FormattedWeight() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExerciseEntry_FormattedDate(t *testing.T) {
	loc := time.UTC
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "january date",
			input:    time.Date(2024, 1, 15, 0, 0, 0, 0, loc),
			expected: "Jan 15, 2024",
		},
		{
			name:     "december date",
			input:    time.Date(2023, 12, 31, 23, 59, 59, 0, loc),
			expected: "Dec 31, 2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &ExerciseEntry{CreatedAt: tt.input}
			got := entry.FormattedDate()
			if got != tt.expected {
				t.Errorf("FormattedDate() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExerciseEntry_FormattedTime(t *testing.T) {
	loc := time.UTC
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "morning time",
			input:    time.Date(2024, 1, 1, 9, 30, 0, 0, loc),
			expected: "9:30 AM",
		},
		{
			name:     "afternoon time",
			input:    time.Date(2024, 1, 1, 14, 45, 0, 0, loc),
			expected: "2:45 PM",
		},
		{
			name:     "midnight",
			input:    time.Date(2024, 1, 1, 0, 0, 0, 0, loc),
			expected: "12:00 AM",
		},
		{
			name:     "noon",
			input:    time.Date(2024, 1, 1, 12, 0, 0, 0, loc),
			expected: "12:00 PM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &ExerciseEntry{CreatedAt: tt.input}
			got := entry.FormattedTime()
			if got != tt.expected {
				t.Errorf("FormattedTime() = %q, want %q", got, tt.expected)
			}
		})
	}
}
