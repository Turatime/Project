package controller

import "github.com/labstack/echo/v4"

type ScheduleController interface {
	List(c echo.Context) error
	Patch(c echo.Context) error
}