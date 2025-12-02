package router

import (
	"github.com/labstack/echo/v4"
	"aoi/pkg/middleware"
)

func New(
	e *echo.Echo,
	fieldCtrl interface{ Create(echo.Context) error; Get(echo.Context) error },
	planGenerate func(echo.Context) error,
	planReplan   func(echo.Context) error,
	planList     func(echo.Context) error,
	measCtrl  interface{ Create(echo.Context) error; List(echo.Context) error },
	schedCtrl interface{ List(echo.Context) error; Patch(echo.Context) error },
	authCtrl  interface{ DevLogin(echo.Context) error; WhoAmI(echo.Context) error },
	kbCtrl    interface{ IngestText(echo.Context) error; IngestURL(echo.Context) error; Search(echo.Context) error },
	healthCtrl interface{ Health(echo.Context) error },

) *echo.Echo {
	e.Use(middleware.DevLogin())
	api := e.Group("")

	api.GET("/whoami", authCtrl.WhoAmI)
	api.GET("/devlogin", authCtrl.DevLogin)
	e.GET("/health", healthCtrl.Health) 
	
	// KB endpoints
	api.POST("/kb/ingest",     kbCtrl.IngestText)
	api.POST("/kb/ingest/url", kbCtrl.IngestURL)
	api.GET("/kb/search",      kbCtrl.Search)

	api.POST("/fields", fieldCtrl.Create)
	api.GET("/fields/:id", fieldCtrl.Get)
	
	g := e.Group("/fields")
	g.POST("/:id/plan", planGenerate)
	g.POST("/:id/replan", planReplan)
	g.GET("/:id/plan", planList)

	api.POST("/fields/:id/measurements", measCtrl.Create)
	api.GET("/fields/:id/measurements", measCtrl.List)

	api.GET("/fields/:id/schedule", schedCtrl.List)
	api.PATCH("/schedule/:task_id", schedCtrl.Patch)
	return e
}