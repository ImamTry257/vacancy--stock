package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"stockvacancy/internal/dto"
)

type JobUsecase interface {
	SyncJobs(ctx context.Context) (*dto.SyncJobsResponse, error)
	ListJobs(ctx context.Context, query dto.JobListQuery) (*dto.JobsListResponse, error)
	GetJobByID(ctx context.Context, id int64) (*dto.JobResponse, error)
}

type HTTPHandler struct {
	jobUsecase JobUsecase
}

func NewHTTPHandler(jobUsecase JobUsecase) *HTTPHandler {
	return &HTTPHandler{jobUsecase: jobUsecase}
}

func (h *HTTPHandler) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("POST /api/v1/sync/jobs", h.handleSyncJobs)
	mux.HandleFunc("GET /api/v1/jobs", h.handleListJobs)
	mux.HandleFunc("GET /api/v1/jobs/", h.handleGetJobByID)
	return mux
}

func (h *HTTPHandler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, dto.HealthResponse{Status: "ok"})
}

func (h *HTTPHandler) handleSyncJobs(w http.ResponseWriter, r *http.Request) {
	response, err := h.jobUsecase.SyncJobs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, response)
}

func (h *HTTPHandler) handleListJobs(w http.ResponseWriter, r *http.Request) {
	query, err := parseJobListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}

	response, err := h.jobUsecase.ListJobs(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *HTTPHandler) handleGetJobByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseJobID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}

	response, err := h.jobUsecase.GetJobByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if response == nil {
		writeJSON(w, http.StatusNotFound, dto.ErrorResponse{Message: "job not found"})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

var validSortBy = map[string]bool{
	"published_at": true,
	"created_at":   true,
	"title":        true,
	"company_name": true,
}

func parseJobListQuery(r *http.Request) (dto.JobListQuery, error) {
	values := r.URL.Query()
	query := dto.JobListQuery{
		Search:         strings.TrimSpace(values.Get("search")),
		Location:       strings.TrimSpace(values.Get("location")),
		EmploymentType: strings.TrimSpace(values.Get("employment_type")),
	}

	if page := values.Get("page"); page != "" {
		parsed, err := strconv.Atoi(page)
		if err != nil {
			return dto.JobListQuery{}, fmt.Errorf("invalid page")
		}
		query.Page = parsed
	}
	if limit := values.Get("limit"); limit != "" {
		parsed, err := strconv.Atoi(limit)
		if err != nil {
			return dto.JobListQuery{}, fmt.Errorf("invalid limit")
		}
		query.Limit = parsed
	}
	if remote := values.Get("remote"); remote != "" {
		parsed, err := strconv.ParseBool(remote)
		if err != nil {
			return dto.JobListQuery{}, fmt.Errorf("invalid remote")
		}
		query.Remote = &parsed
	}
	if isIntl := values.Get("is_international"); isIntl != "" {
		parsed, err := strconv.ParseBool(isIntl)
		if err != nil {
			return dto.JobListQuery{}, fmt.Errorf("invalid is_international")
		}
		query.IsInternational = &parsed
	}
	if sortBy := strings.ToLower(strings.TrimSpace(values.Get("sort_by"))); sortBy != "" {
		if !validSortBy[sortBy] {
			return dto.JobListQuery{}, fmt.Errorf("invalid sort_by: allowed values are published_at, created_at, title, company_name")
		}
		query.SortBy = sortBy
	}
	if sortDir := strings.ToLower(strings.TrimSpace(values.Get("sort_dir"))); sortDir != "" {
		if sortDir != "asc" && sortDir != "desc" {
			return dto.JobListQuery{}, fmt.Errorf("invalid sort_dir: allowed values are asc, desc")
		}
		query.SortDir = sortDir
	}

	return query, nil
}

func parseJobID(path string) (int64, error) {
	prefix := "/api/v1/jobs/"
	if !strings.HasPrefix(path, prefix) {
		return 0, errors.New("invalid job id path")
	}
	idText := strings.TrimPrefix(path, prefix)
	if idText == "" {
		return 0, errors.New("job id is required")
	}
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil {
		return 0, errors.New("invalid job id")
	}
	return id, nil
}

func writeError(w http.ResponseWriter, statusCode int, err error) {
	writeJSON(w, statusCode, dto.ErrorResponse{Message: err.Error()})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
