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
			expected: "15/01/24",
		},
		{
			name:     "december date",
			input:    time.Date(2023, 12, 31, 23, 59, 59, 0, loc),
			expected: "31/12/23",
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

func TestExerciseType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		input    ExerciseType
		expected bool
	}{
		{name: "strength is valid", input: ExerciseTypeStrength, expected: true},
		{name: "cardio is valid", input: ExerciseTypeCardio, expected: true},
		{name: "other is valid", input: ExerciseTypeOther, expected: true},
		{name: "empty is invalid", input: "", expected: false},
		{name: "unknown is invalid", input: "hybrid", expected: false},
		{name: "uppercase is invalid", input: "STRENGTH", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.IsValid()
			if got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "empty string is valid", input: "", expected: true},
		{name: "valid https URL", input: "https://example.com/video.mp4", expected: true},
		{name: "valid http URL", input: "http://example.com/image.jpg", expected: true},
		{name: "valid URL with port", input: "https://example.com:8080/video.mp4", expected: true},
		{name: "valid URL with query params", input: "https://youtube.com/watch?v=abc", expected: true},
		{name: "no scheme is invalid", input: "example.com/video.mp4", expected: false},
		{name: "no host is invalid", input: "https://", expected: false},
		{name: "just path is invalid", input: "/video.mp4", expected: false},
		{name: "relative path is invalid", input: "path/to/file.jpg", expected: false},
		{name: "javascript scheme is invalid", input: "javascript:alert(1)", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateURL(tt.input)
			if got != tt.expected {
				t.Errorf("ValidateURL(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
