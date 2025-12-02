package delivery

import "gorm.io/gorm"

type Delivery struct {
	gorm.Model                      // ID, CreatedAt, UpdatedAt, DeletedAt
	FieldID         uint   `json:"field_id" gorm:"index"`    // แปลงไหน
	Date            string `json:"date"     gorm:"index"`    // YYYY-MM-DD
	MillName        string `json:"mill_name"`
	MillQuotaTon    float64 `json:"mill_quota_ton"`
	TimeWindowFrom  *string `json:"time_window_from"`         // HH:MM
	TimeWindowTo    *string `json:"time_window_to"`           // HH:MM
	Notes           string  `json:"notes"`
	Status          string  `json:"status" gorm:"index"`      // planned|booked|loading|delivered|paid
	TicketNo        *string `json:"ticket_no"`
	ActualWeightTon *float64 `json:"actual_weight_ton"`
	PricePerTon     *float64 `json:"price_per_ton"`
	NetAmount       *float64 `json:"net_amount"`
}
