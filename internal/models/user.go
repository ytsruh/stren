package models

import "time"

// User represents an authenticated user of the strength tracker.
type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
