package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"stren/internal/views/admin"
)

// AdminListFeedback renders the admin feedback inbox with an
// optional open/closed filter.
func (h *Handler) AdminListFeedback(c echo.Context) error {
	filter := c.QueryParam("filter")

	claims := GetClaims(c)
	feedback, err := h.feedbackCtrl.AdminList(filter)
	if err != nil {
		return err
	}

	return render(c, admin.AdminFeedbackList(feedback, filter, claims.Name, true, claims.IsAdmin))
}

func (h *Handler) AdminFeedbackDetail(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	feedback, err := h.feedbackCtrl.AdminDetail(id)
	if err != nil {
		return err
	}

	return render(c, admin.AdminFeedbackDetail(feedback, claims.Name, true, claims.IsAdmin))
}

func (h *Handler) AdminCloseFeedback(c echo.Context) error {
	id := c.Param("id")

	if err := h.feedbackCtrl.Close(id); err != nil {
		return err
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return render(c, admin.AdminFeedbackClosedSuccess())
	}

	return c.Redirect(http.StatusSeeOther, "/admin/feedback")
}