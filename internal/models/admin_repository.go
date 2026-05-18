package models

// AdminRepository defines the interface for admin-only exercise operations.
// This abstraction allows handlers to be tested with mock implementations.
type AdminRepository interface {
	GetByID(id string) (*Exercise, error)
	Update(id string, name string) (*Exercise, error)
	CreateNoTx(name string) (string, error)
	List() ([]Exercise, error)
}

// Compile-time check to ensure ExerciseRepository implements AdminRepository.
var _ AdminRepository = (*ExerciseRepository)(nil)