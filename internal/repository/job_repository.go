package repository

import (
	"context"

	"stockvacancy/internal/entity"
)

type JobFilter struct {
	Limit           int
	Offset          int
	Search          string
	Location        string
	EmploymentType  string
	Remote          *bool
	IsInternational *bool
	SortBy          string // published_at | created_at | title | company_name
	SortDir         string // asc | desc
}

type JobRepository interface {
	UpsertJobs(ctx context.Context, jobs []entity.Job) (inserted int, updated int, err error)
	ListJobs(ctx context.Context, filter JobFilter) ([]entity.Job, int, error)
	GetJobByID(ctx context.Context, id int64) (*entity.Job, error)
}
