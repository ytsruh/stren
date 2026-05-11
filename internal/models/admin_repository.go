package models

// AdminRepository defines the interface for admin-only exercise operations.
// This abstraction allows handlers to be tested with mock implementations.
type AdminRepository interface {
	GetByID(id int64) (*Exercise, error)
	Update(id int64, name string) (*Exercise, error)
	CreateNoTx(name string) (int64, error)
	List() ([]Exercise, error)
}

// Compile-time check to ensure ExerciseRepository implements AdminRepository.
var _ AdminRepository = (*ExerciseRepository)(nil)