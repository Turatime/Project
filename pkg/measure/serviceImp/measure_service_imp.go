package serviceImp

import (
"aoi/entities"
repo "aoi/pkg/measure/repository"
"aoi/pkg/measure/service"
)

type measureSvc struct{ r repo.MeasureRepository }

func NewMeasureService(r repo.MeasureRepository) service.MeasureService { return &measureSvc{r} }

func (s *measureSvc) Create(m *entities.Measurement) (*entities.Measurement, error) {
if err := s.r.Create(m); err != nil { return nil, err }
return m, nil
}

func (s *measureSvc) Recent(fieldID uint, days int) ([]entities.Measurement, error) {
return s.r.Recent(fieldID, days)
}