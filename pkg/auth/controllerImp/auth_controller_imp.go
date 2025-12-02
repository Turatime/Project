package controllerImp

import (
	"net/http"
	"github.com/labstack/echo/v4"
	"aoi/pkg/auth/controller"
)

type authCtrl struct{}

func NewAuthController() controller.AuthController { return &authCtrl{} }

func (h *authCtrl) DevLogin(c echo.Context) error {
	uid := c.QueryParam("uid")
	if uid == "" { uid = "U_DEV_DEFAULT" }
	c.SetCookie(&http.Cookie{Name: "LINE_UID", Value: uid, Path: "/"})
	return c.JSON(http.StatusOK, map[string]string{"uid": uid})
}

func (h *authCtrl) WhoAmI(c echo.Context) error {
	v := c.Get("uid")
	uid, _ := v.(string)
	return c.JSON(http.StatusOK, map[string]string{"uid": uid})
}
