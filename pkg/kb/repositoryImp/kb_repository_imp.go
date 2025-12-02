package repositoryImp

import (
	"aoi/entities"
	"aoi/pkg/kb/repository"
	"gorm.io/gorm"
)

type repo struct{ db *gorm.DB }
func New(db *gorm.DB) repository.KBRepository { return &repo{db} }

func (r *repo) CreateDoc(d *entities.KBDocument) error          { return r.db.Create(d).Error }
func (r *repo) BulkInsertChunks(cs []entities.KBChunk) error     { return r.db.Create(&cs).Error }
func (r *repo) ListDocs() ([]entities.KBDocument, error)         { var ds []entities.KBDocument; return ds, r.db.Order("doc_id DESC").Find(&ds).Error }
func (r *repo) AllChunks() ([]entities.KBChunk, error)           { var cs []entities.KBChunk; return cs, r.db.Find(&cs).Error }
func (r *repo) DocsByIDs(ids []uint) (map[uint]entities.KBDocument, error) {
    if len(ids) == 0 { return map[uint]entities.KBDocument{}, nil }
    var ds []entities.KBDocument
    if err := r.db.Where("doc_id IN ?", ids).Find(&ds).Error; err != nil { return nil, err }
    m := make(map[uint]entities.KBDocument, len(ds))
    for i := range ds { m[ds[i].DocID] = ds[i] }
    return m, nil
}
