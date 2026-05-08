package entity

import "time"

type SyncLog struct {
	ID             int64
	Source         string
	Status         string
	TotalFetched   int
	TotalInserted  int
	TotalUpdated   int
	StartedAt      time.Time
	FinishedAt     *time.Time
	ErrorMessage   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
