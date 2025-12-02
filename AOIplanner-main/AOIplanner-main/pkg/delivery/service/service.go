package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"aoi/pkg/delivery"
)

type Service interface {
	Create(in *delivery.Delivery) error
	ListByField(fieldID uint, from *time.Time, to *time.Time) ([]delivery.Delivery, error)
	UpdatePartial(id uint, patch DeliveryPatch) (*delivery.Delivery, error)
}

type DeliveryPatch struct {
	Status          *string   `json:"status"`
	TicketNo        *string   `json:"ticket_no"`
	ActualWeightTon *float64  `json:"actual_weight_ton"`
	PricePerTon     *float64  `json:"price_per_ton"`
	NetAmount       *float64  `json:"net_amount"`
	TimeWindowFrom  *string   `json:"time_window_from"`
	TimeWindowTo    *string   `json:"time_window_to"`
	Notes           *string   `json:"notes"`
	Date            *string   `json:"date"`
	MillName        *string   `json:"mill_name"`
	MillQuotaTon    *float64  `json:"mill_quota_ton"`
	FieldID         *uint     `json:"field_id"`
}

type service struct{ db *gorm.DB }

func New(db *gorm.DB) Service { return &service{db: db} }

func (s *service) Create(in *delivery.Delivery) error {
	if in == nil {
		return errors.New("nil delivery")
	}
	// default status
	if in.Status == "" {
		in.Status = "planned"
	}
	return s.db.Create(in).Error
}

func (s *service) ListByField(fieldID uint, from *time.Time, to *time.Time) ([]delivery.Delivery, error) {
	q := s.db.Model(&delivery.Delivery{}).Where("field_id = ?", fieldID)
	if from != nil {
		q = q.Where("date >= ?", from.Format("2006-01-02"))
	}
	if to != nil {
		q = q.Where("date <= ?", to.Format("2006-01-02"))
	}
	var out []delivery.Delivery
	if err := q.Order("date asc, id asc").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *service) UpdatePartial(id uint, patch DeliveryPatch) (*delivery.Delivery, error) {
	var d delivery.Delivery
	if err := s.db.First(&d, id).Error; err != nil {
		return nil, err
	}
	// apply patch (เฉพาะฟิลด์ที่ != nil)
	if patch.Status != nil {
		d.Status = *patch.Status
	}
	if patch.TicketNo != nil {
		d.TicketNo = patch.TicketNo
	}
	if patch.ActualWeightTon != nil {
		d.ActualWeightTon = patch.ActualWeightTon
	}
	if patch.PricePerTon != nil {
		d.PricePerTon = patch.PricePerTon
	}
	if patch.NetAmount != nil {
		d.NetAmount = patch.NetAmount
	}
	if patch.TimeWindowFrom != nil {
		d.TimeWindowFrom = patch.TimeWindowFrom
	}
	if patch.TimeWindowTo != nil {
		d.TimeWindowTo = patch.TimeWindowTo
	}
	if patch.Notes != nil {
		d.Notes = *patch.Notes
	}
	if patch.Date != nil {
		d.Date = *patch.Date
	}
	if patch.MillName != nil {
		d.MillName = *patch.MillName
	}
	if patch.MillQuotaTon != nil {
		d.MillQuotaTon = *patch.MillQuotaTon
	}
	if patch.FieldID != nil {
		d.FieldID = *patch.FieldID
	}
	if err := s.db.Save(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}
