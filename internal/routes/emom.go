package routes

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"stren/internal/views"
)

func (h *Handler) EMOMPage(c echo.Context) error {
	claims := GetClaims(c)
	isAdmin := claims != nil && claims.IsAdmin
	return h.emomCtrl.EMOMPage(c, isAdmin)
}

func (h *Handler) EMOMValidationError(c echo.Context) error {
	return render(c, views.Toast("error", "Invalid Rounds", "Rounds must be between 1 and 15"))
}

func (h *Handler) EMOMRoundToast(c echo.Context) error {
	round := c.FormValue("round")
	r, err := strconv.Atoi(round)
	if err != nil || r < 1 {
		r = 1
	}
	return render(c, views.Toast("info", "Round "+round+" Complete", ""))
}