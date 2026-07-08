// Package routes provides HTTP route handlers for the strength tracker application.
package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"stren/internal/views"
	"stren/internal/views/dashboard"
	exerciseviews "stren/internal/views/exercise"
)

// --- Exercise Entry Handlers ---

// Dashboard renders the main dashboard page.
func (h *Handler) Dashboard(c echo.Context) error {
	claims := GetClaims(c)
	exerciseEntries, err := h.exerciseEntryCtrl.ListExerciseEntriesLast7Days(claims.UserID)
	if err != nil {
		return err
	}

	unit := "kg"
	if user := h.GetUser(c); user != nil {
		unit = user.WeightUnitDisplay()
	}

	return render(c, dashboard.Dashboard(exerciseEntries, claims.Name, true, claims.IsAdmin, unit))
}

// NewExerciseEntryForm renders the form for creating a new set.
// When reached via /exercises/:id/new the path param preselects that exercise.
func (h *Handler) NewExerciseEntryForm(c echo.Context) error {
	exercises, err := h.exerciseEntryCtrl.List()
	if err != nil {
		return err
	}

	claims := GetClaims(c)

	preselectedID := ""
	if id := c.Param("id"); id != "" {
		exercise, err := h.exerciseEntryCtrl.GetExerciseByID(id, claims.UserID)
		if err != nil {
			return err
		}
		if exercise != nil {
			preselectedID = exercise.ID
		}
	}

	unit := "kg"
	if user := h.GetUser(c); user != nil {
		unit = user.WeightUnitDisplay()
	}

	return render(c, exerciseviews.ExerciseEntryForm(exercises, preselectedID, claims.Name, true, claims.IsAdmin, unit))
}

// CreateExerciseEntry handles the creation of a new set (with one or more entries).
func (h *Handler) CreateExerciseEntry(c echo.Context) error {
	exerciseID := c.FormValue("exercise_id")
	if exerciseID == "" {
		return render(c, exerciseviews.ExerciseEntryFormError("Exercise is required"))
	}

	sets, err := parseExerciseEntrySets(c, h.validator)
	if err != nil {
		return render(c, exerciseviews.ExerciseEntryFormError(friendlyError(err)))
	}
	if len(sets) == 0 {
		return render(c, exerciseviews.ExerciseEntryFormError("Add at least one set"))
	}

	claims := GetClaims(c)
	notes := c.FormValue("notes")

	// created_at is rendered as datetime-local; the form field carries the
	// user's wall-clock pick. We default to time.Now() so an empty/absent
	// value preserves the "log it now" behaviour. Parse errors surface as a
	// user-visible error rather than silently falling back, since the user
	// clearly intended to set a date.
	createdAt := time.Now()
	if dateStr := c.FormValue("created_at"); dateStr != "" {
		parsed, parseErr := time.Parse("2006-01-02T15:04", dateStr)
		if parseErr != nil {
			return render(c, exerciseviews.ExerciseEntryFormError("Invalid date format"))
		}
		createdAt = parsed
	}

	created, err := h.exerciseEntryCtrl.CreateExerciseEntries(claims.UserID, exerciseID, notes, createdAt, sets)
	if err != nil {
		return render(c, exerciseviews.ExerciseEntryFormError("Failed to save set: "+err.Error()))
	}

	// Check if htmx request
	if c.Request().Header.Get("HX-Request") == "true" {
		msg := "Set saved!"
		if len(created) > 1 {
			msg = fmt.Sprintf("%d sets saved!", len(created))
		}
		c.Response().Header().Set("HX-Trigger", `{"triggerRedirect": "/"}`)
		return render(c, exerciseviews.ExerciseEntryFormSuccessToast(msg))
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// EditExerciseEntryForm renders the form for editing a set.
func (h *Handler) EditExerciseEntryForm(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	exerciseEntry, err := h.exerciseEntryCtrl.GetExerciseEntry(id, claims.UserID)
	if err != nil {
		return err
	}
	if exerciseEntry == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Set not found")
	}

	unit := "kg"
	if user := h.GetUser(c); user != nil {
		unit = user.WeightUnitDisplay()
	}

	return render(c, exerciseviews.EditExerciseEntryForm(exerciseEntry, claims.Name, true, claims.IsAdmin, unit))
}

// GetExerciseEntry returns a single exercise entry (for API/hx-get).
func (h *Handler) GetExerciseEntry(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	exerciseEntry, err := h.exerciseEntryCtrl.GetExerciseEntry(id, claims.UserID)
	if err != nil {
		return err
	}
	if exerciseEntry == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Set not found")
	}

	unit := "kg"
	if user := h.GetUser(c); user != nil {
		unit = user.WeightUnitDisplay()
	}

	return render(c, dashboard.ExerciseEntryRow(*exerciseEntry, unit))
}

// UpdateExerciseEntry handles updating an existing exercise entry.
func (h *Handler) UpdateExerciseEntry(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)

	existing, err := h.exerciseEntryCtrl.GetExerciseEntry(id, claims.UserID)
	if err != nil {
		return err
	}
	if existing == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Set not found")
	}

	exerciseEntry, err := parseExerciseEntryForm(c, h.validator)
	if err != nil {
		return render(c, exerciseviews.ExerciseEntryFormError(friendlyError(err)))
	}

	createdAt := existing.CreatedAt

	if dateStr := c.FormValue("created_at"); dateStr != "" {
		createdAt, err = time.Parse("2006-01-02T15:04", dateStr)
		if err != nil {
			return render(c, exerciseviews.ExerciseEntryFormError("Invalid date format"))
		}
	}

	_, err = h.exerciseEntryCtrl.UpdateExerciseEntry(id, claims.UserID, existing.ExerciseID, exerciseEntry.Notes, exerciseEntry.Reps, exerciseEntry.Weight, exerciseEntry.RestTime, createdAt)
	if err != nil {
		return render(c, exerciseviews.ExerciseEntryFormError("Failed to update set: "+err.Error()))
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return render(c, exerciseviews.ExerciseEntryFormSuccessToast("Set updated!"))
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// DeleteExerciseEntry handles deleting an exercise entry.
func (h *Handler) DeleteExerciseEntry(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	if err := h.exerciseEntryCtrl.DeleteExerciseEntry(id, claims.UserID); err != nil {
		return err
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", `{"triggerRedirect": "/"}`)
		return render(c, views.Toast("success", "Set deleted!", ""))
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// ExerciseHistory shows all exercise entries for a specific exercise.
//
// Reads the optional ?page=N query parameter (1-indexed, clamped to 1) and
// returns a paginated page of exercise entries together with lifetime stats.
// When the request comes from htmx (HX-Request: true), the response is the
// table fragment only so it can swap into the existing #history-table-wrap
// region.
func (h *Handler) ExerciseHistory(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	exercise, err := h.exerciseEntryCtrl.GetExerciseByID(id, claims.UserID)
	if err != nil {
		return err
	}

	page := parsePage(c.QueryParam("page"))

	history, err := h.exerciseEntryCtrl.GetExerciseEntriesByExercise(exercise.ID, claims.UserID, page)
	if err != nil {
		return err
	}

	unit := "kg"
	if user := h.GetUser(c); user != nil {
		unit = user.WeightUnitDisplay()
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("Vary", "HX-Request")
		return render(c, exerciseviews.HistoryTable(exercise.ID, history, unit))
	}
	chartExerciseEntries, err := h.exerciseEntryCtrl.GetRecentExerciseEntriesForChart(exercise.ID, claims.UserID)
	if err != nil {
		return err
	}
	return render(c, exerciseviews.ExerciseHistory(exercise, history, chartExerciseEntries, claims.Name, true, claims.IsAdmin, unit))
}

// ExerciseChart renders the dedicated chart view for a specific exercise:
// a full-width line chart of the user's full workout history for that
// exercise, with the shared ExerciseNav button group at the top so the
// user can switch to the History or Advanced sub-views.
//
// All of the user's exercise entries for the exercise are fetched (uncapped)
// and handed to the view, which aggregates to "heaviest weight per calendar
// day" before plotting. With fewer than 2 unique days the view shows a
// short empty-state message instead of a chart.
func (h *Handler) ExerciseChart(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	exercise, err := h.exerciseEntryCtrl.GetExerciseByID(id, claims.UserID)
	if err != nil {
		return err
	}
	if exercise == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Exercise not found")
	}

	chartExerciseEntries, err := h.exerciseEntryCtrl.GetAllExerciseEntriesForChart(exercise.ID, claims.UserID)
	if err != nil {
		return err
	}

	unit := "kg"
	if user := h.GetUser(c); user != nil {
		unit = user.WeightUnitDisplay()
	}

	return render(c, exerciseviews.ExerciseChart(exercise, chartExerciseEntries, claims.Name, true, claims.IsAdmin, unit))
}

// ExerciseChartAdvanced renders the advanced chart view for a specific
// exercise: a full-width scatter plot of every set the user has logged,
// with reps on the x axis and weight on the y axis. All of the user's
// exercise entries for the exercise are fetched (uncapped) and handed to
// the view, which plots one translucent dot per set without per-day
// aggregation. With fewer than 2 exercise entries the view shows a short
// empty-state message instead of a chart.
func (h *Handler) ExerciseChartAdvanced(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	exercise, err := h.exerciseEntryCtrl.GetExerciseByID(id, claims.UserID)
	if err != nil {
		return err
	}
	if exercise == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Exercise not found")
	}

	chartExerciseEntries, err := h.exerciseEntryCtrl.GetAllExerciseEntriesForChart(exercise.ID, claims.UserID)
	if err != nil {
		return err
	}

	unit := "kg"
	if user := h.GetUser(c); user != nil {
		unit = user.WeightUnitDisplay()
	}

	return render(c, exerciseviews.ExerciseChartAdvanced(exercise, chartExerciseEntries, claims.Name, true, claims.IsAdmin, unit))
}

// parsePage reads the ?page=N query param and returns a clamped 1-indexed page
// number. Defaults to 1 for empty or non-numeric values; values below 1 are
// clamped to 1.
func parsePage(raw string) int {
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 1
	}
	if n < 1 {
		return 1
	}
	return n
}

// ListExercisesJSON returns all exercises as JSON (for autocomplete).
func (h *Handler) ListExercisesJSON(c echo.Context) error {
	exercises, err := h.exerciseEntryCtrl.List()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, exercises)
}

// ListExercisesUI renders the exercises list page.
func (h *Handler) ListExercisesUI(c echo.Context) error {
	claims := GetClaims(c)
	exercises, err := h.exerciseEntryCtrl.List()
	if err != nil {
		return err
	}

	return render(c, exerciseviews.ExercisesList(exercises, claims.Name, true, claims.IsAdmin))
}
