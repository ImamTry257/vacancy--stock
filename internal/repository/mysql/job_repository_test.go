package mysqlrepo

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"stockvacancy/internal/entity"
)

type stubExecResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (s stubExecResult) LastInsertId() (int64, error) { return s.lastInsertID, nil }
func (s stubExecResult) RowsAffected() (int64, error) { return s.rowsAffected, nil }

type stubDB struct {
	execFn      func(context.Context, string, ...any) (sql.Result, error)
	queryFn     func(context.Context, string, ...any) (*sql.Rows, error)
	queryRowFn  func(context.Context, string, ...any) rowScanner
	beginTxFn   func(context.Context, *sql.TxOptions) (transaction, error)
}

func (s stubDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.execFn(ctx, query, args...)
}

func (s stubDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.queryFn(ctx, query, args...)
}

func (s stubDB) QueryRowContext(ctx context.Context, query string, args ...any) rowScanner {
	return s.queryRowFn(ctx, query, args...)
}

func (s stubDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (transaction, error) {
	return s.beginTxFn(ctx, opts)
}

type stubTx struct {
	execCalls   int
	execFn      func(context.Context, string, ...any) (sql.Result, error)
	commitFn    func() error
	rollbackFn  func() error
}

func (s *stubTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	s.execCalls++
	return s.execFn(ctx, query, args...)
}

func (s *stubTx) Commit() error { return s.commitFn() }
func (s *stubTx) Rollback() error { return s.rollbackFn() }

type stubRow struct {
	scanFn func(dest ...any) error
}

func (s stubRow) Scan(dest ...any) error { return s.scanFn(dest...) }

func TestUpsertJobsReturnsInsertedAndUpdatedCounts(t *testing.T) {
	tx := &stubTx{
		execFn: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			if !strings.Contains(query, "INSERT INTO jobs") {
				t.Fatalf("unexpected query: %s", query)
			}
			if len(args) != 13 {
				t.Fatalf("expected 13 args, got %d", len(args))
			}
			return stubExecResult{rowsAffected: 1}, nil
		},
		commitFn:   func() error { return nil },
		rollbackFn: func() error { return nil },
	}

	repo := NewJobRepositoryWithExecutor(stubDB{
		beginTxFn: func(ctx context.Context, opts *sql.TxOptions) (transaction, error) {
			return tx, nil
		},
	})

	jobs := []entity.Job{
		{ExternalID: "job-1", Title: "Backend Engineer", CompanyName: "Acme", Location: "Jakarta", EmploymentType: "Full Time", SalaryText: "", Remote: true, URL: "https://example.com/1", Source: "arbeitnow", Description: "desc"},
		{ExternalID: "job-2", Title: "Frontend Engineer", CompanyName: "Beta", Location: "Bandung", EmploymentType: "Contract", SalaryText: "", Remote: false, URL: "https://example.com/2", Source: "arbeitnow", Description: "desc"},
	}

	inserted, updated, err := repo.UpsertJobs(context.Background(), jobs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inserted != 2 {
		t.Fatalf("expected inserted=2, got %d", inserted)
	}
	if updated != 0 {
		t.Fatalf("expected updated=0, got %d", updated)
	}
	if tx.execCalls != 2 {
		t.Fatalf("expected 2 exec calls, got %d", tx.execCalls)
	}
}

func TestUpsertJobsRollsBackOnExecFailure(t *testing.T) {
	rolledBack := false
	tx := &stubTx{
		execFn: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return nil, errors.New("boom")
		},
		commitFn:   func() error { return nil },
		rollbackFn: func() error { rolledBack = true; return nil },
	}

	repo := NewJobRepositoryWithExecutor(stubDB{
		beginTxFn: func(ctx context.Context, opts *sql.TxOptions) (transaction, error) {
			return tx, nil
		},
	})

	_, _, err := repo.UpsertJobs(context.Background(), []entity.Job{{ExternalID: "job-1"}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !rolledBack {
		t.Fatal("expected rollback to be called")
	}
}
