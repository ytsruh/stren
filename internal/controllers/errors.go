// Package controllers provides business logic for the strength tracker application.
package controllers

import "errors"

// Common controller errors.
var (
	// ErrInvalidCredentials is returned when login credentials are incorrect.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrEmailExists is returned when attempting to register with an already-used email.
	ErrEmailExists = errors.New("email already registered")

	// ErrExerciseNameExists is returned when attempting to create or rename
	// an exercise to a name that's already in use by another exercise.
	ErrExerciseNameExists = errors.New("an exercise with this name already exists")
)
