package models

import (
	"testing"

	"stren/internal/db"
)

// userTestHarness returns a connected in-memory DB plus a UserRepository.
// Closing the returned database is the caller's responsibility.
func userTestHarness(t *testing.T) (*UserRepository, *db.DB) {
	t.Helper()

	database, err := db.NewLocalConnection(":memory:")
	if err != nil {
		t.Fatalf("in-memory db: %v", err)
	}

	return NewUserRepository(database), database
}

func TestUserRepository_CreateAndGet_RoundTripsWeightFields(t *testing.T) {
	repo, database := userTestHarness(t)
	defer database.Close()

	created := &User{
		Name:         "Round Trip",
		Email:        "round-trip@example.com",
		PasswordHash: "hash",
	}
	if err := repo.CreateUser(created); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	// CreateUser doesn't write the goal/unit columns — those are set via
	// the profile form (UpdateUser). Set them now and assert round-trip.
	target := 75.0
	created.TargetWeight = &target
	created.WeightUnit = "lbs"
	if err := repo.UpdateUser(created); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	// GetUserByEmail round-trips both new fields.
	got, err := repo.GetUserByEmail(created.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.TargetWeight == nil {
		t.Fatal("expected TargetWeight to be non-nil after round-trip")
	}
	if *got.TargetWeight != 75.0 {
		t.Errorf("TargetWeight = %v, want 75.0", *got.TargetWeight)
	}
	if got.WeightUnit != "lbs" {
		t.Errorf("WeightUnit = %q, want %q", got.WeightUnit, "lbs")
	}
	if !got.HasWeightGoal() {
		t.Error("expected HasWeightGoal() to be true after round-trip")
	}

	// GetUserByID returns the same shape.
	byID, err := repo.GetUserByID(created.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if byID == nil || byID.TargetWeight == nil || *byID.TargetWeight != 75.0 || byID.WeightUnit != "lbs" {
		t.Errorf("GetUserByID lost weight fields: %+v", byID)
	}
}

func TestUserRepository_DefaultsWeightFields(t *testing.T) {
	// A user created with no explicit goal/unit should come back with a nil
	// target and the default "kg" unit (matches the SQL DEFAULT and the
	// route's defaultWeightUnit).
	repo, database := userTestHarness(t)
	defer database.Close()

	created := &User{
		Name:         "Defaults",
		Email:        "defaults@example.com",
		PasswordHash: "hash",
	}
	if err := repo.CreateUser(created); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	got, err := repo.GetUserByID(created.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.TargetWeight != nil {
		t.Errorf("expected nil TargetWeight, got %v", *got.TargetWeight)
	}
	if got.HasWeightGoal() {
		t.Error("expected HasWeightGoal() to be false when target is nil")
	}
	if got.WeightUnit != "kg" {
		t.Errorf("WeightUnit = %q, want %q (SQL default)", got.WeightUnit, "kg")
	}
}

func TestUserRepository_UpdateUser_PersistsWeightFields(t *testing.T) {
	repo, database := userTestHarness(t)
	defer database.Close()

	created := &User{
		Name:         "Updater",
		Email:        "updater@example.com",
		PasswordHash: "hash",
	}
	if err := repo.CreateUser(created); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Set a goal + unit.
	first := 80.0
	created.TargetWeight = &first
	created.WeightUnit = "lbs"
	if err := repo.UpdateUser(created); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	got, err := repo.GetUserByID(created.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got == nil || got.TargetWeight == nil || *got.TargetWeight != 80.0 || got.WeightUnit != "lbs" {
		t.Fatalf("after first update: %+v", got)
	}

	// Clear the goal and switch unit back to kg. The route submits a
	// nil *float64 when the user empties the field, so the repo must
	// write SQL NULL in that case (not 0.0).
	created.TargetWeight = nil
	created.WeightUnit = "kg"
	if err := repo.UpdateUser(created); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	got, err = repo.GetUserByID(created.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.TargetWeight != nil {
		t.Errorf("expected nil TargetWeight after clearing, got %v", *got.TargetWeight)
	}
	if got.HasWeightGoal() {
		t.Error("expected HasWeightGoal() to be false after clearing the goal")
	}
	if got.WeightUnit != "kg" {
		t.Errorf("WeightUnit = %q, want %q", got.WeightUnit, "kg")
	}
}

func TestUser_HasWeightGoal(t *testing.T) {
	target := 70.0
	cases := []struct {
		name string
		user User
		want bool
	}{
		{name: "nil target", user: User{}, want: false},
		{name: "set target", user: User{TargetWeight: &target}, want: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.HasWeightGoal(); got != tt.want {
				t.Errorf("HasWeightGoal() = %v, want %v", got, tt.want)
			}
		})
	}
}
