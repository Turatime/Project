package repositoryImp

import (
	"time"

	"gorm.io/gorm"

	"aoi/pkg/delivery"
	"aoi/pkg/delivery/repository"
)

type sqliteRepo struct{ db *gorm.DB }

func New(db *gorm.DB) repository.Repo { return &sqliteRepo{db: db} }

func (r *sqliteRepo) Create(d *delivery.Delivery) error { return r.db.Create(d).Error }

func (r *sqliteRepo) Update(d *delivery.Delivery) error { return r.db.Save(d).Error }

func (r *sqliteRepo) FindByID(id uint) (*delivery.Delivery, error) {
	var out delivery.Delivery
	if err := r.db.First(&out, id).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *sqliteRepo) ListByField(fieldID uint, from, to *time.Time) ([]delivery.Delivery, error) {
	q := r.db.Model(&delivery.Delivery{}).Where("field_id = ?", fieldID)
	if from != nil {
		q = q.Where("date >= ?", from.Format("2006-01-02"))
	}
	if to != nil {
		q = q.Where("date <= ?", to.Format("2006-01-02"))
	}
	var list []delivery.Delivery
	return list, q.Order("date asc, id asc").Find(&list).Error
}
