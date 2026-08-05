// Package routes provides HTTP route handlers for the strength tracker application.
// This package wraps the Echo framework and contains only HTTP-level concerns:
// request parsing, response rendering, redirects, and middleware.
// All business logic lives in the controllers package.
package routes

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"stren/internal/controllers"
	"stren/internal/models"
	"stren/internal/utils"
	"stren/internal/views"
)

// Handler holds dependencies for HTTP route handlers.
type Handler struct {
	authCtrl               *controllers.AuthController
	authRecoveryCtrl       *controllers.AuthRecoveryController
	exerciseEntryCtrl      *controllers.ExerciseEntryController
	adminCtrl              *controllers.AdminController
	adminUserCtrl          *controllers.AdminUserController
	feedbackCtrl           *controllers.FeedbackController
	timersCtrl             *controllers.TimersController
	weightCtrl             *controllers.WeightController
	pushCtrl               *controllers.PushController
	adminNotificationsCtrl *controllers.AdminNotificationsController
	goalsCtrl              *controllers.GoalsController
	pushRepo               models.PushSubscriptionRepo
	vapidPublicKey         string
	pushConfigured         bool
	userRepo               models.UserRepo
	jwtService             *utils.JWTService
	validator              utils.Validator
	// clock is the time source the profile route uses
	// when computing the next reminder fire time on form
	// save. Tests substitute a fixed clock to assert on
	// the exact value written to next_fire_at; production
	// uses a wall clock.
	clock Clock
	// imageProcessor and imageUploader are the dependencies for
	// the admin exercise image upload route. Both are interfaces
	// (defined in admin_images.go) so tests can substitute fakes
	// without touching the real S3 client or imaging package.
	imageProcessor exerciseImageProcessor
	imageUploader  exerciseImageUploader
	// imageConfig controls the two variants produced per upload.
	imageConfig ExerciseImageConfig
}

// Clock is the time source the profile route uses to stamp
// the next reminder fire. RealClock is the production default;
// tests inject a fixed clock to keep assertions deterministic.
type Clock interface {
	Now() time.Time
}

// RealClock returns the wall clock.
type RealClock struct{}

// Now returns time.Now().
func (RealClock) Now() time.Time { return time.Now() }

// NewHandler creates a new route handler instance.
func NewHandler(authCtrl *controllers.AuthController, authRecoveryCtrl *controllers.AuthRecoveryController, exerciseEntryCtrl *controllers.ExerciseEntryController, adminCtrl *controllers.AdminController, adminUserCtrl *controllers.AdminUserController, feedbackCtrl *controllers.FeedbackController, timersCtrl *controllers.TimersController, weightCtrl *controllers.WeightController, pushCtrl *controllers.PushController, adminNotificationsCtrl *controllers.AdminNotificationsController, goalsCtrl *controllers.GoalsController, pushRepo models.PushSubscriptionRepo, vapidPublicKey string, pushConfigured bool, userRepo models.UserRepo, jwtService *utils.JWTService, validator utils.Validator, imageProcessor exerciseImageProcessor, imageUploader exerciseImageUploader, imageConfig ExerciseImageConfig) *Handler {
	return &Handler{
		authCtrl:               authCtrl,
		authRecoveryCtrl:       authRecoveryCtrl,
		exerciseEntryCtrl:      exerciseEntryCtrl,
		adminCtrl:              adminCtrl,
		adminUserCtrl:          adminUserCtrl,
		feedbackCtrl:           feedbackCtrl,
		timersCtrl:             timersCtrl,
		weightCtrl:             weightCtrl,
		pushCtrl:               pushCtrl,
		adminNotificationsCtrl: adminNotificationsCtrl,
		goalsCtrl:              goalsCtrl,
		pushRepo:               pushRepo,
		vapidPublicKey:         vapidPublicKey,
		pushConfigured:         pushConfigured,
		userRepo:               userRepo,
		jwtService:             jwtService,
		validator:              validator,
		clock:                  RealClock{},
		imageProcessor:         imageProcessor,
		imageUploader:          imageUploader,
		imageConfig:            imageConfig,
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

	// Password recovery
	e.GET("/forgot", h.ForgotPasswordForm)
	e.POST("/forgot", h.RequestPasswordReset)
	e.GET("/reset", h.ResetPasswordForm)
	e.POST("/reset", h.ResetPassword)

	// Routes
	e.GET("/", h.Dashboard)

	// User profile
	e.GET("/profile", h.Profile)
	e.POST("/profile", h.UpdateProfile)

	// Exercise entry CRUD
	e.GET("/exercise-entries/new", h.NewExerciseEntryForm)
	e.POST("/exercise-entries", h.CreateExerciseEntry)
	e.GET("/exercise-entries/:id/edit", h.EditExerciseEntryForm)
	e.GET("/exercise-entries/:id", h.GetExerciseEntry)
	e.PUT("/exercise-entries/:id", h.UpdateExerciseEntry)
	e.DELETE("/exercise-entries/:id", h.DeleteExerciseEntry)

	// Exercise history
	e.GET("/exercises", h.ListExercisesUI)
	e.GET("/exercises/:id", h.ExerciseHistory)

	// Exercise chart views
	e.GET("/exercises/:id/chart", h.ExerciseChart)
	e.GET("/exercises/:id/chart/advanced", h.ExerciseChartAdvanced)

	// New set pre-filled for a specific exercise
	e.GET("/exercises/:id/new", h.NewExerciseEntryForm)

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
	e.GET("/weight/compare-modal", h.CompareWeightModal)
	e.GET("/weight/export", h.ExportWeightZip)
	e.GET("/weight/:id/edit", h.EditWeightForm)
	e.PUT("/weight/:id", h.UpdateWeight)
	e.DELETE("/weight/:id", h.DeleteWeight)

	// Photo upload (presigned URL for direct browser → R2)
	e.POST("/api/weight/photo-upload", h.PhotoUploadURL)

	// Goals
	e.GET("/goals", h.GoalsPage)
	e.GET("/goals/new", h.NewGoalForm)
	e.POST("/goals", h.CreateGoal)
	e.GET("/goals/:id/edit", h.EditGoalForm)
	e.PUT("/goals/:id", h.UpdateGoal)
	e.POST("/goals/:id/complete", h.MarkGoalComplete)
	e.POST("/goals/:id/reopen", h.ReopenGoal)
	e.DELETE("/goals/:id", h.DeleteGoal)

	// Push subscription endpoints (authenticated)
	e.POST("/api/push/subscribe", h.PushSubscribe)
	e.DELETE("/api/push/unsubscribe", h.PushUnsubscribe)

	// Admin routes
	admin := e.Group("/admin", AdminMiddleware())
	admin.GET("/exercises", h.AdminListExercises)
	admin.GET("/exercises/new", h.AdminNewExerciseForm)
	admin.POST("/exercises", h.AdminCreateExercise)
	admin.GET("/exercises/:id/edit", h.AdminEditExerciseForm)
	admin.POST("/exercises/:id", h.AdminUpdateExercise)
	// Admin image upload (multipart, 10 MB cap applied inside the
	// handler via http.MaxBytesReader so a hostile client can't
	// exhaust memory before the multipart parser runs).
	admin.POST("/exercises/image-upload", h.AdminExerciseImageUpload)
	admin.GET("/feedback", h.AdminListFeedback)
	admin.GET("/feedback/:id", h.AdminFeedbackDetail)
	admin.POST("/feedback/:id/close", h.AdminCloseFeedback)
	admin.GET("/users", h.AdminListUsers)
	admin.GET("/notifications", h.AdminNotificationsForm)
	admin.POST("/notifications/send", h.AdminNotificationsSend)
	admin.POST("/notifications/send-weight-reminder", h.AdminNotificationsSendWeightReminder)

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
		MaxAge:   int(utils.JWTTTL.Seconds()), // matches JWT exp claim
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

// exerciseEntryFormInput represents the parsed and validated form data for an exercise entry.
type exerciseEntryFormInput struct {
	ExerciseName string  `validate:"omitempty,min=1,max=100"`
	Reps         int     `validate:"gte=1,lte=1000"`
	Weight       float64 `validate:"gte=0,lte=5000"`
	Notes        string  `validate:"max=500"`
	RestTime     int     `validate:"gte=0,lte=3600"`
}

// parseExerciseEntryForm extracts form values, converts types, and validates the input.
// It returns an HTTP error with a user-friendly message if any validation fails.
func parseExerciseEntryForm(c echo.Context, v utils.Validator) (*models.ExerciseEntry, error) {
	input := exerciseEntryFormInput{
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

// setFieldPattern matches a single set's key in the multi-set form, e.g.
// "sets[2][reps]". Capture group 1 is the index, group 2 is the field name
// ("reps", "weight", or "rest_time").
var setFieldPattern = regexp.MustCompile(`^sets\[(\d+)\]\[(reps|weight|rest_time)\]$`)

// parseExerciseEntrySets extracts the array of set inputs from the multi-set
// form payload. Form field names use PHP-style bracket notation:
// sets[N][reps], sets[N][weight], sets[N][rest_time]. Echo's c.FormValue only
// returns scalar values, so we read PostForm directly.
//
// Rows with an empty reps value are skipped (the user didn't fill them in).
// The total number of submitted rows — including empty ones — must not exceed
// views.MaxSetsPerExerciseEntry, otherwise a malicious client could submit
// an unbounded payload. Each non-empty row is validated with the same struct
// tags used by parseExerciseEntryForm so per-row limits (reps 1–1000, weight
// 0–5000, rest 0–3600) carry over for free.
func parseExerciseEntrySets(c echo.Context, v utils.Validator) ([]controllers.ExerciseSetInput, error) {
	postForm, err := c.FormParams()
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid form payload")
	}

	// Group by set index. rows[i] holds the raw strings for the i-th set row;
	// only rows with at least one field are counted toward the cap.
	rows := map[int]map[string]string{}
	indices := []int{}
	for key, values := range postForm {
		m := setFieldPattern.FindStringSubmatch(key)
		if m == nil || len(values) == 0 {
			continue
		}
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		if rows[idx] == nil {
			rows[idx] = map[string]string{}
			indices = append(indices, idx)
		}
		rows[idx][m[2]] = values[0]
	}

	// Cap by submitted row count (not just non-empty ones) to stop a hostile
	// client from sending sets[0..99999] with all reps empty.
	if len(indices) > views.MaxSetsPerExerciseEntry {
		return nil, echo.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("Maximum %d sets per exercise entry", views.MaxSetsPerExerciseEntry),
		)
	}

	// Process rows in submission order so saved exercise entries follow the
	// order the user typed them (helpful for descending weight/drop-set
	// patterns).
	sort.Ints(indices)
	sets := make([]controllers.ExerciseSetInput, 0, len(indices))
	for _, idx := range indices {
		row := rows[idx]
		repsStr := strings.TrimSpace(row["reps"])
		if repsStr == "" {
			// Empty row — user didn't fill this set in. Skip silently.
			continue
		}

		reps, err := strconv.Atoi(repsStr)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Set %d: reps must be a positive integer", idx+1))
		}

		weightStr := strings.TrimSpace(row["weight"])
		weight, err := strconv.ParseFloat(weightStr, 64)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Set %d: weight must be a valid positive number", idx+1))
		}

		restTime := 0
		if rt := strings.TrimSpace(row["rest_time"]); rt != "" {
			restTime, err = strconv.Atoi(rt)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Set %d: rest time must be a whole number of seconds", idx+1))
			}
		}

		input := exerciseEntryFormInput{Reps: reps, Weight: weight, RestTime: restTime}
		if err := v.ValidateStruct(&input); err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Set %d: %s", idx+1, friendlyValidationError(err)))
		}

		sets = append(sets, controllers.ExerciseSetInput{
			Reps:     reps,
			Weight:   weight,
			RestTime: restTime,
		})
	}

	return sets, nil
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
