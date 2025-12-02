package serviceImp

import (
	"aoi/entities"
	repo "aoi/pkg/field/repository"
	"aoi/pkg/field/service"
)

type fieldSvc struct{ r repo.FieldRepository }

func NewFieldService(r repo.FieldRepository) service.FieldService { return &fieldSvc{r} }

func (s *fieldSvc) CreateField(f *entities.Field) (*entities.Field, error) {
	if err := s.r.Create(f); err != nil { return nil, err }
	return f, nil
}

func (s *fieldSvc) GetFieldByID(id uint, uid string) (*entities.Field, error) {
	return s.r.FindByID(id, uid)
}
