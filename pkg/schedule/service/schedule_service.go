package service

import "aoi/entities"

type ScheduleService interface {
List(fieldID uint, from, to string) ([]entities.ScheduleTask, error)
Patch(taskID uint, status string, qty *float64) error
}