package controllerImp

import (
	"net/http"
	"strconv"
	"time"
	"github.com/labstack/echo/v4"
	"aoi/entities"
	"aoi/pkg/field/repository"
)

type FieldCtrl struct{ repo repository.FieldRepository }

func New(repo repository.FieldRepository) *FieldCtrl { return &FieldCtrl{repo} }

type createReq struct {
	Variety string `json:"variety"`
	CropType string `json:"crop_type"`
	AreaRai float64 `json:"area_rai"`
	Province string `json:"province"`
	District string `json:"district"`
	SoilTexture string `json:"soil_texture"`
	PumpM3H *float64 `json:"pump_m3h"`
	IrrigationSrc string `json:"irrigation_src"`
	BudgetTier string `json:"budget_tier"`
	FertBase string `json:"fert_base"`
	PlantingDate string `json:"planting_date"`
}

func (h *FieldCtrl) Create(c echo.Context) error {
	uid := c.Get("uid").(string)
	var req createReq
	if err := c.Bind(&req); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error":"bad json"}) }
	pd, _ := time.Parse("2006-01-02", req.PlantingDate)
	f := &entities.Field{UserID: uid, Variety: req.Variety, CropType: req.CropType, AreaRai: req.AreaRai, Province: req.Province, District: req.District, SoilTexture: req.SoilTexture, PumpM3H: req.PumpM3H, IrrigationSrc: req.IrrigationSrc, BudgetTier: req.BudgetTier, FertBase: req.FertBase, PlantingDate: pd}
	if err := h.repo.Create(f); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusCreated, f)
}

func (h *FieldCtrl) Get(c echo.Context) error {
	uid := c.Get("uid").(string)
	id, _ := strconv.Atoi(c.Param("id"))
	f, err := h.repo.FindByID(uint(id), uid)
	if err != nil { return c.JSON(http.StatusNotFound, map[string]string{"error":"not found"}) }
	return c.JSON(http.StatusOK, f)
}