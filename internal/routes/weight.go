package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"stren/internal/models"
	"stren/internal/utils"
	"stren/internal/views"
)

// weightFormInput represents the parsed and validated form data for a weight entry.
type weightFormInput struct {
	Weight float64 `validate:"gte=0,lte=1000"`
	Notes  string  `validate:"max=1000"`
}

// parseWeightForm extracts form values, converts types, and validates the input.
// It returns an HTTP error with a user-friendly message if any validation fails.
func parseWeightForm(c echo.Context, v utils.Validator) (*models.WeightEntry, error) {
	input := weightFormInput{
		Notes: c.FormValue("notes"),
	}

	weight, err := strconv.ParseFloat(c.FormValue("weight"), 64)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Weight must be a valid positive number")
	}
	input.Weight = weight

	if err := v.ValidateStruct(&input); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, friendlyValidationError(err))
	}

	return &models.WeightEntry{
		Weight: input.Weight,
		Notes:  input.Notes,
	}, nil
}

// WeightPage renders the weight entries list page.
func (h *Handler) WeightPage(c echo.Context) error {
	claims := GetClaims(c)
	entries, err := h.weightCtrl.ListWeightEntries(claims.UserID)
	if err != nil {
		return err
	}

	return render(c, views.WeightPage(entries, claims.Name, true, claims.IsAdmin))
}

// NewWeightForm renders the form for creating a new weight entry.
func (h *Handler) NewWeightForm(c echo.Context) error {
	claims := GetClaims(c)
	return render(c, views.WeightForm(claims.Name, true, claims.IsAdmin))
}

// CreateWeight handles the creation of a new weight entry.
func (h *Handler) CreateWeight(c echo.Context) error {
	entry, err := parseWeightForm(c, h.validator)
	if err != nil {
		return render(c, views.WeightFormError(friendlyError(err)))
	}

	claims := GetClaims(c)
	_, err = h.weightCtrl.CreateWeightEntry(claims.UserID, entry.Weight, entry.Notes)
	if err != nil {
		return render(c, views.WeightFormError("Failed to save weight entry: "+err.Error()))
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", `{"triggerRedirect": "/weight"}`)
		return render(c, views.WeightFormSuccessToast("Weight saved!"))
	}

	return c.Redirect(http.StatusSeeOther, "/weight")
}

// EditWeightForm renders the form for editing a weight entry.
func (h *Handler) EditWeightForm(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	entry, err := h.weightCtrl.GetWeightEntry(id, claims.UserID)
	if err != nil {
		return err
	}
	if entry == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Weight entry not found")
	}

	return render(c, views.EditWeightForm(entry, claims.Name, true, claims.IsAdmin))
}

// UpdateWeight handles updating an existing weight entry.
func (h *Handler) UpdateWeight(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)

	existing, err := h.weightCtrl.GetWeightEntry(id, claims.UserID)
	if err != nil {
		return err
	}
	if existing == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Weight entry not found")
	}

	entry, err := parseWeightForm(c, h.validator)
	if err != nil {
		return render(c, views.WeightFormError(friendlyError(err)))
	}

	createdAt := existing.CreatedAt
	if dateStr := c.FormValue("created_at"); dateStr != "" {
		createdAt, err = time.Parse("2006-01-02T15:04", dateStr)
		if err != nil {
			return render(c, views.WeightFormError("Invalid date format"))
		}
	}

	_, err = h.weightCtrl.UpdateWeightEntry(id, claims.UserID, entry.Weight, entry.Notes, createdAt)
	if err != nil {
		return render(c, views.WeightFormError("Failed to update weight entry: "+err.Error()))
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return render(c, views.WeightFormSuccessToast("Weight entry updated!"))
	}

	return c.Redirect(http.StatusSeeOther, "/weight")
}

// DeleteWeight handles deleting a weight entry.
func (h *Handler) DeleteWeight(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	if err := h.weightCtrl.DeleteWeightEntry(id, claims.UserID); err != nil {
		return err
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", `{"triggerRedirect": "/weight"}`)
		return render(c, views.WeightDeleteSuccess())
	}

	return c.Redirect(http.StatusSeeOther, "/weight")
}