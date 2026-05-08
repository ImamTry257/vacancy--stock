package source

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchJobsMapsResponseToEntities(t *testing.T) {
	now := time.Now().UTC()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"data": [{
				"slug": "backend-engineer-1",
				"company_name": "Acme",
				"title": "Backend Engineer",
				"description": "desc",
				"remote": true,
				"url": "https://example.com/jobs/1",
				"location": "Jakarta",
				"job_types": ["Full Time"],
				"created_at": %d
			}]
		}` , now.Unix())
	}))
	defer server.Close()

	repo := NewArbeitNowRepository(server.URL, 5*time.Second)
	jobs, err := repo.FetchJobs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	job := jobs[0]
	if job.ExternalID != "backend-engineer-1" {
		t.Fatalf("unexpected external id: %s", job.ExternalID)
	}
	if job.CompanyName != "Acme" {
		t.Fatalf("unexpected company: %s", job.CompanyName)
	}
	if job.EmploymentType != "Full Time" {
		t.Fatalf("unexpected employment type: %s", job.EmploymentType)
	}
	if !job.Remote {
		t.Fatal("expected remote job")
	}
	if job.Source != "arbeitnow" {
		t.Fatalf("unexpected source: %s", job.Source)
	}
	if job.PublishedAt == nil {
		t.Fatal("expected published at to be set")
	}
}
