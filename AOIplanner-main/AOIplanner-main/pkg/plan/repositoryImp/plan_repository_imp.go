package repositoryImp

import (
	"aoi/entities"
	"aoi/pkg/plan/repository"
	"gorm.io/gorm"
)

type planRepo struct{ db *gorm.DB }

func New(db *gorm.DB) repository.PlanRepository { return &planRepo{db} }

func (r *planRepo) Create(p *entities.Plan) error { return r.db.Create(p).Error }

func (r *planRepo) LatestByField(fieldID uint) (*entities.Plan, error) {
	var p entities.Plan
	if err := r.db.Where("field_id = ?", fieldID).Order("version DESC").First(&p).Error; err != nil { return nil, err }
	return &p, nil
}

func (r *planRepo) ListByField(fieldID uint) ([]entities.Plan, error) {
	var ps []entities.Plan
	if err := r.db.Where("field_id = ?", fieldID).Order("version ASC").Find(&ps).Error; err != nil { return nil, err }
	return ps, nil
}