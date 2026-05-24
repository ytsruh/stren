package models

type CreateExerciseParams struct {
	Name        string
	Description string
	VideoURL    string
	ImgURL      string
	Type        ExerciseType
}

type UpdateExerciseParams struct {
	Name        string
	Description string
	VideoURL    string
	ImgURL      string
	Type        ExerciseType
}

// AdminRepository defines the interface for admin-only exercise operations.
// This abstraction allows handlers to be tested with mock implementations.
type AdminRepository interface {
	GetByID(id string) (*Exercise, error)
	Update(id string, params UpdateExerciseParams) (*Exercise, error)
	CreateNoTx(params CreateExerciseParams) (string, error)
	List() ([]Exercise, error)
}

// Compile-time check to ensure ExerciseRepository implements AdminRepository.
var _ AdminRepository = (*ExerciseRepository)(nil)