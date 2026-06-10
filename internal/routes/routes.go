// Package routes provides HTTP route handlers for the strength tracker application.
// This package wraps the Echo framework and contains only HTTP-level concerns:
// request parsing, response rendering, redirects, and middleware.
// All business logic lives in the controllers package.
package routes

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"stren/internal/controllers"
	"stren/internal/models"
	"stren/internal/utils"
)

// Handler holds dependencies for HTTP route handlers.
type Handler struct {
	authCtrl        *controllers.AuthController
	entryCtrl       *controllers.EntryController
	adminCtrl       *controllers.AdminController
	adminUserCtrl   *controllers.AdminUserController
	feedbackCtrl    *controllers.FeedbackController
	timerCtrl       *controllers.TimerController
	emomCtrl        *controllers.EMOMController
	weightCtrl      *controllers.WeightController
	userRepo        models.UserRepo
	jwtService      *utils.JWTService
	validator       utils.Validator
}

// NewHandler creates a new route handler instance.
func NewHandler(authCtrl *controllers.AuthController, entryCtrl *controllers.EntryController, adminCtrl *controllers.AdminController, adminUserCtrl *controllers.AdminUserController, feedbackCtrl *controllers.FeedbackController, timerCtrl *controllers.TimerController, emomCtrl *controllers.EMOMController, weightCtrl *controllers.WeightController, userRepo models.UserRepo, jwtService *utils.JWTService, validator utils.Validator) *Handler {
	return &Handler{
		authCtrl:        authCtrl,
		entryCtrl:       entryCtrl,
		adminCtrl:       adminCtrl,
		adminUserCtrl:   adminUserCtrl,
		feedbackCtrl:    feedbackCtrl,
		timerCtrl:       timerCtrl,
		emomCtrl:        emomCtrl,
		weightCtrl:      weightCtrl,
		userRepo:         userRepo,
		jwtService:       jwtService,
		validator:        validator,
	}
}

// RegisterRoutes registers all routes with the Echo instance.
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())
	e.Use(AuthMiddleware(h.jwtService))

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
	e.POST("/profile", h.UpdateProfile)

	// Entry CRUD
	e.GET("/entries/new", h.NewEntryForm)
	e.POST("/entries", h.CreateEntry)
	e.GET("/entries/:id/edit", h.EditEntryForm)
	e.GET("/entries/:id", h.GetEntry)
	e.PUT("/entries/:id", h.UpdateEntry)
	e.DELETE("/entries/:id", h.DeleteEntry)

	// Exercise history
	e.GET("/exercises", h.ListExercisesUI)
	e.GET("/exercises/:id", h.ExerciseHistory)

	// Feedback
	e.GET("/feedback", h.FeedbackForm)
	e.POST("/feedback", h.SubmitFeedback)

	// Timer
	e.GET("/timer", h.TimerPage)
	e.POST("/timer/error", h.TimerValidationError)

	// EMOM
	e.GET("/timer/emom", h.EMOMPage)
	e.POST("/timer/emom/error", h.EMOMValidationError)
	e.POST("/timer/emom/round", h.EMOMRoundToast)

	// Weight entries
	e.GET("/weight", h.WeightPage)
	e.GET("/weight/new", h.NewWeightForm)
	e.POST("/weight", h.CreateWeight)
	e.GET("/weight/:id/edit", h.EditWeightForm)
	e.PUT("/weight/:id", h.UpdateWeight)
	e.DELETE("/weight/:id", h.DeleteWeight)

	// Admin routes
	admin := e.Group("/admin", AdminMiddleware())
	admin.GET("/exercises", h.AdminListExercises)
	admin.GET("/exercises/new", h.AdminNewExerciseForm)
	admin.POST("/exercises", h.AdminCreateExercise)
	admin.GET("/exercises/:id/edit", h.AdminEditExerciseForm)
	admin.POST("/exercises/:id", h.AdminUpdateExercise)
	admin.GET("/feedback", h.AdminListFeedback)
	admin.GET("/feedback/:id", h.AdminFeedbackDetail)
	admin.POST("/feedback/:id/close", h.AdminCloseFeedback)
	admin.GET("/users", h.AdminListUsers)

	// API routes for htmx
	e.GET("/api/exercises", h.ListExercisesJSON)

	// Static files (PWA assets, icons, etc.) - registered last as catch-all
	e.Static("/", "public")
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
	ExerciseName string  `validate:"omitempty,min=1,max=100"`
	Reps         int     `validate:"gte=1,lte=1000"`
	Weight       float64 `validate:"gte=0,lte=5000"`
	Notes        string  `validate:"max=500"`
	RestTime     int     `validate:"gte=0,lte=3600"`
}

// parseEntryForm extracts form values, converts types, and validates the input.
// It returns an HTTP error with a user-friendly message if any validation fails.
func parseEntryForm(c echo.Context, v utils.Validator) (*models.ExerciseEntry, error) {
	input := entryFormInput{
		Notes: c.FormValue("notes"),
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

	restTime, err := strconv.Atoi(c.FormValue("rest_time"))
	if err != nil {
		restTime = 0
	}
	input.RestTime = restTime

	if err := v.ValidateStruct(&input); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, friendlyValidationError(err))
	}

	return &models.ExerciseEntry{
		ExerciseName: input.ExerciseName,
		Notes:        input.Notes,
		Reps:         input.Reps,
		Weight:       input.Weight,
		RestTime:     input.RestTime,
	}, nil
}

// friendlyValidationError converts validation error messages to user-friendly strings.
func friendlyValidationError(err error) string {
	return friendlyError(err)
}

// friendlyError extracts the message from Echo HTTP errors and makes validation errors user-friendly.
func friendlyError(err error) string {
	msg := err.Error()
	// Strip "code=XXX, message=" prefix if present
	if strings.HasPrefix(msg, "code=") {
		if idx := strings.Index(msg, ", message="); idx != -1 {
			msg = msg[idx+len(", message="):]
		}
	}
	if strings.Contains(msg, "Reps") && strings.Contains(msg, "lte") {
		return "Reps must be 1000 or less"
	}
	if strings.Contains(msg, "Weight") && strings.Contains(msg, "lte") {
		return "Weight must be 5000 or less"
	}
	if strings.Contains(msg, "ExerciseName") && strings.Contains(msg, "min") {
		return "Exercise name is required"
	}
	if strings.Contains(msg, "Notes") && strings.Contains(msg, "max") && strings.Contains(msg, "500") {
		return "Notes must be 500 characters or less"
	}
	if strings.Contains(msg, "Notes") && strings.Contains(msg, "max") && strings.Contains(msg, "1000") {
		return "Notes must be 1000 characters or less"
	}
	if strings.Contains(msg, "Weight") && strings.Contains(msg, "lte") {
		return "Weight must be 1000 or less"
	}
	return msg
}
