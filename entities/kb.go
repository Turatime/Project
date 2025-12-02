package entities

import "time"

type KBDocument struct {
	DocID     uint      `gorm:"primaryKey" json:"doc_id"`
	Title     string    `json:"title"`
	SourceURL string    `json:"source_url"`
	Tags      string    `json:"tags"`
	CreatedAt time.Time
}

type KBChunk struct {
	ChunkID   uint      `gorm:"primaryKey" json:"chunk_id"`
	DocID     uint      `gorm:"index" json:"doc_id"`
	Ord       int       `json:"ord"`
	Text      string    `json:"text"`
	Embedding []byte    `json:"-"`
	CreatedAt time.Time
}
