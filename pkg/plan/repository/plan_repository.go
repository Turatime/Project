package repository

import "aoi/entities"

type PlanRepository interface {
	Create(p *entities.Plan) error
	LatestByField(fieldID uint) (*entities.Plan, error)
	ListByField(fieldID uint) ([]entities.Plan, error)
}