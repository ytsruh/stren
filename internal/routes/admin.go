package routes

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/models"
	"stren/internal/views/admin"
)

// --- Admin User Handlers ---

// AdminListUsers renders the admin page listing all users.
func (h *Handler) AdminListUsers(c echo.Context) error {
	claims := GetClaims(c)
	users, err := h.adminUserCtrl.ListUsers()
	if err != nil {
		return err
	}

	return render(c, admin.AdminUserList(users, claims.Name, true, claims.IsAdmin))
}

// --- Admin Exercise Handlers ---

// AdminListExercises renders the admin page listing all exercises.
func (h *Handler) AdminListExercises(c echo.Context) error {
	claims := GetClaims(c)
	exercises, err := h.adminCtrl.List()
	if err != nil {
		return err
	}

	return render(c, admin.AdminExerciseList(exercises, claims.Name, true, claims.IsAdmin))
}

// AdminNewExerciseForm renders the form for creating a new exercise.
func (h *Handler) AdminNewExerciseForm(c echo.Context) error {
	claims := GetClaims(c)
	return render(c, admin.AdminExerciseForm(admin.AdminExerciseFormData{IsEdit: false}, claims.Name, true, claims.IsAdmin))
}

// AdminCreateExercise handles creating a new exercise.
func (h *Handler) AdminCreateExercise(c echo.Context) error {
	name := c.FormValue("name")
	if name == "" {
		return render(c, admin.AdminExerciseFormError("Exercise name is required"))
	}

	params := models.CreateExerciseParams{
		Name:        name,
		Description: c.FormValue("description"),
		VideoURL:    c.FormValue("video_url"),
		ImgURL:      c.FormValue("img_url"),
		Type:        models.ExerciseType(c.FormValue("type")),
	}

	_, err := h.adminCtrl.Create(params)
	if err != nil {
		if errors.Is(err, controllers.ErrExerciseNameExists) {
			return render(c, admin.AdminExerciseFormError("An exercise with this name already exists"))
		}
		c.Logger().Errorf("admin create exercise failed: %v", err)
		return render(c, admin.AdminExerciseFormError("Failed to save exercise. Please try again."))
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", `{"triggerRedirect": "/admin/exercises"}`)
		return render(c, admin.AdminExerciseSuccessToast("Exercise created!"))
	}
	return c.Redirect(http.StatusSeeOther, "/admin/exercises")
}

// AdminEditExerciseForm renders the form for editing an existing exercise.
func (h *Handler) AdminEditExerciseForm(c echo.Context) error {
	id := c.Param("id")

	exercise, err := h.adminCtrl.Get(id)
	if err != nil {
		if errors.Is(err, controllers.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Exercise not found")
		}
		return err
	}

	claims := GetClaims(c)
	return render(c, admin.AdminExerciseForm(admin.AdminExerciseFormData{Exercise: exercise, IsEdit: true}, claims.Name, true, claims.IsAdmin))
}

// AdminUpdateExercise handles updating an existing exercise.
func (h *Handler) AdminUpdateExercise(c echo.Context) error {
	id := c.Param("id")

	name := c.FormValue("name")
	if name == "" {
		return render(c, admin.AdminExerciseFormError("Exercise name is required"))
	}

	params := models.UpdateExerciseParams{
		Name:        name,
		Description: c.FormValue("description"),
		VideoURL:    c.FormValue("video_url"),
		ImgURL:      c.FormValue("img_url"),
		Type:        models.ExerciseType(c.FormValue("type")),
	}

	_, err := h.adminCtrl.Update(id, params)
	if err != nil {
		if errors.Is(err, controllers.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Exercise not found")
		}
		if errors.Is(err, controllers.ErrExerciseNameExists) {
			return render(c, admin.AdminExerciseFormError("An exercise with this name already exists"))
		}
		c.Logger().Errorf("admin update exercise failed: %v", err)
		return render(c, admin.AdminExerciseFormError("Failed to save exercise. Please try again."))
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", `{"triggerRedirect": "/admin/exercises"}`)
		return render(c, admin.AdminExerciseSuccessToast("Exercise updated!"))
	}
	return c.Redirect(http.StatusSeeOther, "/admin/exercises")
}