package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/views"
)

func (h *Handler) FeedbackForm(c echo.Context) error {
	claims := GetClaims(c)
	return render(c, views.FeedbackPage(claims.Name, true, claims.IsAdmin))
}

func (h *Handler) SubmitFeedback(c echo.Context) error {
	title := c.FormValue("title")
	message := c.FormValue("message")

	claims := GetClaims(c)
	err := h.feedbackCtrl.Submit(title, message, claims.UserID)
	if err != nil {
		if err == controllers.ErrTitleTooShort {
			return render(c, views.FeedbackFormError("Title must be at least 5 characters"))
		}
		if err == controllers.ErrTitleTooLong {
			return render(c, views.FeedbackFormError("Title must be at most 100 characters"))
		}
		if err == controllers.ErrMessageTooShort {
			return render(c, views.FeedbackFormError("Message must be at least 10 characters"))
		}
		if err == controllers.ErrMessageTooLong {
			return render(c, views.FeedbackFormError("Message must be at most 1000 characters"))
		}
		return render(c, views.FeedbackFormError("Failed to submit feedback: "+err.Error()))
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/")
		c.Response().WriteHeader(http.StatusSeeOther)
		return nil
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

func (h *Handler) AdminListFeedback(c echo.Context) error {
	filter := c.QueryParam("filter")

	claims := GetClaims(c)
	feedback, err := h.feedbackCtrl.AdminList(filter)
	if err != nil {
		return err
	}

	return render(c, views.AdminFeedbackList(feedback, filter, claims.Name, true, claims.IsAdmin))
}

func (h *Handler) AdminFeedbackDetail(c echo.Context) error {
	id := c.Param("id")

	claims := GetClaims(c)
	feedback, err := h.feedbackCtrl.AdminDetail(id)
	if err != nil {
		return err
	}

	return render(c, views.AdminFeedbackDetail(feedback, claims.Name, true, claims.IsAdmin))
}

func (h *Handler) AdminCloseFeedback(c echo.Context) error {
	id := c.Param("id")

	if err := h.feedbackCtrl.Close(id); err != nil {
		return err
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return render(c, views.AdminFeedbackClosedSuccess())
	}

	return c.Redirect(http.StatusSeeOther, "/admin/feedback")
}

func (h *Handler) AdminFeedback(c echo.Context) error {
	return h.AdminListFeedback(c)
}