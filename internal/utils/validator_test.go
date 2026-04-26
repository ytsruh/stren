package utils

import (
	"strings"
	"testing"
)

// mockValidator is a test double that returns a configurable error.
type mockValidator struct {
	err error
}

func (m *mockValidator) ValidateStruct(s interface{}) error {
	return m.err
}

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("expected non-nil validator")
	}
}

func TestDefaultValidator_ValidateStruct_Success(t *testing.T) {
	v := NewValidator()

	type ValidStruct struct {
		Name  string `validate:"required,min=1,max=50"`
		Email string `validate:"required,email"`
		Age   int    `validate:"gte=0,lte=150"`
	}

	s := ValidStruct{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
	}

	if err := v.ValidateStruct(&s); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDefaultValidator_ValidateStruct_RequiredFailure(t *testing.T) {
	v := NewValidator()

	type RequiredStruct struct {
		Name string `validate:"required"`
	}

	s := RequiredStruct{}

	err := v.ValidateStruct(&s)
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
	if !strings.Contains(err.Error(), "Name is required") {
		t.Fatalf("expected error to contain 'Name is required', got %q", err.Error())
	}
}

func TestDefaultValidator_ValidateStruct_EmailFailure(t *testing.T) {
	v := NewValidator()

	type EmailStruct struct {
		Email string `validate:"email"`
	}

	s := EmailStruct{Email: "not-an-email"}

	err := v.ValidateStruct(&s)
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
	if !strings.Contains(err.Error(), "must be a valid email address") {
		t.Fatalf("expected email validation error, got %q", err.Error())
	}
}

func TestDefaultValidator_ValidateStruct_MinFailure(t *testing.T) {
	v := NewValidator()

	type MinStruct struct {
		Password string `validate:"min=6"`
	}

	s := MinStruct{Password: "short"}

	err := v.ValidateStruct(&s)
	if err == nil {
		t.Fatal("expected error for short password")
	}
	if !strings.Contains(err.Error(), "must be at least 6") {
		t.Fatalf("expected min validation error, got %q", err.Error())
	}
}

func TestDefaultValidator_ValidateStruct_MaxFailure(t *testing.T) {
	v := NewValidator()

	type MaxStruct struct {
		Name string `validate:"max=5"`
	}

	s := MaxStruct{Name: "this is way too long"}

	err := v.ValidateStruct(&s)
	if err == nil {
		t.Fatal("expected error for long name")
	}
	if !strings.Contains(err.Error(), "must be at most 5") {
		t.Fatalf("expected max validation error, got %q", err.Error())
	}
}

func TestDefaultValidator_ValidateStruct_GteFailure(t *testing.T) {
	v := NewValidator()

	type GteStruct struct {
		Reps int `validate:"gte=1"`
	}

	s := GteStruct{Reps: 0}

	err := v.ValidateStruct(&s)
	if err == nil {
		t.Fatal("expected error for reps below minimum")
	}
	if !strings.Contains(err.Error(), "Reps must be at least 1") {
		t.Fatalf("expected gte validation error, got %q", err.Error())
	}
}

func TestDefaultValidator_ValidateStruct_LteFailure(t *testing.T) {
	v := NewValidator()

	type LteStruct struct {
		Reps int `validate:"lte=100"`
	}

	s := LteStruct{Reps: 200}

	err := v.ValidateStruct(&s)
	if err == nil {
		t.Fatal("expected error for reps above maximum")
	}
	if !strings.Contains(err.Error(), "Reps must be at most 100") {
		t.Fatalf("expected lte validation error, got %q", err.Error())
	}
}

func TestDefaultValidator_ValidateStruct_MultipleErrors(t *testing.T) {
	v := NewValidator()

	type MultiStruct struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}

	s := MultiStruct{}

	err := v.ValidateStruct(&s)
	if err == nil {
		t.Fatal("expected error for multiple failures")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Name is required") {
		t.Fatalf("expected Name error, got %q", msg)
	}
	if !strings.Contains(msg, "Email is required") {
		t.Fatalf("expected Email error, got %q", msg)
	}
}
