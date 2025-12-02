package repositoryImp

import (
	"aoi/entities"
	"aoi/pkg/field/repository"
	"gorm.io/gorm"
)

type fieldRepo struct{ db *gorm.DB }

func New(db *gorm.DB) repository.FieldRepository { return &fieldRepo{db} }

func (r *fieldRepo) Create(f *entities.Field) error { return r.db.Create(f).Error }

func (r *fieldRepo) FindByID(id uint, uid string) (*entities.Field, error) {
	var f entities.Field
	if err := r.db.Where("field_id = ? AND user_id = ?", id, uid).First(&f).Error; err != nil { return nil, err }
	return &f, nil
}