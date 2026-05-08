package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"stockvacancy/internal/entity"
	"stockvacancy/internal/repository"
)

type arbeitNowResponse struct {
	Data []arbeitNowJob `json:"data"`
}

type arbeitNowJob struct {
	Slug        string          `json:"slug"`
	CompanyName string          `json:"company_name"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Remote      bool            `json:"remote"`
	URL         string          `json:"url"`
	Location    string          `json:"location"`
	JobTypes    []string        `json:"job_types"`
	CreatedAt   json.RawMessage `json:"created_at"`
}

type ArbeitNowRepository struct {
	url    string
	client *http.Client
}

func NewArbeitNowRepository(url string, timeout time.Duration) *ArbeitNowRepository {
	return &ArbeitNowRepository{
		url: url,
		client: &http.Client{Timeout: timeout},
	}
}

func (r *ArbeitNowRepository) FetchJobs(ctx context.Context) ([]entity.Job, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.url, nil)
	if err != nil {
		return nil, fmt.Errorf("build source request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request source api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("source api returned status %d", resp.StatusCode)
	}

	var payload arbeitNowResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode source api response: %w", err)
	}

	scrapedAt := time.Now().UTC()
	jobs := make([]entity.Job, 0, len(payload.Data))
	for _, item := range payload.Data {
		var publishedAt *time.Time
		if len(item.CreatedAt) > 0 {
			if parsed := parseArbeitNowCreatedAt(item.CreatedAt); parsed != nil {
				publishedAt = parsed
			}
		}

		jobs = append(jobs, entity.Job{
			ExternalID:     item.Slug,
			Title:          item.Title,
			CompanyName:    item.CompanyName,
			Location:       item.Location,
			EmploymentType: strings.Join(item.JobTypes, ", "),
			SalaryText:     "",
			Remote:         item.Remote,
			URL:            item.URL,
			Source:         r.SourceName(),
			Description:    item.Description,
			PublishedAt:    publishedAt,
			ScrapedAt:      scrapedAt,
		})
	}

	return jobs, nil
}

func (r *ArbeitNowRepository) SourceName() string {
	return "arbeitnow"
}

func parseArbeitNowCreatedAt(raw json.RawMessage) *time.Time {
	var unixSeconds int64
	if err := json.Unmarshal(raw, &unixSeconds); err == nil {
		parsed := time.Unix(unixSeconds, 0).UTC()
		return &parsed
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil && text != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, text); parseErr == nil {
			utc := parsed.UTC()
			return &utc
		}
	}

	return nil
}

var _ repository.SourceRepository = (*ArbeitNowRepository)(nil)
