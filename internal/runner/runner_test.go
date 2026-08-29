package runner

import (
	"context"
	"sync"
	"testing"
	"time"

	"site-analyzer/internal/model"
)

type trackingScanner struct {
	mu        sync.Mutex
	active    int
	maxActive int
}

func (s *trackingScanner) Scan(_ context.Context, target string) model.Result {
	s.mu.Lock()
	s.active++
	s.maxActive = max(s.maxActive, s.active)
	s.mu.Unlock()

	time.Sleep(10 * time.Millisecond)

	s.mu.Lock()
	s.active--
	s.mu.Unlock()
	return model.Result{InputURL: target}
}

func TestRunnerRun(t *testing.T) {
	t.Parallel()

	s := &trackingScanner{}
	r, err := New(s, 2)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	targets := []string{"one", "two", "three", "four"}
	results, err := r.Run(context.Background(), targets)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for i, result := range results {
		if result.InputURL != targets[i] {
			t.Fatalf("result[%d] = %q, want %q", i, result.InputURL, targets[i])
		}
	}
	if s.maxActive != 2 {
		t.Fatalf("maximum concurrency = %d, want 2", s.maxActive)
	}
}

func TestNewValidationAndEmptyRun(t *testing.T) {
	t.Parallel()

	if _, err := New(nil, 1); err == nil {
		t.Fatal("New(nil) error = nil")
	}
	if _, err := New(&trackingScanner{}, 0); err == nil {
		t.Fatal("New(workers=0) error = nil")
	}
	r, err := New(&trackingScanner{}, 1)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	results, err := r.Run(context.Background(), nil)
	if err != nil || len(results) != 0 {
		t.Fatalf("Run(nil) = %v, %v; want empty success", results, err)
	}
}

func TestRunnerCancellation(t *testing.T) {
	t.Parallel()

	r, err := New(&trackingScanner{}, 1)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results, err := r.Run(ctx, []string{"one", "two"})
	if err == nil || len(results) != 0 {
		t.Fatalf("Run(cancelled) = %v, %v; want no results and an error", results, err)
	}
}
