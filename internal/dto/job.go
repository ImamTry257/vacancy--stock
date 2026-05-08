package dto

type SyncJobsRequest struct{}

type JobListQuery struct {
	Page            int
	Limit           int
	Search          string
	Location        string
	EmploymentType  string
	Remote          *bool
	IsInternational *bool
	SortBy          string // published_at | created_at | title | company_name
	SortDir         string // asc | desc
}

type JobResponse struct {
	ID              int64   `json:"id"`
	ExternalID      string  `json:"external_id"`
	Title           string  `json:"title"`
	CompanyName     string  `json:"company_name"`
	Location        string  `json:"location"`
	EmploymentType  string  `json:"employment_type"`
	SalaryText      string  `json:"salary_text"`
	Remote          bool    `json:"remote"`
	IsInternational bool    `json:"is_international"`
	URL             string  `json:"url"`
	Source          string  `json:"source"`
	Description     string  `json:"description"`
	PublishedAt     *string `json:"published_at"`
	ScrapedAt       string  `json:"scraped_at"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type JobsListResponse struct {
	Data       []JobResponse `json:"data"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	Total      int           `json:"total"`
	TotalPages int           `json:"total_pages"`
}

type SyncJobsResponse struct {
	Source        string `json:"source"`
	TotalFetched  int    `json:"total_fetched"`
	TotalInserted int    `json:"total_inserted"`
	TotalUpdated  int    `json:"total_updated"`
}
