package entities

import "time"

type Field struct {
	FieldID       uint      `gorm:"primaryKey" json:"field_id"`
	UserID        string    `json:"user_id" gorm:"index"`
	Variety       string    `json:"variety"`
	CropType      string    `json:"crop_type"` // new_plant|ratoon
	AreaRai       float64   `json:"area_rai"`
	Province      string    `json:"province"`
	District      string    `json:"district"`
	SoilTexture   string    `json:"soil_texture"` // sand|loam|clay
	PumpM3H       *float64  `json:"pump_m3h"`
	IrrigationSrc string    `json:"irrigation_src"` // well|surface|none
	BudgetTier    string    `json:"budget_tier"`    // low|med|high
	FertBase      string    `json:"fert_base"`      // organic|chemical|mixed
	PlantingDate  time.Time `json:"planting_date"`

	CreatedAt time.Time
	UpdatedAt time.Time
}