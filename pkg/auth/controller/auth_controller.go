package controller

import "github.com/labstack/echo/v4"

type AuthController interface {
	DevLogin(c echo.Context) error
	WhoAmI(c echo.Context) error
}
