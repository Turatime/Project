package entities

import "time"

type ScheduleTask struct {
	TaskID   uint      `gorm:"primaryKey" json:"task_id"`
	FieldID  uint      `gorm:"index" json:"field_id"`
	PlanID   uint      `gorm:"index" json:"plan_id"`
	Date     time.Time `json:"date"`
	Title    string    `json:"title"`
	Type     string    `json:"type"` // irrigation|fertilizer|pest|observe
	Qty      *float64  `json:"qty"`
	Unit     string    `json:"unit"`
	Notes    string    `json:"notes"`
	Status   string    `json:"status"` // todo|done|skipped
	CreatedAt time.Time
	UpdatedAt time.Time
}