package repositoryImp

import (
	"time"
	"aoi/entities"
	"aoi/pkg/schedule/repository"
	"gorm.io/gorm"
)

type schedRepo struct{ db *gorm.DB }

func New(db *gorm.DB) repository.ScheduleRepository { return &schedRepo{db} }

func (r *schedRepo) BulkInsert(ts []entities.ScheduleTask) error { return r.db.Create(&ts).Error }

func (r *schedRepo) List(fieldID uint, from, to string) ([]entities.ScheduleTask, error) {
	var out []entities.ScheduleTask
	var s, e time.Time
	var err error
	if from != "" { s, err = time.Parse("2006-01-02", from); if err!=nil { s = time.Time{} } }
	if to != "" { e, err = time.Parse("2006-01-02", to); if err!=nil { e = time.Time{} } }
	q := r.db.Where("field_id = ?", fieldID)
	if !s.IsZero() { q = q.Where("date >= ?", s) }
	if !e.IsZero() { q = q.Where("date <= ?", e) }
	if err := q.Order("date ASC").Find(&out).Error; err != nil { return nil, err }
	return out, nil
}

func (r *schedRepo) PatchStatus(taskID uint, status string, qty *float64) error {
	upd := map[string]any{"status": status}
	if qty != nil { upd["qty"] = qty }
	return r.db.Model(&entities.ScheduleTask{}).Where("task_id = ?", taskID).Updates(upd).Error
}