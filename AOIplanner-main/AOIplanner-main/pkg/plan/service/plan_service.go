package service

import (
	"aoi/entities"
)

type PlanService interface {
	GenerateFirstPlan(field *entities.Field) (*entities.Plan, []entities.ScheduleTask, error)
	Replan(field *entities.Field) (*entities.Plan, []entities.ScheduleTask, *entities.ReplanLog, error)
}