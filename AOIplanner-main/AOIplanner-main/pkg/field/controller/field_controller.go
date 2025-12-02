package controller

import "github.com/labstack/echo/v4"

type FieldController interface {
	Create(c echo.Context) error
	Get(c echo.Context) error
}