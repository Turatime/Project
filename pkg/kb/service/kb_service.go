package service

import "aoi/entities"

type KBService interface {
	UpsertDocument(title, tags, text, sourceURL string) (*entities.KBDocument, int, error)
	Search(query string, k int) ([]entities.KBChunk, error)
	DocsMeta(ids []uint) (map[uint]entities.KBDocument, error)
}
