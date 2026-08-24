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
			exerciseEntry := &ExerciseEntry{Weight: tt.weight}
			got := exerciseEntry.FormattedWeight(tt.unit)
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
			exerciseEntry := &ExerciseEntry{CreatedAt: tt.input}
			got := exerciseEntry.FormattedDate()
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

// --- Cardio helpers ---

func TestExerciseEntry_IsCardio(t *testing.T) {
	tests := []struct {
		name     string
		entry    ExerciseEntry
		expected bool
	}{
		{name: "cardio entry", entry: ExerciseEntry{ExerciseType: ExerciseTypeCardio}, expected: true},
		{name: "strength entry", entry: ExerciseEntry{ExerciseType: ExerciseTypeStrength}, expected: false},
		{name: "other entry", entry: ExerciseEntry{ExerciseType: ExerciseTypeOther}, expected: false},
		// Legacy rows (pre-cardio) have an empty type and must render as strength.
		{name: "empty type defaults to strength", entry: ExerciseEntry{}, expected: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.IsCardio(); got != tt.expected {
				t.Errorf("IsCardio() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{name: "zero", input: 0, expected: "00:00"},
		{name: "negative clamps to zero", input: -5, expected: "00:00"},
		{name: "under a minute", input: 45, expected: "00:45"},
		{name: "exactly a minute", input: 60, expected: "01:00"},
		{name: "25 minutes (typical run)", input: 1500, expected: "25:00"},
		{name: "over an hour switches to H:MM:SS", input: 3930, expected: "1:05:30"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatDuration(tt.input); got != tt.expected {
				t.Errorf("FormatDuration(%d) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatDistance(t *testing.T) {
	tests := []struct {
		name     string
		meters   float64
		unit     string
		expected string
	}{
		{name: "5k in km", meters: 5000, unit: "km", expected: "5.00 km"},
		{name: "5k normalised from empty unit", meters: 5000, unit: "", expected: "5.00 km"},
		{name: "5k in miles", meters: 5000, unit: "mi", expected: "3.11 mi"},
		{name: "unknown unit falls back to km", meters: 42195, unit: "yards", expected: "42.20 km"}, // marathon distance
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatDistance(tt.meters, tt.unit); got != tt.expected {
				t.Errorf("FormatDistance(%v, %q) = %q, want %q", tt.meters, tt.unit, got, tt.expected)
			}
		})
	}
}

func TestExerciseEntry_PaceSecPerKm(t *testing.T) {
	tests := []struct {
		name     string
		entry    ExerciseEntry
		expected float64
	}{
		{
			name:     "25 min for 5 km is 300 sec/km",
			entry:    ExerciseEntry{DurationSeconds: 1500, DistanceMeters: 5000},
			expected: 300,
		},
		{name: "no distance means no pace", entry: ExerciseEntry{DurationSeconds: 1500}, expected: 0},
		{name: "no duration means no pace", entry: ExerciseEntry{DistanceMeters: 5000}, expected: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entry.PaceSecPerKm()
			if got != tt.expected {
				t.Errorf("PaceSecPerKm() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatPace(t *testing.T) {
	tests := []struct {
		name      string
		secPerKm  float64
		unit      string
		expected  string
	}{
		{name: "5:00 /km", secPerKm: 300, unit: "km", expected: "5:00 /km"},
		{name: "sub-minute seconds are zero padded", secPerKm: 298.4, unit: "km", expected: "4:58 /km"},
		{name: "mi converts km pace to mile pace", secPerKm: 300, unit: "mi", expected: "8:02 /mi"},
		{name: "non-positive pace renders empty", secPerKm: 0, unit: "km", expected: ""},
		{name: "negative pace renders empty", secPerKm: -10, unit: "km", expected: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatPace(tt.secPerKm, tt.unit); got != tt.expected {
				t.Errorf("FormatPace(%v, %q) = %q, want %q", tt.secPerKm, tt.unit, got, tt.expected)
			}
		})
	}
}

func TestNormalizeDistanceUnit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "km passes through", input: "km", expected: DistanceUnitKm},
		{name: "mi passes through", input: "mi", expected: DistanceUnitMi},
		{name: "empty defaults to km", input: "", expected: DistanceUnitKm},
		{name: "unknown defaults to km", input: "miles", expected: DistanceUnitKm},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeDistanceUnit(tt.input); got != tt.expected {
				t.Errorf("NormalizeDistanceUnit(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestUser_DistanceUnitDisplay(t *testing.T) {
	var nilUser *User
	if got := nilUser.DistanceUnitDisplay(); got != DistanceUnitKm {
		t.Errorf("nil user DistanceUnitDisplay() = %q, want km", got)
	}
	u := &User{DistanceUnit: "mi"}
	if got := u.DistanceUnitDisplay(); got != DistanceUnitMi {
		t.Errorf("DistanceUnitDisplay() = %q, want mi", got)
	}
	u = &User{DistanceUnit: "nautical-miles"}
	if got := u.DistanceUnitDisplay(); got != DistanceUnitKm {
		t.Errorf("unknown unit DistanceUnitDisplay() = %q, want km fallback", got)
	}
}

func TestExerciseEntry_Summary(t *testing.T) {
	strength := &ExerciseEntry{Reps: 5, Weight: 100, ExerciseType: ExerciseTypeStrength}
	if got := strength.Summary("kg", "km"); got != "5 × 100.0 kg" {
		t.Errorf("strength Summary() = %q, want %q", got, "5 × 100.0 kg")
	}

	cardio := &ExerciseEntry{DurationSeconds: 1500, DistanceMeters: 5200, ExerciseType: ExerciseTypeCardio}
	if got := cardio.Summary("kg", "km"); got != "25:00 · 5.20 km" {
		t.Errorf("cardio Summary() = %q, want %q", got, "25:00 · 5.20 km")
	}

	// Empty type (legacy rows) renders the strength summary.
	legacy := &ExerciseEntry{Reps: 8, Weight: 60}
	if got := legacy.Summary("lbs", "mi"); got != "8 × 60.0 lbs" {
		t.Errorf("legacy Summary() = %q, want %q", got, "8 × 60.0 lbs")
	}
}
