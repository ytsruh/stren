// Package handlers provides HTTP handlers for the strength tracker application.
// This package wraps Echo framework for easy replacement.
package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"stren/internal/models"
	"stren/internal/views"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	repo models.Repository
}

// NewHandler creates a new handler instance
func NewHandler(repo models.Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes registers all routes with the Echo instance
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())

	// Static files (PWA assets, icons, etc.)
	e.Static("/", "public")

	// Serve manifest with correct MIME type for PWA compatibility
	e.GET("/manifest.json", h.ServeManifest)

	// Routes
	e.GET("/", h.Dashboard)
	
	// Entry CRUD
	e.GET("/entries/new", h.NewEntryForm)
	e.POST("/entries", h.CreateEntry)
	e.GET("/entries/:id/edit", h.EditEntryForm)
	e.GET("/entries/:id", h.GetEntry)
	e.PUT("/entries/:id", h.UpdateEntry)
	e.DELETE("/entries/:id", h.DeleteEntry)
	
	// Exercise history
	e.GET("/exercises/:name", h.ExerciseHistory)
	
	// API routes for htmx
	e.GET("/api/exercises", h.ListExerciseTypes)
}

// Dashboard renders the main dashboard page
func (h *Handler) Dashboard(c echo.Context) error {
	entries, err := h.repo.ListEntries(100) // Last 100 entries
	if err != nil {
		return err
	}
	
	return render(c, views.Dashboard(entries))
}

// NewEntryForm renders the form for creating a new entry
func (h *Handler) NewEntryForm(c echo.Context) error {
	types, err := h.repo.ListTypes()
	if err != nil {
		return err
	}
	
	return render(c, views.EntryForm(nil, types, false))
}

// CreateEntry handles the creation of a new entry
func (h *Handler) CreateEntry(c echo.Context) error {
	entry, err := h.parseEntryForm(c)
	if err != nil {
		return render(c, views.EntryFormError(err.Error()))
	}
	
	entry.CreatedAt = time.Now()
	
	if err := h.repo.CreateEntry(entry); err != nil {
		return render(c, views.EntryFormError("Failed to save entry: "+err.Error()))
	}
	
	// Check if htmx request
	if c.Request().Header.Get("HX-Request") == "true" {
		return render(c, views.EntryFormSuccess("Entry saved! Form reset for next set.", true))
	}
	
	return render(c, views.EntryFormSuccess("Entry saved successfully!", false))
}

// EditEntryForm renders the form for editing an entry
func (h *Handler) EditEntryForm(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid entry ID")
	}
	
	entry, err := h.repo.GetEntry(id)
	if err != nil {
		return err
	}
	if entry == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Entry not found")
	}
	
	types, err := h.repo.ListTypes()
	if err != nil {
		return err
	}
	
	return render(c, views.EntryForm(entry, types, true))
}

// GetEntry returns a single entry (for API/hx-get)
func (h *Handler) GetEntry(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid entry ID")
	}
	
	entry, err := h.repo.GetEntry(id)
	if err != nil {
		return err
	}
	if entry == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Entry not found")
	}
	
	return render(c, views.EntryRow(*entry))
}

// UpdateEntry handles updating an existing entry
func (h *Handler) UpdateEntry(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid entry ID")
	}
	
	entry, err := h.parseEntryForm(c)
	if err != nil {
		return render(c, views.EntryFormError(err.Error()))
	}
	
	entry.ID = id
	
	// Parse date if provided (for edits)
	if dateStr := c.FormValue("created_at"); dateStr != "" {
		createdAt, err := time.Parse("2006-01-02T15:04", dateStr)
		if err != nil {
			return render(c, views.EntryFormError("Invalid date format"))
		}
		entry.CreatedAt = createdAt
	}
	
	if err := h.repo.UpdateEntryWithDate(entry); err != nil {
		return render(c, views.EntryFormError("Failed to update entry: "+err.Error()))
	}
	
	if c.Request().Header.Get("HX-Request") == "true" {
		return render(c, views.EntryFormSuccess("Entry updated!", false))
	}
	
	return c.Redirect(http.StatusSeeOther, "/")
}

// DeleteEntry handles deleting an entry
func (h *Handler) DeleteEntry(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid entry ID")
	}
	
	if err := h.repo.DeleteEntry(id); err != nil {
		return err
	}
	
	// Return empty row for htmx to swap
	return render(c, views.DeletedRow())
}

// ExerciseHistory shows all entries for a specific exercise
func (h *Handler) ExerciseHistory(c echo.Context) error {
	name := c.Param("name")
	
	entries, err := h.repo.GetEntriesByExercise(name)
	if err != nil {
		return err
	}
	
	return render(c, views.ExerciseHistory(name, entries))
}

// ListExerciseTypes returns all exercise types as JSON (for autocomplete)
func (h *Handler) ListExerciseTypes(c echo.Context) error {
	types, err := h.repo.ListTypes()
	if err != nil {
		return err
	}
	
	return c.JSON(http.StatusOK, types)
}

// ServeManifest serves the web app manifest with the correct MIME type.
// Browsers expect application/manifest+json for manifest files.
func (h *Handler) ServeManifest(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, "application/manifest+json")
	return c.File("public/manifest.json")
}

// Helper functions

func (h *Handler) parseEntryForm(c echo.Context) (*models.ExerciseEntry, error) {
	entry := &models.ExerciseEntry{
		ExerciseName: c.FormValue("exercise_name"),
		Notes:        c.FormValue("notes"),
	}
	
	if entry.ExerciseName == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Exercise name is required")
	}
	
	reps, err := strconv.Atoi(c.FormValue("reps"))
	if err != nil || reps < 1 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Reps must be a positive integer")
	}
	entry.Reps = reps
	
	weight, err := strconv.ParseFloat(c.FormValue("weight"), 64)
	if err != nil || weight < 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Weight must be a valid positive number")
	}
	entry.Weight = weight
	
	return entry, nil
}

func render(c echo.Context, component templ.Component) error {
	return component.Render(c.Request().Context(), c.Response())
}