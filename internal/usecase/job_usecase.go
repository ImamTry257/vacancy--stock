package usecase

import (
	"context"
	"fmt"
	"math"
	"time"

	"stockvacancy/internal/dto"
	"stockvacancy/internal/entity"
	"stockvacancy/internal/htmlutil"
	"stockvacancy/internal/repository"
)

type JobUsecase struct {
	jobRepo     repository.JobRepository
	syncLogRepo repository.SyncLogRepository
	sourceRepo  repository.SourceRepository
}

func NewJobUsecase(jobRepo repository.JobRepository, syncLogRepo repository.SyncLogRepository, sourceRepo repository.SourceRepository) *JobUsecase {
	return &JobUsecase{
		jobRepo:     jobRepo,
		syncLogRepo: syncLogRepo,
		sourceRepo:  sourceRepo,
	}
}

func (u *JobUsecase) SyncJobs(ctx context.Context) (*dto.SyncJobsResponse, error) {
	syncLog := &entity.SyncLog{
		Source:    u.sourceRepo.SourceName(),
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}
	if err := u.syncLogRepo.Create(ctx, syncLog); err != nil {
		return nil, fmt.Errorf("create sync log: %w", err)
	}

	jobs, err := u.sourceRepo.FetchJobs(ctx)
	if err != nil {
		_ = u.syncLogRepo.MarkFailed(ctx, syncLog.ID, err.Error())
		return nil, fmt.Errorf("fetch jobs from source: %w", err)
	}

	// normalise HTML descriptions to plain text before persisting
	for i := range jobs {
		jobs[i].Description = htmlutil.StripHTML(jobs[i].Description)
	}

	inserted, updated, err := u.jobRepo.UpsertJobs(ctx, jobs)
	if err != nil {
		_ = u.syncLogRepo.MarkFailed(ctx, syncLog.ID, err.Error())
		return nil, fmt.Errorf("upsert jobs: %w", err)
	}

	if err := u.syncLogRepo.MarkSuccess(ctx, syncLog.ID, len(jobs), inserted, updated); err != nil {
		return nil, fmt.Errorf("mark sync success: %w", err)
	}

	return &dto.SyncJobsResponse{
		Source:        u.sourceRepo.SourceName(),
		TotalFetched:  len(jobs),
		TotalInserted: inserted,
		TotalUpdated:  updated,
	}, nil
}

func (u *JobUsecase) ListJobs(ctx context.Context, query dto.JobListQuery) (*dto.JobsListResponse, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	sortBy := query.SortBy
	if sortBy == "" {
		sortBy = "published_at"
	}
	sortDir := query.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}

	offset := (page - 1) * limit
	jobs, total, err := u.jobRepo.ListJobs(ctx, repository.JobFilter{
		Limit:           limit,
		Offset:          offset,
		Search:          query.Search,
		Location:        query.Location,
		EmploymentType:  query.EmploymentType,
		Remote:          query.Remote,
		IsInternational: query.IsInternational,
		SortBy:          sortBy,
		SortDir:         sortDir,
	})
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}

	items := make([]dto.JobResponse, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, mapJobToResponse(job))
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return &dto.JobsListResponse{
		Data:       items,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (u *JobUsecase) GetJobByID(ctx context.Context, id int64) (*dto.JobResponse, error) {
	job, err := u.jobRepo.GetJobByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get job by id: %w", err)
	}
	if job == nil {
		return nil, nil
	}
	response := mapJobToResponse(*job)
	return &response, nil
}

func mapJobToResponse(job entity.Job) dto.JobResponse {
	var publishedAt *string
	if job.PublishedAt != nil {
		formatted := job.PublishedAt.UTC().Format(time.RFC3339)
		publishedAt = &formatted
	}

	return dto.JobResponse{
		ID:              job.ID,
		ExternalID:      job.ExternalID,
		Title:           job.Title,
		CompanyName:     job.CompanyName,
		Location:        job.Location,
		EmploymentType:  job.EmploymentType,
		SalaryText:      job.SalaryText,
		Remote:          job.Remote,
		IsInternational: job.IsInternational,
		URL:             job.URL,
		Source:          job.Source,
		Description:     job.Description,
		PublishedAt:     publishedAt,
		ScrapedAt:       job.ScrapedAt.UTC().Format(time.RFC3339),
		CreatedAt:       job.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       job.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
