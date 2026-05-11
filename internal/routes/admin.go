package routes

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/views"
)

// --- Admin Exercise Type Handlers ---

// AdminListExercises renders the admin page listing all exercise types.
func (h *Handler) AdminListExercises(c echo.Context) error {
	claims := GetClaims(c)
	types, err := h.adminCtrl.ListTypes()
	if err != nil {
		return err
	}

	return render(c, views.AdminExerciseList(types, claims.Name, true, claims.IsAdmin))
}

// AdminNewExerciseForm renders the form for creating a new exercise type.
func (h *Handler) AdminNewExerciseForm(c echo.Context) error {
	claims := GetClaims(c)
	return render(c, views.AdminExerciseForm(views.AdminExerciseFormData{IsEdit: false}, claims.Name, true, claims.IsAdmin))
}

// AdminCreateExercise handles creating a new exercise type.
func (h *Handler) AdminCreateExercise(c echo.Context) error {
	name := c.FormValue("name")
	if name == "" {
		return render(c, views.AdminExerciseFormError("Exercise name is required"))
	}

	_, err := h.adminCtrl.CreateType(name)
	if err != nil {
		return render(c, views.AdminExerciseFormError(err.Error()))
	}

	return c.Redirect(http.StatusSeeOther, "/admin/exercises")
}

// AdminEditExerciseForm renders the form for editing an existing exercise type.
func (h *Handler) AdminEditExerciseForm(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid exercise ID")
	}

	exerciseType, err := h.adminCtrl.GetType(id)
	if err != nil {
		if errors.Is(err, controllers.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Exercise not found")
		}
		return err
	}

	claims := GetClaims(c)
	return render(c, views.AdminExerciseForm(views.AdminExerciseFormData{ExerciseType: exerciseType, IsEdit: true}, claims.Name, true, claims.IsAdmin))
}

// AdminUpdateExercise handles updating an existing exercise type.
func (h *Handler) AdminUpdateExercise(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid exercise ID")
	}

	name := c.FormValue("name")
	if name == "" {
		return render(c, views.AdminExerciseFormError("Exercise name is required"))
	}

	_, err = h.adminCtrl.UpdateType(id, name)
	if err != nil {
		if errors.Is(err, controllers.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Exercise not found")
		}
		return render(c, views.AdminExerciseFormError(err.Error()))
	}

	return c.Redirect(http.StatusSeeOther, "/admin/exercises")
}