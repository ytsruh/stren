package routes

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"stren/internal/controllers"
	"stren/internal/utils"
	"stren/internal/views/goals"
)

// goalFormInput is the validated form payload for create/update. The
// route layer is the only place that translates the form values to
// Go types; the controller receives a clean CreateGoalInput or
// UpdateGoalInput and never sees a stringly-typed date.
type goalFormInput struct {
	Title       string `validate:"min=1,max=200"`
	Description string `validate:"max=2000"`
	// The three dates are passed through unchanged after a parse
	// step done by parseGoalDates. We don't tag them with validate
	// here because the parser is the boundary that rejects bad
	// strings; the controller still receives a typed *time.Time.
	StartDateStr  string
	TargetDateStr string
	EndDateStr    string
}

// parseGoalForm reads the form, runs the validator over the
// stringly-typed fields, and parses the three optional date inputs
// into *time.Time (nil for empty / missing). Dates are in the
// "YYYY-MM-DD" format produced by the native HTML5 date picker
// (input type="date"). A malformed date returns a friendly 400.
func parseGoalForm(c echo.Context, v utils.Validator) (controllers.CreateGoalInput, error) {
	input := goalFormInput{
		Title:         strings.TrimSpace(c.FormValue("title")),
		Description:   strings.TrimSpace(c.FormValue("description")),
		StartDateStr:  c.FormValue("start_date"),
		TargetDateStr: c.FormValue("target_date"),
		EndDateStr:    c.FormValue("end_date"),
	}
	if err := v.ValidateStruct(&input); err != nil {
		return controllers.CreateGoalInput{}, echo.NewHTTPError(http.StatusBadRequest, friendlyValidationError(err))
	}
	start, err := parseGoalDate(input.StartDateStr)
	if err != nil {
		return controllers.CreateGoalInput{}, echo.NewHTTPError(http.StatusBadRequest, "Start date: "+err.Error())
	}
	target, err := parseGoalDate(input.TargetDateStr)
	if err != nil {
		return controllers.CreateGoalInput{}, echo.NewHTTPError(http.StatusBadRequest, "Target date: "+err.Error())
	}
	end, err := parseGoalDate(input.EndDateStr)
	if err != nil {
		return controllers.CreateGoalInput{}, echo.NewHTTPError(http.StatusBadRequest, "End date: "+err.Error())
	}
	return controllers.CreateGoalInput{
		Title:       input.Title,
		Description: input.Description,
		StartDate:   start,
		TargetDate:  target,
		EndDate:     end,
	}, nil
}

// parseGoalDate parses a "YYYY-MM-DD" value (the native
// <input type="date"> format) into a *time.Time. Empty input
// returns (nil, nil) so the caller treats the date as absent. A
// non-empty value that fails to parse returns a user-friendly error.
func parseGoalDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("invalid date format")
	}
	return &t, nil
}

// GoalsPage renders the full list page.
func (h *Handler) GoalsPage(c echo.Context) error {
	claims := GetClaims(c)
	goalsList, err := h.goalsCtrl.ListGoals(claims.UserID)
	if err != nil {
		return err
	}
	return render(c, goals.GoalsPage(goalsList, claims.Name, true, claims.IsAdmin))
}

// NewGoalForm renders the empty create form.
func (h *Handler) NewGoalForm(c echo.Context) error {
	claims := GetClaims(c)
	return render(c, goals.NewGoalForm(claims.Name, true, claims.IsAdmin))
}

// CreateGoal handles POST /goals. On success it redirects (full
// page reload) to /goals so the user sees the new card. The htmx
// path uses HX-Trigger to set a 500ms delayed redirect via the
// layout's triggerRedirect handler.
func (h *Handler) CreateGoal(c echo.Context) error {
	in, err := parseGoalForm(c, h.validator)
	if err != nil {
		return render(c, goals.GoalFormError(friendlyError(err)))
	}
	claims := GetClaims(c)
	created, err := h.goalsCtrl.CreateGoal(claims.UserID, in)
	if err != nil {
		return render(c, goals.GoalFormError("Failed to save goal: "+err.Error()))
	}
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", `{"triggerRedirect": "/goals", "goalCreated": null}`)
		return render(c, goals.GoalFormSuccessToast("Goal saved!"))
	}
	_ = created
	return c.Redirect(http.StatusSeeOther, "/goals")
}

// EditGoalForm renders the prefilled edit form.
func (h *Handler) EditGoalForm(c echo.Context) error {
	id := c.Param("id")
	claims := GetClaims(c)
	g, err := h.goalsCtrl.GetGoal(id, claims.UserID)
	if err != nil {
		if err == controllers.ErrGoalNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Goal not found")
		}
		return err
	}
	return render(c, goals.EditGoalForm(g, claims.Name, true, claims.IsAdmin))
}

// UpdateGoal handles PUT /goals/:id. Same redirect-on-success
// pattern as CreateGoal.
func (h *Handler) UpdateGoal(c echo.Context) error {
	id := c.Param("id")
	claims := GetClaims(c)
	if _, err := h.goalsCtrl.GetGoal(id, claims.UserID); err != nil {
		if err == controllers.ErrGoalNotFound {
			return render(c, goals.GoalFormError("Goal not found"))
		}
		return err
	}
	in, err := parseGoalForm(c, h.validator)
	if err != nil {
		return render(c, goals.GoalFormError(friendlyError(err)))
	}
	if _, err := h.goalsCtrl.UpdateGoal(id, claims.UserID, controllers.UpdateGoalInput{
		Title:       in.Title,
		Description: in.Description,
		StartDate:   in.StartDate,
		TargetDate:  in.TargetDate,
		EndDate:     in.EndDate,
	}); err != nil {
		return render(c, goals.GoalFormError("Failed to update goal: "+err.Error()))
	}
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", `{"triggerRedirect": "/goals", "goalUpdated": null}`)
		return render(c, goals.GoalFormSuccessToast("Goal updated!"))
	}
	return c.Redirect(http.StatusSeeOther, "/goals")
}

// MarkGoalComplete handles POST /goals/:id/complete. The button
// lives on the card, so we want the card to "move" from the
// active section to the completed section in place. We re-fetch
// the full goal list after the state change and return both
// section wrappers (each with hx-swap-oob="true") so htmx
// replaces both in the DOM in a single round trip. The
// goalCompleted event triggers the confetti burst on the page.
func (h *Handler) MarkGoalComplete(c echo.Context) error {
	id := c.Param("id")
	claims := GetClaims(c)
	if _, err := h.goalsCtrl.MarkComplete(id, claims.UserID, time.Now()); err != nil {
		if err == controllers.ErrGoalNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Goal not found")
		}
		return err
	}

	// Re-fetch the full list so the OOB response reflects the
	// new state of both sections (active cards, completed cards,
	// and the visibility / count of the completed wrapper).
	goalsList, err := h.goalsCtrl.ListGoals(claims.UserID)
	if err != nil {
		return err
	}

	c.Response().Header().Set("HX-Trigger", `{"goalCompleted": null}`)
	return render(c, goals.GoalsSections(goalsList, claims.IsAdmin))
}

// ReopenGoal handles POST /goals/:id/reopen. Same OOB pattern as
// MarkGoalComplete — the card "moves" from the completed section
// back to the active section in place. No confetti on reopen —
// that's only for the "you did it!" moment, not the "I
// un-checked it" moment.
func (h *Handler) ReopenGoal(c echo.Context) error {
	id := c.Param("id")
	claims := GetClaims(c)
	if _, err := h.goalsCtrl.Reopen(id, claims.UserID); err != nil {
		if err == controllers.ErrGoalNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Goal not found")
		}
		return err
	}

	goalsList, err := h.goalsCtrl.ListGoals(claims.UserID)
	if err != nil {
		return err
	}

	c.Response().Header().Set("HX-Trigger", `{"goalReopened": null}`)
	return render(c, goals.GoalsSections(goalsList, claims.IsAdmin))
}

// DeleteGoal handles DELETE /goals/:id. The delete button is only
// rendered on the edit form, where the card id doesn't exist as a
// DOM target; the route always responds with HX-Redirect so the
// browser navigates back to /goals after the delete commits.
func (h *Handler) DeleteGoal(c echo.Context) error {
	id := c.Param("id")
	claims := GetClaims(c)
	if err := h.goalsCtrl.DeleteGoal(id, claims.UserID); err != nil {
		if err == controllers.ErrGoalNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Goal not found")
		}
		return err
	}
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/goals")
		c.Response().Header().Set("HX-Trigger", `{"goalDeleted": null}`)
		return c.NoContent(http.StatusOK)
	}
	return c.Redirect(http.StatusSeeOther, "/goals")
}
