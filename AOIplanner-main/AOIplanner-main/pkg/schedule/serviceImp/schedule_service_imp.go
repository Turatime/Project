package serviceImp

import (
"aoi/pkg/schedule/service"
repo "aoi/pkg/schedule/repository"
"aoi/entities"
)

type schedSvc struct{ r repo.ScheduleRepository }

func NewScheduleService(r repo.ScheduleRepository) service.ScheduleService { return &schedSvc{r} }

func (s *schedSvc) List(fieldID uint, from, to string) ([]entities.ScheduleTask, error) {
return s.r.List(fieldID, from, to)
}

func (s *schedSvc) Patch(taskID uint, status string, qty *float64) error {
return s.r.PatchStatus(taskID, status, qty)
}