package routes

import (
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

// ExportWeightZip streams a zip archive of the authenticated user's
// weight entries (and any photos that can be fetched from R2) as the
// response body. Content-Disposition forces a download with a
// date-stamped filename. The zip is built in a background goroutine
// (see controllers.WeightController.ExportWeightZip) so the response
// streams end-to-end.
//
// This is the last web-only weight surface: the weight CRUD pages
// moved to the iOS client (/api/v1/weight), but the bulk export
// download remains a web convenience.
func (h *Handler) ExportWeightZip(c echo.Context) error {
	claims := GetClaims(c)

	unit := "kg"
	if user := h.GetUser(c); user != nil {
		unit = user.WeightUnitDisplay()
	}

	reader, filename, err := h.weightCtrl.ExportWeightZip(c.Request().Context(), claims.UserID, unit)
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
