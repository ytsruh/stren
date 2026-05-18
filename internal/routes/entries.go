// Package routes provides HTTP route handlers for the strength tracker application.
package routes

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"stren/internal/views"
)

// --- Entry Handlers ---

// Dashboard renders the main dashboard page.
func (h *Handler) Dashboard(c echo.Context) error {
	claims := GetClaims(c)
	entries, err := h.entryCtrl.ListEntriesLast30Days(claims.UserID)
	if err != nil {
		return err
	}

	return render(c, views.Dashboard(entries, claims.Name, true, claims.IsAdmin))
}

// NewEntryForm renders the form for creating a new entry.
func (h *Handler) NewEntryForm(c echo.Context) error {
	exercises, err := h.entryCtrl.List()
	if err != nil {
		return err
	}

	claims := GetClaims(c)
	return render(c, views.EntryForm(exercises, claims.Name, true, claims.IsAdmin))
}

// CreateEntry handles the creation of a new entry.
func (h *Handler) CreateEntry(c echo.Context) error {
	entry, err := parseEntryForm(c, h.validator)
	if err != nil {
		return render(c, views.EntryFormError(friendlyError(err)))
	}

	claims := GetClaims(c)
	exerciseName := c.FormValue("exercise_name")
	if exerciseName == "" {
		return render(c, views.EntryFormError("Exercise name is required"))
	}
	_, err = h.entryCtrl.CreateEntry(claims.UserID, exerciseName, entry.Notes, entry.Reps, entry.Weight, entry.RestTime)
	if err != nil {
		return render(c, views.EntryFormError("Failed to save entry: "+err.Error()))
	}

	// Check if htmx request
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/")
		c.Response().WriteHeader(http.StatusSeeOther)
		return nil
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// EditEntryForm renders the form for editing an entry.
func (h *Handler) EditEntryForm(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	entry, err := h.entryCtrl.GetEntry(id, claims.UserID)
	if err != nil {
		return err
	}
	if entry == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Entry not found")
	}

	return render(c, views.EditEntryForm(entry, claims.Name, true, claims.IsAdmin))
}

// GetEntry returns a single entry (for API/hx-get).
func (h *Handler) GetEntry(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	entry, err := h.entryCtrl.GetEntry(id, claims.UserID)
	if err != nil {
		return err
	}
	if entry == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Entry not found")
	}

	return render(c, views.EntryRow(*entry))
}

// UpdateEntry handles updating an existing entry.
func (h *Handler) UpdateEntry(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)

	existing, err := h.entryCtrl.GetEntry(id, claims.UserID)
	if err != nil {
		return err
	}
	if existing == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Entry not found")
	}

	entry, err := parseEntryForm(c, h.validator)
	if err != nil {
		return render(c, views.EntryFormError(friendlyError(err)))
	}

	createdAt := time.Now()

	if dateStr := c.FormValue("created_at"); dateStr != "" {
		createdAt, err = time.Parse("2006-01-02T15:04", dateStr)
		if err != nil {
			return render(c, views.EntryFormError("Invalid date format"))
		}
	}

	_, err = h.entryCtrl.UpdateEntry(id, claims.UserID, existing.ExerciseName, entry.Notes, entry.Reps, entry.Weight, entry.RestTime, createdAt)
	if err != nil {
		return render(c, views.EntryFormError("Failed to update entry: "+err.Error()))
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return render(c, views.EntryFormSuccessToast("Entry updated!"))
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// DeleteEntry handles deleting an entry.
func (h *Handler) DeleteEntry(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	if err := h.entryCtrl.DeleteEntry(id, claims.UserID); err != nil {
		return err
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/")
		c.Response().WriteHeader(http.StatusSeeOther)
		return render(c, views.Toast("success", "Entry deleted!", ""))
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// ExerciseHistory shows all entries for a specific exercise.
func (h *Handler) ExerciseHistory(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	exercise, err := h.entryCtrl.GetExerciseByID(id, claims.UserID)
	if err != nil {
		return err
	}

	entries, err := h.entryCtrl.GetEntriesByExercise(exercise.Name, claims.UserID)
	if err != nil {
		return err
	}

	return render(c, views.ExerciseHistory(exercise.Name, entries, claims.Name, true, claims.IsAdmin))
}

// ListExercises returns all exercises as JSON (for autocomplete).
func (h *Handler) ListExercises(c echo.Context) error {
	exercises, err := h.entryCtrl.List()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, exercises)
}
