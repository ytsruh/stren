package models

// AdminRepository defines the interface for admin-only exercise type operations.
// This abstraction allows handlers to be tested with mock implementations.
type AdminRepository interface {
	GetTypeByID(id int64) (*ExerciseType, error)
	UpdateType(id int64, name string) (*ExerciseType, error)
	CreateTypeNoTx(name string) (int64, error)
	ListTypes() ([]ExerciseType, error)
}

// Compile-time check to ensure ExerciseRepository implements AdminRepository.
var _ AdminRepository = (*ExerciseRepository)(nil)