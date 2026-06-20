package routes

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"stren/internal/models"
	"stren/internal/utils"
	"stren/internal/views"
	"stren/internal/views/weight"
)

// weightFormInput represents the parsed and validated form data for a weight entry.
type weightFormInput struct {
	Weight float64 `validate:"gte=0,lte=1000"`
	Notes  string  `validate:"max=1000"`
}

// parseWeightForm extracts form values, converts types, and validates the input.
// It returns an HTTP error with a user-friendly message if any validation fails.
// The returned entry contains the typed values plus the photo_key from the
// hidden form field (empty string if not provided).
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
		Weight:   input.Weight,
		Notes:    input.Notes,
		PhotoKey: c.FormValue("photo_key"),
	}, nil
}

// WeightPage renders the weight entries list page.
func (h *Handler) WeightPage(c echo.Context) error {
	claims := GetClaims(c)
	entries, err := h.weightCtrl.ListWeightEntries(claims.UserID)
	if err != nil {
		return err
	}

	// Look up the user's body-weight goal (if any) to drive the progress
	// widget. A failed load is logged and treated as "no goal" so the page
	// still renders.
	var target *float64
	user, err := h.userRepo.GetUserByID(claims.UserID)
	if err != nil {
		c.Logger().Errorf("failed to load user for weight page: %v", err)
	} else if user != nil {
		target = user.TargetWeight
	}

	return render(c, weight.WeightPage(entries, claims.Name, true, claims.IsAdmin, target))
}

// NewWeightForm renders the form for creating a new weight entry.
func (h *Handler) NewWeightForm(c echo.Context) error {
	claims := GetClaims(c)
	return render(c, weight.WeightForm(claims.Name, true, claims.IsAdmin))
}

// CreateWeight handles the creation of a new weight entry.
func (h *Handler) CreateWeight(c echo.Context) error {
	entry, err := parseWeightForm(c, h.validator)
	if err != nil {
		return render(c, weight.WeightFormError(friendlyError(err)))
	}

	claims := GetClaims(c)
	_, err = h.weightCtrl.CreateWeightEntry(claims.UserID, entry.Weight, entry.Notes, entry.PhotoKey)
	if err != nil {
		return render(c, weight.WeightFormError("Failed to save weight entry: "+err.Error()))
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", `{"triggerRedirect": "/weight"}`)
		return render(c, weight.WeightFormSuccessToast("Weight saved!"))
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

	return render(c, weight.EditWeightForm(entry, claims.Name, true, claims.IsAdmin))
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
		return render(c, weight.WeightFormError(friendlyError(err)))
	}

	createdAt := existing.CreatedAt
	if dateStr := c.FormValue("created_at"); dateStr != "" {
		createdAt, err = time.Parse("2006-01-02T15:04", dateStr)
		if err != nil {
			return render(c, weight.WeightFormError("Invalid date format"))
		}
	}

	// Resolve the final photo_key:
	//  - if "remove_photo" is checked, clear it (and best-effort delete from R2)
	//  - else if a new photo_key was uploaded, use it
	//  - else keep the existing one
	photoKey := existing.PhotoKey
	removePhoto := c.FormValue("remove_photo") == "true"
	if removePhoto {
		if existing.HasPhoto() {
			if delErr := utils.DeleteObject(existing.PhotoKey); delErr != nil {
				// log but don't fail the update
				c.Logger().Warnf("failed to delete weight photo %q from R2: %v", existing.PhotoKey, delErr)
			}
		}
		photoKey = ""
	} else if entry.PhotoKey != "" {
		photoKey = entry.PhotoKey
	}

	_, err = h.weightCtrl.UpdateWeightEntry(id, claims.UserID, entry.Weight, entry.Notes, photoKey, createdAt)
	if err != nil {
		return render(c, weight.WeightFormError("Failed to update weight entry: "+err.Error()))
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", `{"triggerRedirect": "/weight"}`)
		return render(c, weight.WeightFormSuccessToast("Weight entry updated!"))
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
		return render(c, weight.WeightDeleteSuccess())
	}

	return c.Redirect(http.StatusSeeOther, "/weight")
}

// CompareWeightModal returns the modal fragment used by the image-comparison
// feature on the weight list page. It is fetched by htmx in response to the
// "Compare" button on the table and is swapped into a container element
// next to the table. On any error the response is a Basecoat error toast
// (still in the same container) so the page can show feedback without
// reloading.
func (h *Handler) CompareWeightModal(c echo.Context) error {
	idA := c.QueryParam("a")
	idB := c.QueryParam("b")

	claims := GetClaims(c)
	entries, err := h.weightCtrl.GetWeightEntriesForCompare(idA, idB, claims.UserID)
	if err != nil {
		return render(c, views.Toast("error", "Cannot compare photos", err.Error()))
	}

	before := entries[0]
	after := entries[1]
	delta := after.Weight - before.Weight
	deltaStr := formatWeightDelta(delta)

	return render(c, weight.CompareModal(
		before.FormattedDateLong(), before.FormattedWeight(), before.PhotoURL(),
		after.FormattedDateLong(), after.FormattedWeight(), after.PhotoURL(),
		deltaStr,
	))
}

// formatWeightDelta produces a short human-readable summary of the
// weight change between two entries. Positive deltas are prefixed with
// "+", negative with "−" (U+2212), and deltas within ±0.05 kg return
// an empty string so the caller can omit the change indicator entirely
// (avoids the awkward "· no change" label).
func formatWeightDelta(delta float64) string {
	switch {
	case delta > 0.05:
		return fmt.Sprintf("+%.1f kg", delta)
	case delta < -0.05:
		return fmt.Sprintf("−%.1f kg", -delta)
	default:
		return ""
	}
}

// ExportWeightZip streams a zip archive of the authenticated user's
// weight entries (and any photos that can be fetched from R2) as the
// response body. Content-Disposition forces a download with a
// date-stamped filename. The zip is built in a background goroutine
// (see controllers.WeightController.ExportWeightZip) so the response
// streams end-to-end.
func (h *Handler) ExportWeightZip(c echo.Context) error {
	claims := GetClaims(c)

	reader, filename, err := h.weightCtrl.ExportWeightZip(c.Request().Context(), claims.UserID)
	if err != nil {
		return err
	}

	c.Response().Header().Set(echo.HeaderContentType, "application/zip")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Response().WriteHeader(http.StatusOK)
	if _, err := io.Copy(c.Response().Writer, reader); err != nil {
		// We've already sent headers, so we can't return a
		// proper error response. Log it and abort the
		// connection so the client doesn't see a truncated
		// zip labelled "OK".
		c.Logger().Errorf("weight export stream failed: %v", err)
	}
	return nil
}
