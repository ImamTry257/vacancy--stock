package repository

import (
	"context"

	"stockvacancy/internal/entity"
)

type SourceRepository interface {
	FetchJobs(ctx context.Context) ([]entity.Job, error)
	SourceName() string
}
