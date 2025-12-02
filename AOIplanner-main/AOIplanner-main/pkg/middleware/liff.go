package middleware


import (
"net/http"
"github.com/labstack/echo/v4"
)


// LIFF is an optional middleware. When enabled=true, it tries to read a LINE UID
// from headers/cookies set by your LIFF frontend. If it cannot find one, it
// returns 401. When enabled=false, it simply passes through (use DevLogin instead).
func LIFF(enabled bool) echo.MiddlewareFunc {
return func(next echo.HandlerFunc) echo.HandlerFunc {
return func(c echo.Context) error {
if !enabled {
return next(c) // bypass in development
}
// Example heuristic: look for X-Line-Uid header or LINE_UID cookie
uid := c.Request().Header.Get("X-Line-Uid")
if uid == "" {
if ck, err := c.Cookie("LINE_UID"); err == nil { uid = ck.Value }
}
if uid == "" {
return c.JSON(http.StatusUnauthorized, map[string]string{"error":"LIFF required: missing UID"})
}
c.Set("uid", uid)
return next(c)
}
}
}