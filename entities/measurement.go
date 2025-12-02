package entities

import "time"

type Measurement struct {
	MeasureID    uint      `gorm:"primaryKey" json:"measure_id"`
	FieldID      uint      `gorm:"index" json:"field_id"`
	Date         time.Time `json:"date"`
	CaneHeightCM *float64  `json:"cane_height_cm"`
	SoilMoistPct *float64  `json:"soil_moist_pct"`
	MoistState   string    `json:"moist_state"` // dry|ok|wet
	RainfallMM   *float64  `json:"rainfall_mm"`
	PestScale    *int      `json:"pest_scale"`
	Note         string    `json:"note"`
	PhotoURL     string    `json:"photo_url"`
	CreatedAt    time.Time
}