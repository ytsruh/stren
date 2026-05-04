// Package routes provides HTTP route handlers for the strength tracker application.
// This package wraps the Echo framework and contains only HTTP-level concerns:
// request parsing, response rendering, redirects, and middleware.
// All business logic lives in the controllers package.
package routes

import (
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"stren/internal/controllers"
	"stren/internal/models"
	"stren/internal/utils"
)

// Handler holds dependencies for HTTP route handlers.
type Handler struct {
	authCtrl   *controllers.AuthController
	entryCtrl  *controllers.EntryController
	jwtService *utils.JWTService
	validator  utils.Validator
}

// NewHandler creates a new route handler instance.
func NewHandler(authCtrl *controllers.AuthController, entryCtrl *controllers.EntryController, jwtService *utils.JWTService, validator utils.Validator) *Handler {
	return &Handler{
		authCtrl:   authCtrl,
		entryCtrl:  entryCtrl,
		jwtService: jwtService,
		validator:  validator,
	}
}

// RegisterRoutes registers all routes with the Echo instance.
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())
	e.Use(AuthMiddleware(h.jwtService))

	// Static files (PWA assets, icons, etc.)
	e.Static("/", "public")

	// Serve manifest with correct MIME type for PWA compatibility
	e.GET("/manifest.json", h.ServeManifest)

	// Auth routes (public, handled by auth middleware skip list)
	e.GET("/login", h.LoginForm)
	e.POST("/login", h.Login)
	e.GET("/register", h.RegisterForm)
	e.POST("/register", h.Register)
	e.POST("/logout", h.Logout)

	// Routes
	e.GET("/", h.Dashboard)

	// User profile
	e.GET("/profile", h.Profile)

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

// ServeManifest serves the web app manifest with the correct MIME type.
// Browsers expect application/manifest+json for manifest files.
func (h *Handler) ServeManifest(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, "application/manifest+json")
	return c.File("public/manifest.json")
}

// Helper functions

func render(c echo.Context, component templ.Component) error {
	return component.Render(c.Request().Context(), c.Response())
}

func setAuthCookie(c echo.Context, token string) {
	cookie := &http.Cookie{
		Name:     utils.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	}
	c.SetCookie(cookie)
}

func clearAuthCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     utils.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	c.SetCookie(cookie)
}

// entryFormInput represents the parsed and validated form data for an exercise entry.
type entryFormInput struct {
	ExerciseName string  `validate:"required,min=1,max=100"`
	Reps         int     `validate:"gte=1,lte=1000"`
	Weight       float64 `validate:"gte=0,lte=5000"`
	Notes        string  `validate:"max=500"`
}

// parseEntryForm extracts form values, converts types, and validates the input.
// It returns an HTTP error with a user-friendly message if any validation fails.
func parseEntryForm(c echo.Context, v utils.Validator) (*models.ExerciseEntry, error) {
	input := entryFormInput{
		ExerciseName: c.FormValue("exercise_name"),
		Notes:        c.FormValue("notes"),
	}

	reps, err := strconv.Atoi(c.FormValue("reps"))
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Reps must be a positive integer")
	}
	input.Reps = reps

	weight, err := strconv.ParseFloat(c.FormValue("weight"), 64)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Weight must be a valid positive number")
	}
	input.Weight = weight

	if err := v.ValidateStruct(&input); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return &models.ExerciseEntry{
		ExerciseName: input.ExerciseName,
		Notes:        input.Notes,
		Reps:         input.Reps,
		Weight:       input.Weight,
	}, nil
}
