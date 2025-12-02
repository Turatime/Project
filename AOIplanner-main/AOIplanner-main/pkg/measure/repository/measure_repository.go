package repository

import "aoi/entities"

type MeasureRepository interface {
	Create(m *entities.Measurement) error
	Recent(fieldID uint, days int) ([]entities.Measurement, error)
}