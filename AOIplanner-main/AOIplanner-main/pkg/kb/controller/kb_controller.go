package controller

import "github.com/labstack/echo/v4"

type KBController interface {
	IngestText(c echo.Context) error
	IngestURL(c echo.Context) error
	Search(c echo.Context) error
}
