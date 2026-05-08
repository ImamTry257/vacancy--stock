package repository

import (
	"context"

	"stockvacancy/internal/entity"
)

type SyncLogRepository interface {
	Create(ctx context.Context, syncLog *entity.SyncLog) error
	MarkSuccess(ctx context.Context, syncLogID int64, totalFetched int, totalInserted int, totalUpdated int) error
	MarkFailed(ctx context.Context, syncLogID int64, errMessage string) error
}
