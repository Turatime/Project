package controllerImp

import (
	"net/http"
	"strconv"
	"time"
	"github.com/labstack/echo/v4"
	"aoi/entities"
	repo "aoi/pkg/measure/repository"
)

type MeasureCtrl struct{ repo repo.MeasureRepository }

func New(repo repo.MeasureRepository) *MeasureCtrl { return &MeasureCtrl{repo} }

type measReq struct {
	Date string `json:"date"`
	CaneHeightCM *float64 `json:"cane_height_cm"`
	SoilMoistPct *float64 `json:"soil_moist_pct"`
	MoistState string `json:"moist_state"`
	RainfallMM *float64 `json:"rainfall_mm"`
	PestScale *int `json:"pest_scale"`
	Note string `json:"note"`
	PhotoURL string `json:"photo_url"`
}

func (h *MeasureCtrl) Create(c echo.Context) error {
	fid, _ := strconv.Atoi(c.Param("id"))
	var req measReq
	if err := c.Bind(&req); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error":"bad json"}) }
	d := time.Now()
	if req.Date != "" { dd, err := time.Parse("2006-01-02", req.Date); if err==nil { d = dd } }
	m := &entities.Measurement{ FieldID: uint(fid), Date: d, CaneHeightCM: req.CaneHeightCM, SoilMoistPct: req.SoilMoistPct, MoistState: req.MoistState, RainfallMM: req.RainfallMM, PestScale: req.PestScale, Note: req.Note, PhotoURL: req.PhotoURL }
	if err := h.repo.Create(m); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusCreated, m)
}

func (h *MeasureCtrl) List(c echo.Context) error {
	fid, _ := strconv.Atoi(c.Param("id"))
	out, err := h.repo.Recent(uint(fid), 60)
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, out)
}