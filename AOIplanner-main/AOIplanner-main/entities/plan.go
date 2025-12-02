package entities

import "time"

type Plan struct {
	PlanID     uint      `gorm:"primaryKey" json:"plan_id"`
	FieldID    uint      `json:"field_id" gorm:"index"`
	Version    int       `json:"version"`
	SummaryMD  string    `json:"summary_md"`
	StagesJSON string    `json:"stages_json"`
	CreatedAt  time.Time
}

type ReplanLog struct {
	ID        uint      `gorm:"primaryKey"`
	FieldID   uint
	PlanID    uint
	Reason    string
	DeltaMD   string
	// NEW: persist UI-selected problems
	Problems  []string  `gorm:"serializer:json" json:"problems,omitempty"`
	CreatedAt time.Time

	// NEW (not persisted): articles suggested by service for response payload
	SuggestedArticles []ArticleRef `gorm:"-" json:"suggested_articles,omitempty"`
}

type ArticleRef struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}