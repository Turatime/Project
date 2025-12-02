package repository

import "aoi/entities"

type FieldRepository interface {
	Create(f *entities.Field) error
	FindByID(id uint, uid string) (*entities.Field, error)
}