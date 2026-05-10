# vacancy-stock

REST API service written in Go that periodically fetches job listings from a configurable external job source and stores them in MySQL. Exposes endpoints to query jobs and manually trigger syncs.

---

## Tech Stack

- **Go 1.22** — standard library only (no web framework)
- **MySQL 8.4** — primary data store
- **Docker / Docker Compose** — containerised dev & deployment

---

## Architecture

```
cmd/api/main.go          — entrypoint: wires deps, starts scheduler + HTTP server
internal/
  config/                — env-based config loader
  database/              — MySQL connection setup
  entity/                — domain structs (Job, SyncLog)
  dto/                   — request/response shapes
  repository/
    mysql/               — MySQL implementations (jobs, sync_logs)
    source/              — external job source adapter (SourceRepository)
  usecase/               — business logic (SyncJobs, ListJobs, GetJobByID)
  handler/               — HTTP handler + route registration
  htmlutil/              — HTML → plain-text stripper for descriptions
  scheduler/             — ticker-based background sync scheduler
migrations/
  001_init.sql           — creates jobs + sync_logs tables
```

### Data flow

1. Scheduler fires every `SYNC_INTERVAL_MINUTES` (default 60 min).
2. `JobUsecase.SyncJobs` calls the source adapter's `FetchJobs` for each configured query keyword.
3. HTML descriptions are stripped to plain text via `htmlutil.StripHTML`.
4. Jobs are upserted into MySQL (unique key: `external_id + source`).
5. A `sync_logs` row records status, counts, and any error.

---

## Database Schema

### `jobs`

| Column           | Type          | Notes                          |
|------------------|---------------|--------------------------------|
| id               | BIGINT PK AI  |                                |
| external_id      | VARCHAR(255)  | ID from source platform        |
| title            | VARCHAR(255)  |                                |
| company_name     | VARCHAR(255)  |                                |
| location         | VARCHAR(255)  |                                |
| employment_type  | VARCHAR(255)  | e.g. Full-time, Contract       |
| salary_text      | VARCHAR(255)  | raw salary string from source  |
| is_remote        | BOOLEAN       |                                |
| is_international | BOOLEAN       |                                |
| url              | TEXT          | link to original listing       |
| source           | VARCHAR(100)  | identifier of the source       |
| description      | MEDIUMTEXT    | plain text (HTML stripped)     |
| published_at     | DATETIME NULL |                                |
| scraped_at       | DATETIME      |                                |
| created_at       | DATETIME      |                                |
| updated_at       | DATETIME      |                                |

Unique key: `(external_id, source)`

### `sync_logs`

Tracks every sync run: source, status (`running` / `success` / `failed`), counts (fetched / inserted / updated), timestamps, and error message.

---

## API Endpoints

### Health check

```
GET /health
```

Response `200`:
```json
{ "status": "ok" }
```

Example:
```bash
curl http://localhost:8080/health
```

---

### List jobs

```
GET /api/v1/jobs
```

Query params:

| Param            | Type    | Default        | Description                                           |
|------------------|---------|----------------|-------------------------------------------------------|
| page             | int     | 1              |                                                       |
| limit            | int     | 10 (max 100)   |                                                       |
| search           | string  |                | full-text search on title / company / description     |
| location         | string  |                | filter by location                                    |
| employment_type  | string  |                | filter by employment type                             |
| remote           | bool    |                | `true` / `false`                                      |
| is_international | bool    |                | `true` / `false`                                      |
| sort_by          | string  | `published_at` | `published_at`, `created_at`, `title`, `company_name` |
| sort_dir         | string  | `desc`         | `asc` / `desc`                                        |

Response `200`:
```json
{
  "data": [ { "...": "..." } ],
  "page": 1,
  "limit": 10,
  "total": 142,
  "total_pages": 15
}
```

Examples:
```bash
# basic list, first page
curl "http://localhost:8080/api/v1/jobs"

# search golang jobs, remote only
curl "http://localhost:8080/api/v1/jobs?search=golang&remote=true"

# filter by location, sort by title ascending, page 2
curl "http://localhost:8080/api/v1/jobs?location=Jakarta&sort_by=title&sort_dir=asc&page=2&limit=20"

# international jobs only
curl "http://localhost:8080/api/v1/jobs?is_international=true"

# full-time backend jobs
curl "http://localhost:8080/api/v1/jobs?search=backend&employment_type=Full-time"
```

---

### Get job by ID

```
GET /api/v1/jobs/{id}
```

Response `200`:
```json
{
  "id": 1,
  "external_id": "abc123",
  "title": "Backend Engineer",
  "company_name": "Acme Corp",
  "location": "Jakarta",
  "employment_type": "Full-time",
  "salary_text": "",
  "remote": false,
  "is_international": false,
  "url": "https://...",
  "source": "job-source",
  "description": "...",
  "published_at": "2024-05-01T00:00:00Z",
  "scraped_at": "2024-05-10T08:00:00Z",
  "created_at": "2024-05-10T08:00:00Z",
  "updated_at": "2024-05-10T08:00:00Z"
}
```

Response `404`:
```json
{ "message": "job not found" }
```

Example:
```bash
curl http://localhost:8080/api/v1/jobs/1
```

---

### Trigger manual sync

```
POST /api/v1/sync/jobs
```

Response `202`:
```json
{
  "source": "job-source",
  "total_fetched": 87,
  "total_inserted": 12,
  "total_updated": 75
}
```

Example:
```bash
curl -X POST http://localhost:8080/api/v1/sync/jobs
```

---

## Configuration

Copy `.env.example` to `.env` and adjust as needed.

| Variable                | Default                                               | Description                               |
|-------------------------|-------------------------------------------------------|-------------------------------------------|
| APP_PORT                | `8080`                                                | HTTP listen port                          |
| APP_ENV                 | `development`                                         |                                           |
| READ_TIMEOUT_SECONDS    | `10`                                                  |                                           |
| WRITE_TIMEOUT_SECONDS   | `10`                                                  |                                           |
| MYSQL_HOST              | `mysql`                                               | hostname (use `localhost` outside Docker) |
| MYSQL_PORT              | `3306`                                                |                                           |
| MYSQL_USER              | `stockvacancy`                                        |                                           |
| MYSQL_PASSWORD          | `stockvacancy`                                        |                                           |
| MYSQL_DATABASE          | `stockvacancy`                                        |                                           |
| MYSQL_PARAMS            | `parseTime=true&multiStatements=true`                 | DSN extra params                          |
| SOURCE_API_URL          | *(set in .env)*                                       | Base URL of the job source                |
| SOURCE_QUERIES          | `software,backend,frontend,...`                       | comma-separated search keywords           |
| SOURCE_TIMEOUT_SECONDS  | `20`                                                  | HTTP timeout for fetching from source     |
| SYNC_INTERVAL_MINUTES   | `60`                                                  | auto-sync interval                        |

---

## Running with Docker Compose

```bash
cp .env.example .env
# edit .env and set SOURCE_API_URL and other values

docker compose up --build
```

The MySQL container auto-runs `migrations/001_init.sql` on first start. The API waits for MySQL to be healthy before starting.

---

## Running locally (without Docker)

Requirements: Go 1.22+, a running MySQL instance.

```bash
cp .env.example .env
# edit .env: set MYSQL_HOST=localhost and your credentials

go run ./cmd/api
```

---

## Running tests

```bash
go test ./...
```

Tests run automatically inside the Docker build stage as well.

---

## Changelog

### Unreleased
- Initial implementation

### v0.1.0
- REST API with `GET /api/v1/jobs`, `GET /api/v1/jobs/{id}`, `POST /api/v1/sync/jobs`, `GET /health`
- MySQL persistence with upsert logic (dedup by `external_id + source`)
- Background scheduler with configurable interval
- HTML-to-plaintext stripping for job descriptions
- `sync_logs` table for tracking sync history
- Docker Compose setup with MySQL 8.4 and multi-stage Dockerfile
- Graceful shutdown on SIGINT/SIGTERM

---

## Project structure notes

- No external web framework — uses Go 1.22's enhanced `http.ServeMux` with method+path patterns (`GET /api/v1/jobs`).
- Only one external dependency: `github.com/go-sql-driver/mysql`.
- Graceful shutdown: SIGINT/SIGTERM stops the scheduler and drains in-flight HTTP requests within 10 seconds.
- Descriptions from the source may contain HTML; `htmlutil.StripHTML` normalises them to plain text before storage.
- The source adapter implements a `SourceRepository` interface — swap or add new sources without touching business logic.
