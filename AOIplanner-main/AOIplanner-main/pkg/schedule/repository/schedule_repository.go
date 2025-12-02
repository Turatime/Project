package repository

import "aoi/entities"

type ScheduleRepository interface {
	BulkInsert([]entities.ScheduleTask) error
	List(fieldID uint, from, to string) ([]entities.ScheduleTask, error)
	PatchStatus(taskID uint, status string, qty *float64) error
}