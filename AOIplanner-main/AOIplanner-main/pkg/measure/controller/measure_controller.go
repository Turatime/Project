package controller

import "github.com/labstack/echo/v4"

type MeasureController interface {
	Create(c echo.Context) error
	List(c echo.Context) error
}