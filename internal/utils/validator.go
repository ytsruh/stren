// Package utils provides shared utility functions used across the application.
package utils

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator defines the interface for struct validation used across the application.
// This abstraction allows the underlying validation library to be swapped or mocked in tests.
type Validator interface {
	ValidateStruct(s interface{}) error
}

// defaultValidator wraps go-playground/validator/v10 with a stable interface.
type defaultValidator struct {
	validate *validator.Validate
}

// NewValidator creates a new Validator instance with recommended defaults.
// It enables required struct validation for forward compatibility with v11+.
func NewValidator() Validator {
	v := validator.New(validator.WithRequiredStructEnabled())
	return &defaultValidator{validate: v}
}

// ValidateStruct validates the given struct using validation tags.
// It returns a human-readable error string if any fields fail validation.
func (dv *defaultValidator) ValidateStruct(s interface{}) error {
	if err := dv.validate.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return fmt.Errorf("%s", formatValidationErrors(validationErrors))
		}
		return err
	}
	return nil
}

// formatValidationErrors converts go-playground validation errors into a
// single human-friendly string suitable for display in UI error messages.
func formatValidationErrors(errs validator.ValidationErrors) string {
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, formatFieldError(e))
	}
	return strings.Join(msgs, "; ")
}

// formatFieldError maps a single validation failure to a readable message.
func formatFieldError(fe validator.FieldError) string {
	field := fe.Field()
	tag := fe.Tag()
	param := fe.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, param)
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, param)
	case "gte":
		return fmt.Sprintf("%s must be at least %s", field, param)
	case "lte":
		return fmt.Sprintf("%s must be at most %s", field, param)
	default:
		return fmt.Sprintf("%s failed validation: %s", field, tag)
	}
}
