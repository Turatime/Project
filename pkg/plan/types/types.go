package types

type StagePlan struct {
	Stage      string  `json:"stage"`
	StartDate  string  `json:"start_date"`
	EndDate    string  `json:"end_date"`
	WaterMMDay float64 `json:"water_mm_day"`
	Notes      string  `json:"notes"`
	Ops        []PlanOp `json:"ops"`
}

type PlanOp struct {
	Date  string   `json:"date"`
	Type  string   `json:"type"`   // irrigation|fertilizer|pest|observe
	Title string   `json:"title"`
	Qty   *float64 `json:"qty,omitempty"`
	Unit  string   `json:"unit,omitempty"`
	Notes string   `json:"notes,omitempty"`
}