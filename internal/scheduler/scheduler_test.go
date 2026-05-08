package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerRunsImmediatelyOnStart(t *testing.T) {
	var count int32
	s := New(10*time.Second, func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	s.Start()
	time.Sleep(50 * time.Millisecond)
	s.Stop()

	if got := atomic.LoadInt32(&count); got < 1 {
		t.Fatalf("expected at least 1 call, got %d", got)
	}
}

func TestSchedulerTicksOnInterval(t *testing.T) {
	var count int32
	s := New(50*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	s.Start()
	time.Sleep(180 * time.Millisecond)
	s.Stop()

	got := atomic.LoadInt32(&count)
	// expect at least 3 calls: immediate + 2 ticks within 180ms
	if got < 3 {
		t.Fatalf("expected at least 3 calls, got %d", got)
	}
}

func TestSchedulerStopsCleanly(t *testing.T) {
	stopped := make(chan struct{})
	s := New(1*time.Second, func(ctx context.Context) error {
		return nil
	})
	s.Start()
	go func() {
		s.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return within 2s")
	}
}

func TestSchedulerLogsErrorButContinues(t *testing.T) {
	var count int32
	s := New(30*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return errors.New("simulated error")
	})
	s.Start()
	time.Sleep(100 * time.Millisecond)
	s.Stop()

	if got := atomic.LoadInt32(&count); got < 2 {
		t.Fatalf("expected scheduler to keep running after error, got %d calls", got)
	}
}
