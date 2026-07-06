package models

import (
	"testing"
	"time"
)

func TestExerciseEntry_FormattedWeight(t *testing.T) {
	tests := []struct {
		name     string
		weight   float64
		unit     string
		expected string
	}{
		{
			name:     "kg whole number",
			weight:   100,
			unit:     "kg",
			expected: "100.0 kg",
		},
		{
			name:     "kg decimal",
			weight:   67.5,
			unit:     "kg",
			expected: "67.5 kg",
		},
		{
			name:     "kg zero",
			weight:   0,
			unit:     "kg",
			expected: "0.0 kg",
		},
		{
			name:     "kg large number",
			weight:   999.9,
			unit:     "kg",
			expected: "999.9 kg",
		},
		{
			name:     "lbs whole number",
			weight:   225,
			unit:     "lbs",
			expected: "225.0 lbs",
		},
		{
			name:     "lbs decimal",
			weight:   67.5,
			unit:     "lbs",
			expected: "67.5 lbs",
		},
		{
			name:     "lbs zero",
			weight:   0,
			unit:     "lbs",
			expected: "0.0 lbs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &ExerciseEntry{Weight: tt.weight}
			got := entry.FormattedWeight(tt.unit)
			if got != tt.expected {
				t.Errorf("FormattedWeight(%q) = %q, want %q", tt.unit, got, tt.expected)
			}
		})
	}
}

func TestFormatWeight(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		unit     string
		expected string
	}{
		{name: "kg whole", value: 100, unit: "kg", expected: "100.0 kg"},
		{name: "kg decimal", value: 67.5, unit: "kg", expected: "67.5 kg"},
		{name: "lbs whole", value: 225, unit: "lbs", expected: "225.0 lbs"},
		{name: "lbs decimal", value: 182.5, unit: "lbs", expected: "182.5 lbs"},
		// FormatWeight does not normalise; callers are expected to
		// pass a value from NormalizeWeightUnit (or User.WeightUnitDisplay)
		// so an empty / unrecognised unit is the caller's bug to surface.
		{name: "empty unit is passed through", value: 100, unit: "", expected: "100.0 "},
		{name: "unrecognised unit is passed through", value: 100, unit: "stones", expected: "100.0 stones"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatWeight(tt.value, tt.unit)
			if got != tt.expected {
				t.Errorf("FormatWeight(%v, %q) = %q, want %q", tt.value, tt.unit, got, tt.expected)
			}
		})
	}
}

func TestNormalizeWeightUnit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "kg", input: "kg", expected: "kg"},
		{name: "lbs", input: "lbs", expected: "lbs"},
		{name: "empty falls back to kg", input: "", expected: "kg"},
		{name: "unrecognised falls back to kg", input: "stones", expected: "kg"},
		{name: "uppercase is not normalised", input: "KG", expected: "kg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeWeightUnit(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeWeightUnit(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestUser_WeightUnitDisplay(t *testing.T) {
	tests := []struct {
		name     string
		user     *User
		expected string
	}{
		{name: "nil user falls back to kg", user: nil, expected: "kg"},
		{name: "stored kg", user: &User{WeightUnit: "kg"}, expected: "kg"},
		{name: "stored lbs", user: &User{WeightUnit: "lbs"}, expected: "lbs"},
		{name: "empty falls back to kg", user: &User{WeightUnit: ""}, expected: "kg"},
		{name: "unrecognised falls back to kg", user: &User{WeightUnit: "stones"}, expected: "kg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.WeightUnitDisplay()
			if got != tt.expected {
				t.Errorf("WeightUnitDisplay() = %q, want %q", got, tt.expected)
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
