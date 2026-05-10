package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stockvacancy/internal/config"
	"stockvacancy/internal/database"
	"stockvacancy/internal/handler"
	mysqlrepo "stockvacancy/internal/repository/mysql"
	sourceRepo "stockvacancy/internal/repository/source"
	"stockvacancy/internal/scheduler"
	"stockvacancy/internal/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := database.NewMySQL(cfg.DSN())
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	jobRepo := mysqlrepo.NewJobRepository(db)
	syncLogRepo := mysqlrepo.NewSyncLogRepository(db)
	source := sourceRepo.NewAggregatorRepository(cfg.SourceAPIURL, cfg.SourceQueries)
	jobUsecase := usecase.NewJobUsecase(jobRepo, syncLogRepo, source)
	httpHandler := handler.NewHTTPHandler(jobUsecase)

	// start auto-sync scheduler
	syncInterval := time.Duration(cfg.SyncIntervalMinutes) * time.Minute
	if syncInterval <= 0 {
		syncInterval = 60 * time.Minute
	}
	syncScheduler := scheduler.New(syncInterval, func(ctx context.Context) error {
		result, syncErr := jobUsecase.SyncJobs(ctx)
		if syncErr != nil {
			return syncErr
		}
		log.Printf("[scheduler] sync result: source=%s fetched=%d inserted=%d updated=%d",
			result.Source, result.TotalFetched, result.TotalInserted, result.TotalUpdated)
		return nil
	})
	syncScheduler.Start()

	server := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      httpHandler.RegisterRoutes(),
		ReadTimeout:  time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeoutSeconds) * time.Second,
	}

	go func() {
		log.Printf("server listening on :%s", cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("shutting down...")
	syncScheduler.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown server: %v", err)
	}
}
