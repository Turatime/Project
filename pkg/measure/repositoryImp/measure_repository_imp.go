package repositoryImp

import (
	"time"
	"aoi/entities"
	"aoi/pkg/measure/repository"
	"gorm.io/gorm"
)

type measureRepo struct{ db *gorm.DB }

func New(db *gorm.DB) repository.MeasureRepository { return &measureRepo{db} }

func (r *measureRepo) Create(m *entities.Measurement) error { return r.db.Create(m).Error }

func (r *measureRepo) Recent(fieldID uint, days int) ([]entities.Measurement, error) {
	var out []entities.Measurement
	cut := time.Now().AddDate(0,0,-days)
	if err := r.db.Where("field_id = ? AND date >= ?", fieldID, cut).Order("date ASC").Find(&out).Error; err != nil { return nil, err }
	return out, nil
}