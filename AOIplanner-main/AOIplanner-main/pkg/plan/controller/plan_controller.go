package controller

import ("github.com/labstack/echo/v4")


type PlanController interface {
	Generate(c echo.Context) error
	Replan(c echo.Context) error
	List(c echo.Context) error
}