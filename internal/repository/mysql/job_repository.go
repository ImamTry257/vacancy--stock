package mysqlrepo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"stockvacancy/internal/entity"
	"stockvacancy/internal/repository"
)

type rowScanner interface {
	Scan(dest ...any) error
}

type transaction interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Commit() error
	Rollback() error
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) rowScanner
	BeginTx(ctx context.Context, opts *sql.TxOptions) (transaction, error)
}

type sqlDB struct {
	db *sql.DB
}

func (s sqlDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

func (s sqlDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

func (s sqlDB) QueryRowContext(ctx context.Context, query string, args ...any) rowScanner {
	return s.db.QueryRowContext(ctx, query, args...)
}

func (s sqlDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (transaction, error) {
	return s.db.BeginTx(ctx, opts)
}

type JobRepository struct {
	exec sqlExecutor
}

func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{exec: sqlDB{db: db}}
}

func (r *JobRepository) UpsertJobs(ctx context.Context, jobs []entity.Job) (int, int, error) {
	return r.upsertJobs(ctx, jobs)
}

func (r *JobRepository) upsertJobs(ctx context.Context, jobs []entity.Job) (int, int, error) {
	if len(jobs) == 0 {
		return 0, 0, nil
	}

	tx, err := r.exec.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("begin tx: %w", err)
	}

	inserted := 0
	updated := 0

	query := `
		INSERT INTO jobs (
			external_id, title, company_name, location, employment_type,
			salary_text, is_remote, is_international, url, source, description, published_at, scraped_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			title = VALUES(title),
			company_name = VALUES(company_name),
			location = VALUES(location),
			employment_type = VALUES(employment_type),
			salary_text = VALUES(salary_text),
			is_remote = VALUES(is_remote),
			is_international = VALUES(is_international),
			url = VALUES(url),
			description = VALUES(description),
			published_at = VALUES(published_at),
			scraped_at = VALUES(scraped_at),
			updated_at = NOW()
	`

	for _, job := range jobs {
		publishedAt := any(nil)
		if job.PublishedAt != nil {
			publishedAt = *job.PublishedAt
		}

		result, execErr := tx.ExecContext(ctx, query,
			job.ExternalID,
			job.Title,
			job.CompanyName,
			job.Location,
			job.EmploymentType,
			job.SalaryText,
			job.Remote,
			job.IsInternational,
			job.URL,
			job.Source,
			job.Description,
			publishedAt,
			job.ScrapedAt,
		)
		if execErr != nil {
			_ = tx.Rollback()
			return 0, 0, fmt.Errorf("upsert job %s: %w", job.ExternalID, execErr)
		}

		rowsAffected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			_ = tx.Rollback()
			return 0, 0, fmt.Errorf("rows affected job %s: %w", job.ExternalID, rowsErr)
		}

		switch rowsAffected {
		case 1:
			inserted++
		case 2:
			updated++
		}
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return 0, 0, fmt.Errorf("commit tx: %w", err)
	}

	return inserted, updated, nil
}

func (r *JobRepository) ListJobs(ctx context.Context, filter repository.JobFilter) ([]entity.Job, int, error) {
	conditions := []string{"1=1"}
	args := make([]any, 0)

	if filter.Search != "" {
		conditions = append(conditions, "(title LIKE ? OR company_name LIKE ? OR description LIKE ?)")
		like := "%" + filter.Search + "%"
		args = append(args, like, like, like)
	}
	if filter.Location != "" {
		conditions = append(conditions, "location LIKE ?")
		args = append(args, "%"+filter.Location+"%")
	}
	if filter.EmploymentType != "" {
		conditions = append(conditions, "employment_type LIKE ?")
		args = append(args, "%"+filter.EmploymentType+"%")
	}
	if filter.Remote != nil {
		conditions = append(conditions, "is_remote = ?")
		args = append(args, *filter.Remote)
	}
	if filter.IsInternational != nil {
		conditions = append(conditions, "is_international = ?")
		args = append(args, *filter.IsInternational)
	}

	whereClause := strings.Join(conditions, " AND ")
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM jobs WHERE %s", whereClause)

	var total int
	if err := r.exec.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count jobs: %w", err)
	}

	orderCol := allowedSortColumn(filter.SortBy)
	orderDir := "DESC"
	if strings.EqualFold(filter.SortDir, "asc") {
		orderDir = "ASC"
	}

	listQuery := fmt.Sprintf(`
		SELECT id, external_id, title, company_name, location, employment_type,
			salary_text, is_remote, is_international, url, source, description, published_at, scraped_at,
			created_at, updated_at
		FROM jobs
		WHERE %s
		ORDER BY %s %s, id DESC
		LIMIT ? OFFSET ?
	`, whereClause, orderCol, orderDir)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.exec.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]entity.Job, 0)
	for rows.Next() {
		var job entity.Job
		var publishedAt sql.NullTime
		if err := rows.Scan(
			&job.ID,
			&job.ExternalID,
			&job.Title,
			&job.CompanyName,
			&job.Location,
			&job.EmploymentType,
			&job.SalaryText,
			&job.Remote,
			&job.IsInternational,
			&job.URL,
			&job.Source,
			&job.Description,
			&publishedAt,
			&job.ScrapedAt,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan job row: %w", err)
		}
		if publishedAt.Valid {
			job.PublishedAt = &publishedAt.Time
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate jobs rows: %w", err)
	}

	return jobs, total, nil
}

// allowedSortColumn returns a safe SQL column name for ORDER BY.
// Only whitelisted values are accepted; anything else falls back to the default.
func allowedSortColumn(col string) string {
	switch strings.ToLower(strings.TrimSpace(col)) {
	case "title":
		return "title"
	case "company_name":
		return "company_name"
	case "created_at":
		return "created_at"
	case "published_at":
		return "COALESCE(published_at, scraped_at)"
	default:
		return "COALESCE(published_at, scraped_at)"
	}
}

func (r *JobRepository) GetJobByID(ctx context.Context, id int64) (*entity.Job, error) {
	query := `
		SELECT id, external_id, title, company_name, location, employment_type,
			salary_text, is_remote, is_international, url, source, description, published_at, scraped_at,
			created_at, updated_at
		FROM jobs
		WHERE id = ?
	`

	var job entity.Job
	var publishedAt sql.NullTime
	if err := r.exec.QueryRowContext(ctx, query, id).Scan(
		&job.ID,
		&job.ExternalID,
		&job.Title,
		&job.CompanyName,
		&job.Location,
		&job.EmploymentType,
		&job.SalaryText,
		&job.Remote,
		&job.IsInternational,
		&job.URL,
		&job.Source,
		&job.Description,
		&publishedAt,
		&job.ScrapedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get job by id: %w", err)
	}
	if publishedAt.Valid {
		job.PublishedAt = &publishedAt.Time
	}
	return &job, nil
}

func NewJobRepositoryWithExecutor(exec sqlExecutor) *JobRepository {
	return &JobRepository{exec: exec}
}

var _ repository.JobRepository = (*JobRepository)(nil)
