package controllers

import (
	"github.com/labstack/echo/v4"

	"stren/internal/views"
)

type EMOMController struct{}

func NewEMOMController() *EMOMController {
	return &EMOMController{}
}

func (ec *EMOMController) EMOMPage(c echo.Context) error {
	data := views.PageData{
		Title:           "EMOM Timer",
		IsAuthenticated: true,
	}
	return views.EMOMPage(data).Render(c.Request().Context(), c.Response())
}