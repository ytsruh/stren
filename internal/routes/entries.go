// Package routes provides HTTP route handlers for the strength tracker application.
package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"stren/internal/views"
)

// --- Entry Handlers ---

// Dashboard renders the main dashboard page.
func (h *Handler) Dashboard(c echo.Context) error {
	claims := GetClaims(c)
	entries, err := h.entryCtrl.ListEntriesLast7Days(claims.UserID)
	if err != nil {
		return err
	}

	return render(c, views.Dashboard(entries, claims.Name, true, claims.IsAdmin))
}

// NewEntryForm renders the form for creating a new entry.
// When reached via /exercises/:id/new the path param preselects that exercise.
func (h *Handler) NewEntryForm(c echo.Context) error {
	exercises, err := h.entryCtrl.List()
	if err != nil {
		return err
	}

	claims := GetClaims(c)

	preselectedID := ""
	if id := c.Param("id"); id != "" {
		exercise, err := h.entryCtrl.GetExerciseByID(id, claims.UserID)
		if err != nil {
			return err
		}
		if exercise != nil {
			preselectedID = exercise.ID
		}
	}

	return render(c, views.EntryForm(exercises, preselectedID, claims.Name, true, claims.IsAdmin))
}

// CreateEntry handles the creation of a new entry (with one or more sets).
func (h *Handler) CreateEntry(c echo.Context) error {
	exerciseID := c.FormValue("exercise_id")
	if exerciseID == "" {
		return render(c, views.EntryFormError("Exercise is required"))
	}

	sets, err := parseEntrySets(c, h.validator)
	if err != nil {
		return render(c, views.EntryFormError(friendlyError(err)))
	}
	if len(sets) == 0 {
		return render(c, views.EntryFormError("Add at least one set"))
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
			return render(c, views.EntryFormError("Invalid date format"))
		}
		createdAt = parsed
	}

	created, err := h.entryCtrl.CreateEntries(claims.UserID, exerciseID, notes, createdAt, sets)
	if err != nil {
		return render(c, views.EntryFormError("Failed to save entry: "+err.Error()))
	}

	// Check if htmx request
	if c.Request().Header.Get("HX-Request") == "true" {
		msg := "Entry saved!"
		if len(created) > 1 {
			msg = fmt.Sprintf("%d sets saved!", len(created))
		}
		c.Response().Header().Set("HX-Trigger", `{"triggerRedirect": "/"}`)
		return render(c, views.EntryFormSuccessToast(msg))
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

	createdAt := existing.CreatedAt

	if dateStr := c.FormValue("created_at"); dateStr != "" {
		createdAt, err = time.Parse("2006-01-02T15:04", dateStr)
		if err != nil {
			return render(c, views.EntryFormError("Invalid date format"))
		}
	}

	_, err = h.entryCtrl.UpdateEntry(id, claims.UserID, existing.ExerciseID, entry.Notes, entry.Reps, entry.Weight, entry.RestTime, createdAt)
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
		c.Response().Header().Set("HX-Trigger", `{"triggerRedirect": "/"}`)
		return render(c, views.Toast("success", "Entry deleted!", ""))
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// ExerciseHistory shows all entries for a specific exercise.
//
// Reads the optional ?page=N query parameter (1-indexed, clamped to 1) and
// returns a paginated page of entries together with lifetime stats. When the
// request comes from htmx (HX-Request: true), the response is the table
// fragment only so it can swap into the existing #history-table-wrap region.
func (h *Handler) ExerciseHistory(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	exercise, err := h.entryCtrl.GetExerciseByID(id, claims.UserID)
	if err != nil {
		return err
	}

	page := parsePage(c.QueryParam("page"))

	history, err := h.entryCtrl.GetEntriesByExercise(exercise.ID, claims.UserID, page)
	if err != nil {
		return err
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("Vary", "HX-Request")
		return render(c, views.HistoryTable(exercise.ID, history))
	}
	chartEntries, err := h.entryCtrl.GetRecentEntriesForChart(exercise.ID, claims.UserID)
	if err != nil {
		return err
	}
	return render(c, views.ExerciseHistory(exercise, history, chartEntries, claims.Name, true, claims.IsAdmin))
}

// ExerciseChart renders the dedicated chart view for a specific exercise:
// a full-width line chart of the user's full workout history for that
// exercise, with the shared ExerciseNav button group at the top so the
// user can switch to the History or Advanced sub-views.
//
// All of the user's entries for the exercise are fetched (uncapped) and
// handed to the view, which aggregates to "heaviest weight per calendar
// day" before plotting. With fewer than 2 unique days the view shows a
// short empty-state message instead of a chart.
func (h *Handler) ExerciseChart(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	exercise, err := h.entryCtrl.GetExerciseByID(id, claims.UserID)
	if err != nil {
		return err
	}
	if exercise == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Exercise not found")
	}

	chartEntries, err := h.entryCtrl.GetAllEntriesForChart(exercise.ID, claims.UserID)
	if err != nil {
		return err
	}

	return render(c, views.ExerciseChart(exercise, chartEntries, claims.Name, true, claims.IsAdmin))
}

// ExerciseChartAdvanced renders the advanced chart view for a specific
// exercise: a full-width scatter plot of every set the user has logged,
// with reps on the x axis and weight on the y axis. All of the user's
// entries for the exercise are fetched (uncapped) and handed to the
// view, which plots one translucent dot per set without per-day
// aggregation. With fewer than 2 entries the view shows a short
// empty-state message instead of a chart.
func (h *Handler) ExerciseChartAdvanced(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	exercise, err := h.entryCtrl.GetExerciseByID(id, claims.UserID)
	if err != nil {
		return err
	}
	if exercise == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Exercise not found")
	}

	chartEntries, err := h.entryCtrl.GetAllEntriesForChart(exercise.ID, claims.UserID)
	if err != nil {
		return err
	}

	return render(c, views.ExerciseChartAdvanced(exercise, chartEntries, claims.Name, true, claims.IsAdmin))
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
	exercises, err := h.entryCtrl.List()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, exercises)
}

// ListExercisesUI renders the exercises list page.
func (h *Handler) ListExercisesUI(c echo.Context) error {
	claims := GetClaims(c)
	exercises, err := h.entryCtrl.List()
	if err != nil {
		return err
	}

	return render(c, views.ExercisesList(exercises, claims.Name, true, claims.IsAdmin))
}
