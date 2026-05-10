package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"stockvacancy/internal/entity"
	"stockvacancy/internal/repository"
)

var nextDataPattern = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)

type aggregatorNextData struct {
	Props struct {
		PageProps struct {
			Jobs []aggregatorJob `json:"jobs"`
		} `json:"pageProps"`
	} `json:"props"`
}

type aggregatorJob struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Slug             string  `json:"slug"`
	ActivationDate   string  `json:"activationDate"`
	Description      string  `json:"description"`
	Qualifications   string  `json:"qualifications"`
	Tenure           string  `json:"tenure"`
	IsWorkFromHome   bool    `json:"isWorkFromHome"`
	IsHybrid         bool    `json:"isHybrid"`
	CompanyName      string  `json:"companyName"`
	ApplyRedirectURL *string `json:"applyRedirectUrl"`
	Company          struct {
		Name string `json:"name"`
	} `json:"company"`
	GoogleLocation struct {
		AddressComponents struct {
			City    string `json:"city"`
			Region  string `json:"region"`
			Country string `json:"country"`
		} `json:"addressComponents"`
	} `json:"googleLocation"`
}

type AggregatorRepository struct {
	baseURL  string
	queries  []string
	client   *http.Client
	sourceID string
}

func NewAggregatorRepository(baseURL string, queries []string) *AggregatorRepository {
	normalizedQueries := make([]string, 0, len(queries))
	seen := make(map[string]struct{})
	for _, query := range queries {
		trimmed := strings.TrimSpace(query)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalizedQueries = append(normalizedQueries, trimmed)
	}
	if len(normalizedQueries) == 0 {
		normalizedQueries = []string{"software"}
	}

	return &AggregatorRepository{
		baseURL:  strings.TrimSpace(baseURL),
		queries:  normalizedQueries,
		client:   &http.Client{Timeout: 20 * time.Second},
		sourceID: "job-aggregator",
	}
}

func (r *AggregatorRepository) FetchJobs(ctx context.Context) ([]entity.Job, error) {
	jobMap := make(map[string]entity.Job)

	for _, query := range r.queries {
		jobs, err := r.fetchJobsForQuery(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("fetch aggregator query %q: %w", query, err)
		}
		for _, job := range jobs {
			if existing, exists := jobMap[job.ExternalID]; exists {
				jobMap[job.ExternalID] = pickRicherJob(existing, job)
				continue
			}
			jobMap[job.ExternalID] = job
		}
	}

	jobs := make([]entity.Job, 0, len(jobMap))
	for _, job := range jobMap {
		jobs = append(jobs, job)
	}

	sort.Slice(jobs, func(i, j int) bool {
		left := jobs[i]
		right := jobs[j]
		leftTime := left.ScrapedAt
		if left.PublishedAt != nil {
			leftTime = *left.PublishedAt
		}
		rightTime := right.ScrapedAt
		if right.PublishedAt != nil {
			rightTime = *right.PublishedAt
		}
		if leftTime.Equal(rightTime) {
			return left.ExternalID < right.ExternalID
		}
		return leftTime.After(rightTime)
	})

	return jobs, nil
}

func (r *AggregatorRepository) fetchJobsForQuery(ctx context.Context, query string) ([]entity.Job, error) {
	requestURL, err := r.buildQueryURL(query)
	if err != nil {
		return nil, fmt.Errorf("build query url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build aggregator request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request aggregator page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aggregator page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read aggregator page body: %w", err)
	}

	matches := nextDataPattern.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return nil, fmt.Errorf("aggregator next data payload not found")
	}

	var payload aggregatorNextData
	if err := json.Unmarshal([]byte(matches[1]), &payload); err != nil {
		return nil, fmt.Errorf("decode aggregator next data: %w", err)
	}

	scrapedAt := time.Now().UTC()
	jobs := make([]entity.Job, 0, len(payload.Props.PageProps.Jobs))
	for _, item := range payload.Props.PageProps.Jobs {
		jobs = append(jobs, mapAggregatorJob(item, scrapedAt, r.SourceName(), requestURL))
	}

	return jobs, nil
}

func (r *AggregatorRepository) buildQueryURL(query string) (string, error) {
	base := strings.TrimRight(r.baseURL, "/")
	if base == "" {
		return "", fmt.Errorf("SOURCE_API_URL is not configured")
	}

	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) >= 5 && strings.EqualFold(segments[0], "home") && strings.EqualFold(segments[1], "te") {
		segments[2] = url.PathEscape(query)
		parsed.Path = "/" + strings.Join(segments, "/")
		parsed.RawQuery = ""
		return parsed.String(), nil
	}

	parsed.Path = "/home/te/" + url.PathEscape(query) + "/loc/Indonesia"
	parsed.RawQuery = ""
	return parsed.String(), nil
}

func mapAggregatorJob(item aggregatorJob, scrapedAt time.Time, sourceName string, requestURL string) entity.Job {
	var publishedAt *time.Time
	if item.ActivationDate != "" {
		if parsed, parseErr := time.Parse(time.RFC3339Nano, item.ActivationDate); parseErr == nil {
			utc := parsed.UTC()
			publishedAt = &utc
		}
	}

	companyName := item.CompanyName
	if companyName == "" {
		companyName = item.Company.Name
	}

	locationParts := []string{}
	if item.GoogleLocation.AddressComponents.City != "" {
		locationParts = append(locationParts, item.GoogleLocation.AddressComponents.City)
	}
	if item.GoogleLocation.AddressComponents.Region != "" {
		locationParts = append(locationParts, item.GoogleLocation.AddressComponents.Region)
	}
	if item.GoogleLocation.AddressComponents.Country != "" {
		locationParts = append(locationParts, item.GoogleLocation.AddressComponents.Country)
	}

	jobURL := buildAggregatorJobURL(requestURL, item.Slug)
	if item.ApplyRedirectURL != nil && *item.ApplyRedirectURL != "" {
		jobURL = *item.ApplyRedirectURL
	}

	description := strings.TrimSpace(item.Description)
	if strings.TrimSpace(item.Qualifications) != "" {
		if description != "" {
			description += "\n\n"
		}
		description += item.Qualifications
	}

	return entity.Job{
		ExternalID:      fmt.Sprintf("%d-%s", item.ID, item.Slug),
		Title:           item.Name,
		CompanyName:     companyName,
		Location:        strings.Join(locationParts, ", "),
		EmploymentType:  item.Tenure,
		SalaryText:      "",
		Remote:          item.IsWorkFromHome || item.IsHybrid,
		IsInternational: isInternationalCountry(item.GoogleLocation.AddressComponents.Country),
		URL:             jobURL,
		Source:          sourceName,
		Description:     description,
		PublishedAt:     publishedAt,
		ScrapedAt:       scrapedAt,
	}
}

func pickRicherJob(existing entity.Job, candidate entity.Job) entity.Job {
	existingScore := richnessScore(existing)
	candidateScore := richnessScore(candidate)
	if candidateScore > existingScore {
		return candidate
	}
	if candidateScore == existingScore {
		existingTime := existing.ScrapedAt
		if existing.PublishedAt != nil {
			existingTime = *existing.PublishedAt
		}
		candidateTime := candidate.ScrapedAt
		if candidate.PublishedAt != nil {
			candidateTime = *candidate.PublishedAt
		}
		if candidateTime.After(existingTime) {
			return candidate
		}
	}
	return existing
}

func richnessScore(job entity.Job) int {
	score := 0
	if strings.TrimSpace(job.Description) != "" {
		score += len(strings.TrimSpace(job.Description))
	}
	if strings.TrimSpace(job.Location) != "" {
		score += 10
	}
	if strings.TrimSpace(job.EmploymentType) != "" {
		score += 10
	}
	if job.Remote {
		score += 5
	}
	return score
}

func (r *AggregatorRepository) SourceName() string {
	return r.sourceID
}

func buildAggregatorJobURL(baseURL string, slug string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return strings.TrimRight(baseURL, "/") + "/job/" + slug
	}
	parsed.Path = "/job/" + slug
	parsed.RawQuery = ""
	return parsed.String()
}

// isInternationalCountry returns true if the country is NOT Indonesia.
// Empty country defaults to false (assume domestic).
func isInternationalCountry(country string) bool {
	if country == "" {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(country))
	domestic := []string{"indonesia", "id", "idn"}
	for _, d := range domestic {
		if normalized == d {
			return false
		}
	}
	return true
}

var _ repository.SourceRepository = (*AggregatorRepository)(nil)
