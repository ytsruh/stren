// Package routes: api_v1.go contains every JSON handler for the
// /api/v1/* namespace. Each handler is intentionally thin: it
// binds the request, validates it, calls the same controller
// method the HTML route uses, and translates the result into a
// DTO. The web app's behavior is unchanged.
package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/models"
)

// --- Auth ---

// APILogin handles POST /api/v1/auth/login. Returns the JWT in
// the response body (not a cookie) so native clients can store
// it in the Keychain. Validation errors return a 400 with the
// same APIError shape the rest of the namespace uses.
func (h *Handler) APILogin(c echo.Context) error {
	var in LoginRequest
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: "invalid request body"})
	}
	if err := h.validator.ValidateStruct(&in); err != nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: friendlyValidationError(err)})
	}

	user, token, err := h.authCtrl.Login(in.Email, in.Password)
	if err != nil {
		if err == controllers.ErrInvalidCredentials {
			return c.JSON(http.StatusUnauthorized, APIError{Error: "invalid email or password"})
		}
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to create session"})
	}
	return c.JSON(http.StatusOK, AuthResponse{Token: token, User: UserFromModel(user)})
}

// APIRegister handles POST /api/v1/auth/register. Same
// behavior as APILogin for the response shape; the controller
// fires the welcome email on a goroutine and we don't wait for
// it before responding.
func (h *Handler) APIRegister(c echo.Context) error {
	var in RegisterRequest
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: "invalid request body"})
	}
	if err := h.validator.ValidateStruct(&in); err != nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: friendlyValidationError(err)})
	}

	user, token, err := h.authCtrl.Register(in.Name, in.Email, in.Password)
	if err != nil {
		if err == controllers.ErrEmailExists {
			return c.JSON(http.StatusConflict, APIError{Error: "an account with that email already exists"})
		}
		return c.JSON(http.StatusBadRequest, APIError{Error: err.Error()})
	}
	return c.JSON(http.StatusOK, AuthResponse{Token: token, User: UserFromModel(user)})
}

// APILogout handles POST /api/v1/auth/logout. Stateless: the
// server has nothing to do because the JWT is self-contained.
// The client discards its token. Returns 204 No Content so the
// iOS app can simply ignore the response.
func (h *Handler) APILogout(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// --- Current user ---

// APIMe handles GET /api/v1/me. Returns the safe public view
// of the authenticated user, read from the user repository.
// Useful as a "is my token still valid?" check on app launch.
func (h *Handler) APIMe(c echo.Context) error {
	claims := GetClaims(c)
	user, err := h.userRepo.GetUserByID(claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to load user"})
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, APIError{Error: "user not found"})
	}
	return c.JSON(http.StatusOK, UserFromModel(user))
}

// --- Exercises ---

// APIListExercises handles GET /api/v1/exercises. Returns
// every exercise in the catalogue. The iOS app uses this to
// populate the picker on the new-set screen.
func (h *Handler) APIListExercises(c echo.Context) error {
	exercises, err := h.exerciseEntryCtrl.List()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to load exercises"})
	}
	return c.JSON(http.StatusOK, ExercisesFromModels(exercises))
}

// --- Exercise entries (sets) ---

// APIListExerciseEntries handles GET /api/v1/exercise-entries.
// Accepts an optional ?days=N query parameter (default 7,
// clamped to [1, 365]) and returns the user's sets from the
// last N days, newest first. Same data the dashboard shows
// for the web app.
func (h *Handler) APIListExerciseEntries(c echo.Context) error {
	claims := GetClaims(c)

	days := 7
	if raw := c.QueryParam("days"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return c.JSON(http.StatusBadRequest, APIError{Error: "days must be a positive integer"})
		}
		if n > 365 {
			n = 365
		}
		days = n
	}

	var (
		entries []models.ExerciseEntry
		err     error
	)
	if days == 7 {
		// Fast path: reuses the existing "last 7 days" query,
		// which is the dashboard's hot path.
		entries, err = h.exerciseEntryCtrl.ListExerciseEntriesLast7Days(claims.UserID)
	} else {
		end := time.Now()
		start := end.AddDate(0, 0, -days)
		entries, err = h.exerciseEntryCtrl.GetExerciseEntriesByDateRange(start, end, claims.UserID)
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to load exercise entries"})
	}
	return c.JSON(http.StatusOK, ExerciseEntriesFromModels(entries))
}

// APIGetExerciseEntry handles GET /api/v1/exercise-entries/:id.
// Returns a single set scoped to the authenticated user.
func (h *Handler) APIGetExerciseEntry(c echo.Context) error {
	claims := GetClaims(c)
	id := c.Param("id")

	entry, err := h.exerciseEntryCtrl.GetExerciseEntry(id, claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to load exercise entry"})
	}
	if entry == nil {
		return c.JSON(http.StatusNotFound, APIError{Error: "exercise entry not found"})
	}
	return c.JSON(http.StatusOK, ExerciseEntryFromModel(*entry))
}

// APICreateExerciseEntries handles POST /api/v1/exercise-entries.
// Mirrors the web form's "create one or more sets at once"
// behavior: the body contains an array of sets, and a single
// exercise entry is created per set, all sharing the same
// exercise, notes, and createdAt timestamp.
func (h *Handler) APICreateExerciseEntries(c echo.Context) error {
	var in CreateExerciseEntriesRequest
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: "invalid request body"})
	}
	if err := h.validator.ValidateStruct(&in); err != nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: friendlyValidationError(err)})
	}

	claims := GetClaims(c)

	// Confirm the exercise actually exists. Without this check
	// the controller would happily create a set linked to a
	// non-existent exercise ID — fine for a typo on the iOS
	// app's side, but the user would never see the set in the
	// history view (which filters by exercise ID).
	exercise, err := h.exerciseEntryCtrl.GetExerciseByID(in.ExerciseID, claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to load exercise"})
	}
	if exercise == nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: "exercise not found"})
	}

	createdAt := time.Now()
	if in.CreatedAt != nil {
		createdAt = *in.CreatedAt
	}

	sets := make([]controllers.ExerciseSetInput, 0, len(in.Sets))
	for _, s := range in.Sets {
		sets = append(sets, controllers.ExerciseSetInput{
			Reps:     s.Reps,
			Weight:   s.Weight,
			RestTime: s.RestTime,
		})
	}

	created, err := h.exerciseEntryCtrl.CreateExerciseEntries(claims.UserID, in.ExerciseID, in.Notes, createdAt, sets)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to save exercise entry"})
	}
	return c.JSON(http.StatusCreated, ExerciseEntriesFromModels(created))
}

// APIUpdateExerciseEntry handles PUT /api/v1/exercise-entries/:id.
// Single-set update, matching the existing HTML PUT handler.
func (h *Handler) APIUpdateExerciseEntry(c echo.Context) error {
	var in UpdateExerciseEntryRequest
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: "invalid request body"})
	}
	if err := h.validator.ValidateStruct(&in); err != nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: friendlyValidationError(err)})
	}

	claims := GetClaims(c)
	id := c.Param("id")

	existing, err := h.exerciseEntryCtrl.GetExerciseEntry(id, claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to load exercise entry"})
	}
	if existing == nil {
		return c.JSON(http.StatusNotFound, APIError{Error: "exercise entry not found"})
	}

	createdAt := existing.CreatedAt
	if in.CreatedAt != nil {
		createdAt = *in.CreatedAt
	}

	updated, err := h.exerciseEntryCtrl.UpdateExerciseEntry(id, claims.UserID, in.ExerciseID, in.Notes, in.Reps, in.Weight, in.RestTime, createdAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to update exercise entry"})
	}
	return c.JSON(http.StatusOK, ExerciseEntryFromModel(*updated))
}

// APIDeleteExerciseEntry handles DELETE /api/v1/exercise-entries/:id.
// Returns 204 on success so the iOS app can simply call it and
// refresh the list.
func (h *Handler) APIDeleteExerciseEntry(c echo.Context) error {
	claims := GetClaims(c)
	id := c.Param("id")

	if err := h.exerciseEntryCtrl.DeleteExerciseEntry(id, claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to delete exercise entry"})
	}
	return c.NoContent(http.StatusNoContent)
}

// --- Exercise history & chart ---

// APIGetExerciseHistory handles GET /api/v1/exercises/:id/history.
// Accepts an optional ?page=N (1-indexed, clamped to 1) and
// returns a page of sets for the given exercise together with
// the lifetime stats (max weight, last set).
func (h *Handler) APIGetExerciseHistory(c echo.Context) error {
	claims := GetClaims(c)
	id := c.Param("id")

	exercise, err := h.exerciseEntryCtrl.GetExerciseByID(id, claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to load exercise"})
	}
	if exercise == nil {
		return c.JSON(http.StatusNotFound, APIError{Error: "exercise not found"})
	}

	page := parsePage(c.QueryParam("page"))
	history, err := h.exerciseEntryCtrl.GetExerciseEntriesByExercise(exercise.ID, claims.UserID, page)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to load history"})
	}
	return c.JSON(http.StatusOK, HistoryPageFromModel(history))
}

// APIGetExerciseChartData handles GET /api/v1/exercises/:id/chart.
// Returns every exercise entry for the given exercise so the
// iOS app can plot the line chart locally with Swift Charts.
// No pagination — the iOS chart needs the full series.
func (h *Handler) APIGetExerciseChartData(c echo.Context) error {
	claims := GetClaims(c)
	id := c.Param("id")

	exercise, err := h.exerciseEntryCtrl.GetExerciseByID(id, claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to load exercise"})
	}
	if exercise == nil {
		return c.JSON(http.StatusNotFound, APIError{Error: "exercise not found"})
	}

	entries, err := h.exerciseEntryCtrl.GetAllExerciseEntriesForChart(exercise.ID, claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIError{Error: "failed to load chart data"})
	}
	return c.JSON(http.StatusOK, ExerciseEntriesFromModels(entries))
}
