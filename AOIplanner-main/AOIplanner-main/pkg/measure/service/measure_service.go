package service

import (
"aoi/entities"
)

type MeasureService interface {
Create(m *entities.Measurement) (*entities.Measurement, error)
Recent(fieldID uint, days int) ([]entities.Measurement, error)
}