package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"stockvacancy/internal/dto"
	"stockvacancy/internal/entity"
	"stockvacancy/internal/repository"
)

type stubJobRepository struct {
	upsertJobsFn func(ctx context.Context, jobs []entity.Job) (int, int, error)
	listJobsFn   func(ctx context.Context, filter repository.JobFilter) ([]entity.Job, int, error)
	getJobByIDFn func(ctx context.Context, id int64) (*entity.Job, error)
}

func (s stubJobRepository) UpsertJobs(ctx context.Context, jobs []entity.Job) (int, int, error) {
	return s.upsertJobsFn(ctx, jobs)
}

func (s stubJobRepository) ListJobs(ctx context.Context, filter repository.JobFilter) ([]entity.Job, int, error) {
	return s.listJobsFn(ctx, filter)
}

func (s stubJobRepository) GetJobByID(ctx context.Context, id int64) (*entity.Job, error) {
	return s.getJobByIDFn(ctx, id)
}

type stubSyncLogRepository struct {
	createFn      func(ctx context.Context, syncLog *entity.SyncLog) error
	markSuccessFn func(ctx context.Context, syncLogID int64, totalFetched int, totalInserted int, totalUpdated int) error
	markFailedFn  func(ctx context.Context, syncLogID int64, errMessage string) error
}

func (s stubSyncLogRepository) Create(ctx context.Context, syncLog *entity.SyncLog) error {
	return s.createFn(ctx, syncLog)
}

func (s stubSyncLogRepository) MarkSuccess(ctx context.Context, syncLogID int64, totalFetched int, totalInserted int, totalUpdated int) error {
	return s.markSuccessFn(ctx, syncLogID, totalFetched, totalInserted, totalUpdated)
}

func (s stubSyncLogRepository) MarkFailed(ctx context.Context, syncLogID int64, errMessage string) error {
	return s.markFailedFn(ctx, syncLogID, errMessage)
}

type stubSourceRepository struct {
	fetchJobsFn func(ctx context.Context) ([]entity.Job, error)
	sourceName  string
}

func (s stubSourceRepository) FetchJobs(ctx context.Context) ([]entity.Job, error) {
	return s.fetchJobsFn(ctx)
}

func (s stubSourceRepository) SourceName() string {
	return s.sourceName
}

func TestSyncJobsReturnsSummaryAndMarksSuccess(t *testing.T) {
	created := false
	markedSuccess := false
	uc := NewJobUsecase(
		stubJobRepository{
			upsertJobsFn: func(ctx context.Context, jobs []entity.Job) (int, int, error) {
				if len(jobs) != 2 {
					t.Fatalf("expected 2 jobs, got %d", len(jobs))
				}
				return 1, 1, nil
			},
			listJobsFn: func(ctx context.Context, filter repository.JobFilter) ([]entity.Job, int, error) { return nil, 0, nil },
			getJobByIDFn: func(ctx context.Context, id int64) (*entity.Job, error) { return nil, nil },
		},
		stubSyncLogRepository{
			createFn: func(ctx context.Context, syncLog *entity.SyncLog) error {
				created = true
				syncLog.ID = 10
				return nil
			},
			markSuccessFn: func(ctx context.Context, syncLogID int64, totalFetched int, totalInserted int, totalUpdated int) error {
				markedSuccess = true
				if syncLogID != 10 {
					t.Fatalf("expected sync log id 10, got %d", syncLogID)
				}
				if totalFetched != 2 || totalInserted != 1 || totalUpdated != 1 {
					t.Fatalf("unexpected totals: %d %d %d", totalFetched, totalInserted, totalUpdated)
				}
				return nil
			},
			markFailedFn: func(ctx context.Context, syncLogID int64, errMessage string) error { return nil },
		},
		stubSourceRepository{
			sourceName: "arbeitnow",
			fetchJobsFn: func(ctx context.Context) ([]entity.Job, error) {
				return []entity.Job{{ExternalID: "1"}, {ExternalID: "2"}}, nil
			},
		},
	)

	resp, err := uc.SyncJobs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("expected sync log create")
	}
	if !markedSuccess {
		t.Fatal("expected sync log success")
	}
	if resp.TotalFetched != 2 || resp.TotalInserted != 1 || resp.TotalUpdated != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestSyncJobsMarksFailureWhenSourceFails(t *testing.T) {
	markedFailed := false
	uc := NewJobUsecase(
		stubJobRepository{
			upsertJobsFn: func(ctx context.Context, jobs []entity.Job) (int, int, error) { return 0, 0, nil },
			listJobsFn: func(ctx context.Context, filter repository.JobFilter) ([]entity.Job, int, error) { return nil, 0, nil },
			getJobByIDFn: func(ctx context.Context, id int64) (*entity.Job, error) { return nil, nil },
		},
		stubSyncLogRepository{
			createFn: func(ctx context.Context, syncLog *entity.SyncLog) error {
				syncLog.ID = 11
				return nil
			},
			markSuccessFn: func(ctx context.Context, syncLogID int64, totalFetched int, totalInserted int, totalUpdated int) error { return nil },
			markFailedFn: func(ctx context.Context, syncLogID int64, errMessage string) error {
				markedFailed = true
				if syncLogID != 11 {
					t.Fatalf("expected sync log id 11, got %d", syncLogID)
				}
				if errMessage == "" {
					t.Fatal("expected error message")
				}
				return nil
			},
		},
		stubSourceRepository{
			sourceName: "arbeitnow",
			fetchJobsFn: func(ctx context.Context) ([]entity.Job, error) {
				return nil, errors.New("source down")
			},
		},
	)

	_, err := uc.SyncJobs(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !markedFailed {
		t.Fatal("expected sync log failure")
	}
}

func TestListJobsAppliesDefaultPagination(t *testing.T) {
	publishedAt := time.Now().UTC()
	recordedLimit := 0
	recordedOffset := 0
	uc := NewJobUsecase(
		stubJobRepository{
			upsertJobsFn: func(ctx context.Context, jobs []entity.Job) (int, int, error) { return 0, 0, nil },
			listJobsFn: func(ctx context.Context, filter repository.JobFilter) ([]entity.Job, int, error) {
				recordedLimit = filter.Limit
				recordedOffset = filter.Offset
				return []entity.Job{{
					ID: 1, ExternalID: "ext-1", Title: "Backend", CompanyName: "Acme", Location: "Jakarta", EmploymentType: "Full Time", Remote: true,
					URL: "https://example.com", Source: "arbeitnow", Description: "desc", PublishedAt: &publishedAt, ScrapedAt: publishedAt, CreatedAt: publishedAt, UpdatedAt: publishedAt,
				}}, 1, nil
			},
			getJobByIDFn: func(ctx context.Context, id int64) (*entity.Job, error) { return nil, nil },
		},
		stubSyncLogRepository{
			createFn: func(ctx context.Context, syncLog *entity.SyncLog) error { return nil },
			markSuccessFn: func(ctx context.Context, syncLogID int64, totalFetched int, totalInserted int, totalUpdated int) error { return nil },
			markFailedFn: func(ctx context.Context, syncLogID int64, errMessage string) error { return nil },
		},
		stubSourceRepository{sourceName: "arbeitnow", fetchJobsFn: func(ctx context.Context) ([]entity.Job, error) { return nil, nil }},
	)

	resp, err := uc.ListJobs(context.Background(), dto.JobListQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recordedLimit != 10 {
		t.Fatalf("expected default limit 10, got %d", recordedLimit)
	}
	if recordedOffset != 0 {
		t.Fatalf("expected offset 0, got %d", recordedOffset)
	}
	if resp.TotalPages != 1 {
		t.Fatalf("expected 1 page, got %d", resp.TotalPages)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Data))
	}
}
