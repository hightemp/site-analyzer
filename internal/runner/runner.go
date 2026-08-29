// Package runner executes scans with bounded concurrency.
package runner

import (
	"context"
	"fmt"
	"sync"

	"site-analyzer/internal/model"
)

// SiteScanner is the contract implemented by a website scanner.
type SiteScanner interface {
	Scan(ctx context.Context, target string) model.Result
}

// Runner distributes targets over a fixed-size worker pool.
type Runner struct {
	scanner SiteScanner
	workers int
}

// New creates a Runner with bounded concurrency.
func New(siteScanner SiteScanner, workers int) (*Runner, error) {
	if siteScanner == nil {
		return nil, fmt.Errorf("scanner is required")
	}
	if workers <= 0 {
		return nil, fmt.Errorf("worker count must be greater than zero")
	}
	return &Runner{scanner: siteScanner, workers: workers}, nil
}

type job struct {
	index  int
	target string
}

type indexedResult struct {
	index  int
	result model.Result
}

// Run scans all targets and returns results in input order.
func (r *Runner) Run(ctx context.Context, targets []string) ([]model.Result, error) {
	if len(targets) == 0 {
		return []model.Result{}, nil
	}

	jobs := make(chan job)
	results := make(chan indexedResult)
	var workers sync.WaitGroup

	workerCount := min(r.workers, len(targets))
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case current, ok := <-jobs:
					if !ok {
						return
					}
					result := r.scanner.Scan(ctx, current.target)
					select {
					case results <- indexedResult{index: current.index, result: result}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for index, target := range targets {
			select {
			case jobs <- job{index: index, target: target}:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		workers.Wait()
		close(results)
	}()

	ordered := make([]model.Result, len(targets))
	done := make([]bool, len(targets))
	for item := range results {
		ordered[item.index] = item.result
		done[item.index] = true
	}
	if err := ctx.Err(); err != nil {
		partial := make([]model.Result, 0, len(targets))
		for index, completed := range done {
			if completed {
				partial = append(partial, ordered[index])
			}
		}
		return partial, fmt.Errorf("scan interrupted: %w", err)
	}
	return ordered, nil
}
