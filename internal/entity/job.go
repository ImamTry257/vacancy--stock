package entity

import "time"

type Job struct {
	ID              int64
	ExternalID      string
	Title           string
	CompanyName     string
	Location        string
	EmploymentType  string
	SalaryText      string
	Remote          bool
	IsInternational bool
	URL             string
	Source          string
	Description     string
	PublishedAt     *time.Time
	ScrapedAt       time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
