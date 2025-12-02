package middleware

import (
	"net/http"
	"github.com/labstack/echo/v4"
)

func DevLogin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			uid := ""
			if ck, err := c.Cookie("LINE_UID"); err == nil { uid = ck.Value }
			if uid == "" {
				if q := c.QueryParam("uid"); q != "" {
					c.SetCookie(&http.Cookie{Name:"LINE_UID", Value:q, Path:"/"}); uid = q
				} else {
					uid = "U_DEV_DEFAULT"
					c.SetCookie(&http.Cookie{Name:"LINE_UID", Value:uid, Path:"/"})
				}
			}
			c.Set("uid", uid)
			return next(c)
		}
	}
}