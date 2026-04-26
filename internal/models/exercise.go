package models

import (
	"database/sql"
	"fmt"
	"time"

	"stren/internal/db"
)

// ExerciseRepository provides CRUD operations for exercises
type ExerciseRepository struct {
	database *db.DB
}

// NewExerciseRepository creates a new exercise repository
func NewExerciseRepository(database *db.DB) *ExerciseRepository {
	return &ExerciseRepository{database: database}
}

// CreateType creates a new exercise type (normalized name)
func (r *ExerciseRepository) CreateType(tx *sql.Tx, name string) (int64, error) {
	query := `INSERT INTO exercise_types (name) VALUES (?) ON CONFLICT(name) DO UPDATE SET name=name RETURNING id`
	
	var id int64
	var err error
	
	if tx != nil {
		err = tx.QueryRow(query, name).Scan(&id)
	} else {
		err = r.database.QueryRow(query, name).Scan(&id)
	}
	
	if err != nil {
		return 0, fmt.Errorf("failed to create exercise type: %w", err)
	}
	
	return id, nil
}

// GetTypeByName gets an exercise type by name
func (r *ExerciseRepository) GetTypeByName(name string) (*ExerciseType, error) {
	var et ExerciseType
	err := r.database.QueryRow(
		`SELECT id, name FROM exercise_types WHERE name = ?`,
		name,
	).Scan(&et.ID, &et.Name)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get exercise type: %w", err)
	}
	
	return &et, nil
}

// ListTypes returns all exercise types
func (r *ExerciseRepository) ListTypes() ([]ExerciseType, error) {
	rows, err := r.database.Query(`SELECT id, name FROM exercise_types ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list exercise types: %w", err)
	}
	defer rows.Close()

	var types []ExerciseType
	for rows.Next() {
		var et ExerciseType
		if err := rows.Scan(&et.ID, &et.Name); err != nil {
			return nil, err
		}
		types = append(types, et)
	}

	return types, rows.Err()
}

// CreateEntry creates a new exercise entry
func (r *ExerciseRepository) CreateEntry(entry *ExerciseEntry) error {
	return r.database.Transaction(func(tx *sql.Tx) error {
		// Create or get exercise type
		typeID, err := r.CreateType(tx, entry.ExerciseName)
		if err != nil {
			return err
		}

		// Create entry
		result, err := tx.Exec(
			`INSERT INTO exercise_entries (exercise_type_id, reps, weight, notes, created_at) 
			 VALUES (?, ?, ?, ?, ?)`,
			typeID, entry.Reps, entry.Weight, entry.Notes, entry.CreatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to create entry: %w", err)
		}

		entry.ID, err = result.LastInsertId()
		return err
	})
}

// GetEntry gets a single entry by ID
func (r *ExerciseRepository) GetEntry(id int64) (*ExerciseEntry, error) {
	var entry ExerciseEntry
	err := r.database.QueryRow(
		`SELECT e.id, e.exercise_type_id, t.name, e.reps, e.weight, e.notes, e.created_at
		 FROM exercise_entries e
		 JOIN exercise_types t ON e.exercise_type_id = t.id
		 WHERE e.id = ?`,
		id,
	).Scan(&entry.ID, &entry.ExerciseTypeID, &entry.ExerciseName, &entry.Reps, &entry.Weight, &entry.Notes, &entry.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}

	return &entry, nil
}

// UpdateEntry updates an existing entry (without changing date)
func (r *ExerciseRepository) UpdateEntry(entry *ExerciseEntry) error {
	return r.database.Transaction(func(tx *sql.Tx) error {
		// Create or get exercise type
		typeID, err := r.CreateType(tx, entry.ExerciseName)
		if err != nil {
			return err
		}

		_, err = tx.Exec(
			`UPDATE exercise_entries 
			 SET exercise_type_id = ?, reps = ?, weight = ?, notes = ?
			 WHERE id = ?`,
			typeID, entry.Reps, entry.Weight, entry.Notes, entry.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update entry: %w", err)
		}

		return nil
	})
}

// UpdateEntryWithDate updates an existing entry including the created_at date
func (r *ExerciseRepository) UpdateEntryWithDate(entry *ExerciseEntry) error {
	return r.database.Transaction(func(tx *sql.Tx) error {
		// Create or get exercise type
		typeID, err := r.CreateType(tx, entry.ExerciseName)
		if err != nil {
			return err
		}

		_, err = tx.Exec(
			`UPDATE exercise_entries
			 SET exercise_type_id = ?, reps = ?, weight = ?, notes = ?, created_at = ?
			 WHERE id = ?`,
			typeID, entry.Reps, entry.Weight, entry.Notes, entry.CreatedAt.Format(time.RFC3339), entry.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update entry: %w", err)
		}

		return nil
	})
}

// DeleteEntry deletes an entry by ID
func (r *ExerciseRepository) DeleteEntry(id int64) error {
	_, err := r.database.Exec(`DELETE FROM exercise_entries WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete entry: %w", err)
	}
	return nil
}

// ListEntries returns entries with optional filtering
func (r *ExerciseRepository) ListEntries(limit int) ([]ExerciseEntry, error) {
	query := `SELECT e.id, e.exercise_type_id, t.name, e.reps, e.weight, e.notes, e.created_at
			  FROM exercise_entries e
			  JOIN exercise_types t ON e.exercise_type_id = t.id
			  ORDER BY e.created_at DESC`
	
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.database.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

// GetEntriesByExercise returns all entries for a specific exercise
func (r *ExerciseRepository) GetEntriesByExercise(exerciseName string) ([]ExerciseEntry, error) {
	rows, err := r.database.Query(
		`SELECT e.id, e.exercise_type_id, t.name, e.reps, e.weight, e.notes, e.created_at
		 FROM exercise_entries e
		 JOIN exercise_types t ON e.exercise_type_id = t.id
		 WHERE t.name = ?
		 ORDER BY e.created_at DESC`,
		exerciseName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get entries by exercise: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

// GetEntriesByDateRange returns entries within a date range
func (r *ExerciseRepository) GetEntriesByDateRange(start, end time.Time) ([]ExerciseEntry, error) {
	rows, err := r.database.Query(
		`SELECT e.id, e.exercise_type_id, t.name, e.reps, e.weight, e.notes, e.created_at
		 FROM exercise_entries e
		 JOIN exercise_types t ON e.exercise_type_id = t.id
		 WHERE e.created_at BETWEEN ? AND ?
		 ORDER BY e.created_at DESC`,
		start.Format(time.RFC3339), end.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get entries by date range: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

func scanEntries(rows *sql.Rows) ([]ExerciseEntry, error) {
	var entries []ExerciseEntry
	for rows.Next() {
		var e ExerciseEntry
		if err := rows.Scan(&e.ID, &e.ExerciseTypeID, &e.ExerciseName, &e.Reps, &e.Weight, &e.Notes, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
