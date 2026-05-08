package scheduler

import (
	"context"
	"log"
	"time"
)

// SyncFunc is the function called on each tick.
type SyncFunc func(ctx context.Context) error

// Scheduler runs a SyncFunc on a fixed interval.
type Scheduler struct {
	interval time.Duration
	fn       SyncFunc
	stop     chan struct{}
	done     chan struct{}
}

// New creates a Scheduler. interval must be > 0.
func New(interval time.Duration, fn SyncFunc) *Scheduler {
	return &Scheduler{
		interval: interval,
		fn:       fn,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start launches the scheduler in a background goroutine.
// It runs the sync immediately on start, then on every interval tick.
// Call Stop to shut it down gracefully.
func (s *Scheduler) Start() {
	go s.run()
}

func (s *Scheduler) run() {
	defer close(s.done)

	log.Printf("[scheduler] starting, interval=%s", s.interval)

	// run immediately on start
	s.tick()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-s.stop:
			log.Printf("[scheduler] stopped")
			return
		}
	}
}

func (s *Scheduler) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), s.interval)
	defer cancel()

	log.Printf("[scheduler] sync started")
	start := time.Now()
	if err := s.fn(ctx); err != nil {
		log.Printf("[scheduler] sync failed after %s: %v", time.Since(start).Round(time.Millisecond), err)
		return
	}
	log.Printf("[scheduler] sync done in %s", time.Since(start).Round(time.Millisecond))
}

// Stop signals the scheduler to stop and waits for the current tick to finish.
func (s *Scheduler) Stop() {
	close(s.stop)
	<-s.done
}
