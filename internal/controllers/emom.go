package controllers

import (
	"github.com/labstack/echo/v4"

	"stren/internal/views"
)

type EMOMController struct{}

func NewEMOMController() *EMOMController {
	return &EMOMController{}
}

func (ec *EMOMController) EMOMPage(c echo.Context, isAdmin bool) error {
	data := views.PageData{
		Title:           "EMOM Timer",
		IsAuthenticated: true,
		IsAdmin:         isAdmin,
	}
	return views.EMOMPage(data).Render(c.Request().Context(), c.Response())
}