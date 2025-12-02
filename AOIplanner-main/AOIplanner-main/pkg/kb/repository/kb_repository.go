package repository

import "aoi/entities"

type KBRepository interface {
	CreateDoc(*entities.KBDocument) error
	BulkInsertChunks([]entities.KBChunk) error
	ListDocs() ([]entities.KBDocument, error)
	AllChunks() ([]entities.KBChunk, error)
	DocsByIDs(ids []uint) (map[uint]entities.KBDocument, error)
}
