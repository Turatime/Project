package serviceImp

import (
	"errors"
	"time"

	"aoi/pkg/delivery"
	"aoi/pkg/delivery/repository"
	svc "aoi/pkg/delivery/service"
	
)

type service struct{ repo repository.Repo }

func New(r repository.Repo) svc.Service { return &service{repo: r} }

func (s *service) Create(d *delivery.Delivery) error {
	if d.Date == "" {
		return errors.New("date is required")
	}
	if d.Status == "" {
		d.Status = "planned"
	}
	return s.repo.Create(d)
}

func (s *service) UpdatePartial(id uint, p svc.DeliveryPatch) (*delivery.Delivery, error) {
	cur, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if p.Status != nil {
		cur.Status = *p.Status
	}
	if p.TicketNo != nil {
		cur.TicketNo = p.TicketNo
	}
	if p.ActualWeightTon != nil {
		cur.ActualWeightTon = p.ActualWeightTon
	}
	if p.PricePerTon != nil {
		cur.PricePerTon = p.PricePerTon
	}
	// auto-calc net amount
	if cur.ActualWeightTon != nil && cur.PricePerTon != nil {
		v := (*cur.ActualWeightTon) * (*cur.PricePerTon)
		cur.NetAmount = &v
	}
	return cur, s.repo.Update(cur)
}

func (s *service) ListByField(fieldID uint, from, to *time.Time) ([]delivery.Delivery, error) {
	return s.repo.ListByField(fieldID, from, to)
}
