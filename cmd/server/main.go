package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"aoi/config"
	"aoi/database"
	"aoi/router"

	// Auth
	authCtrlImp "aoi/pkg/auth/controllerImp"

	// Field
	fieldCtrlImp "aoi/pkg/field/controllerImp"
	fieldRepoImp "aoi/pkg/field/repositoryImp"

	// Measure
	measCtrlImp "aoi/pkg/measure/controllerImp"
	measRepoImp "aoi/pkg/measure/repositoryImp"

	// Schedule
	schedCtrlImp "aoi/pkg/schedule/controllerImp"
	schedRepoImp "aoi/pkg/schedule/repositoryImp"

	// Plan
	planCtrlImp "aoi/pkg/plan/controllerImp"
	planRepoImp "aoi/pkg/plan/repositoryImp"
	planSvc "aoi/pkg/plan/serviceImp"

	// Rules/LLM
	"aoi/pkg/ai"
	"aoi/pkg/climate"

	// KB (names/paths must match repo exactly)
	kbCtrlImp    "aoi/pkg/kb/controllerImp"
	kbRepoImp    "aoi/pkg/kb/repositoryImp"
	kbServiceImp "aoi/pkg/kb/serviceImp"
	kbEmbedder   "aoi/pkg/kb/embedder"

	// Health
	healthCtrlImp "aoi/pkg/health/controllerImp"

	delCtrlImp "aoi/pkg/delivery/controllerImp"
    dsvc "aoi/pkg/delivery/service"
    "aoi/pkg/delivery"

)

func main() {
	// 1) Config
	cfg := config.Load()

	// 2) DB (sqlite) + automigrate
	db := database.OpenSQLite(cfg.DBPath)

	// หลัง db := database.OpenSQLite(cfg.DBPath)
if err := db.AutoMigrate(&delivery.Delivery{}); err != nil {
    log.Fatalf("auto-migrate delivery: %v", err)
}
// สร้าง service + controller แล้ว register routes
delSvc := dsvc.New(db)          // สมมติ service.New(db) มีอยู่
delCtrl := delCtrlImp.New(delSvc)


	// 3) Echo
	e := echo.New()
	e.Use(echoMiddleware.Recover())
	delCtrl.Register(e)
	// Static (keep your existing behavior)
	e.Static("/static", "static")
	e.File("/", "static/index.html")
	if _, err := os.Stat("static/app.js"); err != nil {
		log.Printf("WARN: static/app.js not found: %v", err)
	}

	// 4) Climate rules
	rules, err := climate.LoadFromFiles(
		"./StageConfig.csv",
		"./CropTypeAdjustments.csv",
		"./Sugarcane_Irrigation_Config.xlsx",
	)
	if err != nil {
		log.Printf("rules warn: %v", err)
	}

	// 5) LLM (mock fallback)
	var llm ai.Client
	if cfg.LLMEndpoint != "" && cfg.LLMAPIKey != "" {
		llm = ai.NewOpenAI(cfg.LLMEndpoint, cfg.LLMAPIKey, cfg.LLMModel)
	} else {
		llm = ai.NewMock()
	}

	// 6) KB wiring — **ensure non-nil embedder**
	emb := kbEmbedder.New(
		os.Getenv("EMB_ENDPOINT"),
		os.Getenv("EMB_API_KEY"),
		os.Getenv("EMB_MODEL"),
	)
	kbRepo := kbRepoImp.New(db)
	kbSvc := kbServiceImp.New(kbRepo, emb)   // ← do NOT pass nil here
	kbCtrl := kbCtrlImp.New(kbSvc)           // ← KBCtrl has field `s`, not `svc`

	// 7) Repos/Controllers
	fRepo := fieldRepoImp.New(db)
	mRepo := measRepoImp.New(db)
	sRepo := schedRepoImp.New(db)
	pRepo := planRepoImp.New(db)
	fCtrl := fieldCtrlImp.New(fRepo)
	meCtrl := measCtrlImp.New(mRepo)
	scCtrl := schedCtrlImp.New(sRepo)

	// Plan service depends on rules/llm/repos + kb
	pSvc := planSvc.NewPlanService(rules, llm, pRepo, sRepo, mRepo, kbSvc)
	plCtrl := planCtrlImp.NewPlanCtrl(db, pSvc)

	// Auth + Health
	authCtrl := authCtrlImp.NewAuthController()
	hCtrl := healthCtrlImp.NewHealthCtrl(db)


	// 8) Router — match actual signature (includes health)
	r := router.New(
		e,
		fCtrl,
		plCtrl.Generate,  // pass functions
		plCtrl.Replan,
		plCtrl.List,
		meCtrl,
		scCtrl,
		authCtrl,
		kbCtrl,
		hCtrl,
	)

	// 9) Start
	log.Printf("listening on :%s", cfg.Port)
	if err := r.Start(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
