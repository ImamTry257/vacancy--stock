package source

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchJobsFromAggregatorMapsIndonesianJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><body><script id="__NEXT_DATA__" type="application/json">{
			"props": {
				"pageProps": {
					"jobs": [{
						"id": 262972,
						"name": "Software Engineer",
						"slug": "software-engineer-5",
						"activationDate": "2026-04-20T01:25:54.767174+00:00",
						"description": "<p>Build backend services</p>",
						"qualifications": "<p>Know Go and MySQL</p>",
						"tenure": "Full time",
						"isWorkFromHome": false,
						"isHybrid": false,
						"companyName": "PT Akhdani Reka Solusi",
						"company": {"name": "PT Akhdani Reka Solusi"},
						"googleLocation": {
							"addressComponents": {
								"city": "Central Jakarta",
								"region": "DKI Jakarta",
								"country": "Indonesia"
							}
						},
						"applyRedirectUrl": null
					}]
				}
			}
		}</script></body></html>`)
	}))
	defer server.Close()

	repo := NewAggregatorRepository(server.URL+"/home/te/software/loc/Indonesia", []string{"software"})
	jobs, err := repo.FetchJobs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	job := jobs[0]
	if job.ExternalID != "262972-software-engineer-5" {
		t.Fatalf("unexpected external id: %s", job.ExternalID)
	}
	if job.CompanyName != "PT Akhdani Reka Solusi" {
		t.Fatalf("unexpected company: %s", job.CompanyName)
	}
	if job.Location != "Central Jakarta, DKI Jakarta, Indonesia" {
		t.Fatalf("unexpected location: %s", job.Location)
	}
	if job.EmploymentType != "Full time" {
		t.Fatalf("unexpected employment type: %s", job.EmploymentType)
	}
	if job.Source != "job-aggregator" {
		t.Fatalf("unexpected source: %s", job.Source)
	}
	if job.URL != server.URL+"/job/software-engineer-5" {
		t.Fatalf("unexpected url: %s", job.URL)
	}
	if job.Remote {
		t.Fatal("expected remote false")
	}
	if job.PublishedAt == nil {
		t.Fatal("expected published at")
	}
}

func TestFetchJobsFromAggregatorDeduplicatesAcrossQueries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		body := `<!DOCTYPE html><html><body><script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"jobs":[]}}}</script></body></html>`
		if r.URL.Path == "/home/te/software/loc/Indonesia" {
			body = `<!DOCTYPE html><html><body><script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"jobs":[{"id":1,"name":"Software Engineer","slug":"software-engineer","activationDate":"2026-04-20T01:25:54.767174+00:00","description":"short","qualifications":"","tenure":"Full time","isWorkFromHome":false,"isHybrid":false,"companyName":"PT A","googleLocation":{"addressComponents":{"city":"Jakarta","region":"DKI Jakarta","country":"Indonesia"}},"applyRedirectUrl":null}]}}}</script></body></html>`
		}
		if r.URL.Path == "/home/te/backend/loc/Indonesia" {
			body = `<!DOCTYPE html><html><body><script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"jobs":[{"id":1,"name":"Software Engineer","slug":"software-engineer","activationDate":"2026-04-21T01:25:54.767174+00:00","description":"much longer description","qualifications":"with qualifications","tenure":"Full time","isWorkFromHome":true,"isHybrid":false,"companyName":"PT A","googleLocation":{"addressComponents":{"city":"Jakarta","region":"DKI Jakarta","country":"Indonesia"}},"applyRedirectUrl":null}]}}}</script></body></html>`
		}
		fmt.Fprint(w, body)
	}))
	defer server.Close()

	repo := NewAggregatorRepository(server.URL+"/home/te/software/loc/Indonesia", []string{"software", "backend"})
	jobs, err := repo.FetchJobs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 deduplicated job, got %d", len(jobs))
	}
	if !jobs[0].Remote {
		t.Fatal("expected richer candidate with remote=true to win dedup")
	}
	if jobs[0].PublishedAt == nil || jobs[0].PublishedAt.Format("2006-01-02") != "2026-04-21" {
		t.Fatalf("expected newer candidate to win, got %+v", jobs[0].PublishedAt)
	}
}
