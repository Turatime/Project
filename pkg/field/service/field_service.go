package service

import "aoi/entities"

type FieldService interface {
	CreateField(f *entities.Field) (*entities.Field, error)
	GetFieldByID(id uint, uid string) (*entities.Field, error)
}
