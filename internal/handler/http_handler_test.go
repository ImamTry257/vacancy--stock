package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stockvacancy/internal/dto"
)

type stubJobUsecase struct {
	syncJobsFn func(ctx context.Context) (*dto.SyncJobsResponse, error)
	listJobsFn func(ctx context.Context, query dto.JobListQuery) (*dto.JobsListResponse, error)
	getJobFn   func(ctx context.Context, id int64) (*dto.JobResponse, error)
}

func (s stubJobUsecase) SyncJobs(ctx context.Context) (*dto.SyncJobsResponse, error) {
	return s.syncJobsFn(ctx)
}

func (s stubJobUsecase) ListJobs(ctx context.Context, query dto.JobListQuery) (*dto.JobsListResponse, error) {
	return s.listJobsFn(ctx, query)
}

func (s stubJobUsecase) GetJobByID(ctx context.Context, id int64) (*dto.JobResponse, error) {
	return s.getJobFn(ctx, id)
}

func TestListJobsReturnsData(t *testing.T) {
	h := NewHTTPHandler(stubJobUsecase{
		syncJobsFn: func(ctx context.Context) (*dto.SyncJobsResponse, error) { return nil, nil },
		listJobsFn: func(ctx context.Context, query dto.JobListQuery) (*dto.JobsListResponse, error) {
			return &dto.JobsListResponse{
				Data: []dto.JobResponse{{ID: 1, Title: "Backend Engineer", CreatedAt: time.Now().UTC().Format(time.RFC3339), UpdatedAt: time.Now().UTC().Format(time.RFC3339), ScrapedAt: time.Now().UTC().Format(time.RFC3339)}},
				Page: 1,
				Limit: 10,
				Total: 1,
				TotalPages: 1,
			}, nil
		},
		getJobFn: func(ctx context.Context, id int64) (*dto.JobResponse, error) { return nil, nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	rec := httptest.NewRecorder()

	h.RegisterRoutes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload dto.JobsListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Data))
	}
}

func TestSyncJobsReturnsAcceptedPayload(t *testing.T) {
	h := NewHTTPHandler(stubJobUsecase{
		syncJobsFn: func(ctx context.Context) (*dto.SyncJobsResponse, error) {
			return &dto.SyncJobsResponse{Source: "arbeitnow", TotalFetched: 5, TotalInserted: 4, TotalUpdated: 1}, nil
		},
		listJobsFn: func(ctx context.Context, query dto.JobListQuery) (*dto.JobsListResponse, error) { return nil, nil },
		getJobFn: func(ctx context.Context, id int64) (*dto.JobResponse, error) { return nil, nil },
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/jobs", nil)
	rec := httptest.NewRecorder()

	h.RegisterRoutes().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}
}
