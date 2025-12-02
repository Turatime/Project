package repository

import (
	"time"

	"aoi/pkg/delivery"
)

type Repo interface {
	Create(d *delivery.Delivery) error
	Update(d *delivery.Delivery) error
	FindByID(id uint) (*delivery.Delivery, error)
	ListByField(fieldID uint, from, to *time.Time) ([]delivery.Delivery, error)
}
