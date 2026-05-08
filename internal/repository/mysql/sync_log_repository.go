package mysqlrepo

import (
	"context"
	"database/sql"
	"fmt"

	"stockvacancy/internal/entity"
	"stockvacancy/internal/repository"
)

type SyncLogRepository struct {
	db *sql.DB
}

func NewSyncLogRepository(db *sql.DB) *SyncLogRepository {
	return &SyncLogRepository{db: db}
}

func (r *SyncLogRepository) Create(ctx context.Context, syncLog *entity.SyncLog) error {
	query := `
		INSERT INTO sync_logs (
			source, status, total_fetched, total_inserted, total_updated,
			started_at, finished_at, error_message, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
	`

	result, err := r.db.ExecContext(ctx, query,
		syncLog.Source,
		syncLog.Status,
		syncLog.TotalFetched,
		syncLog.TotalInserted,
		syncLog.TotalUpdated,
		syncLog.StartedAt,
		syncLog.FinishedAt,
		syncLog.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("create sync log: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get sync log last insert id: %w", err)
	}
	syncLog.ID = id
	return nil
}

func (r *SyncLogRepository) MarkSuccess(ctx context.Context, syncLogID int64, totalFetched int, totalInserted int, totalUpdated int) error {
	query := `
		UPDATE sync_logs
		SET status = 'success', total_fetched = ?, total_inserted = ?, total_updated = ?, finished_at = NOW(), updated_at = NOW()
		WHERE id = ?
	`
	if _, err := r.db.ExecContext(ctx, query, totalFetched, totalInserted, totalUpdated, syncLogID); err != nil {
		return fmt.Errorf("mark sync log success: %w", err)
	}
	return nil
}

func (r *SyncLogRepository) MarkFailed(ctx context.Context, syncLogID int64, errMessage string) error {
	query := `
		UPDATE sync_logs
		SET status = 'failed', error_message = ?, finished_at = NOW(), updated_at = NOW()
		WHERE id = ?
	`
	if _, err := r.db.ExecContext(ctx, query, errMessage, syncLogID); err != nil {
		return fmt.Errorf("mark sync log failed: %w", err)
	}
	return nil
}

var _ repository.SyncLogRepository = (*SyncLogRepository)(nil)
