package concurrency

import (
	"context"
	"fmt"
	"sync"
)

// Result holds a concurrent task outcome.
type Result[T any] struct {
	Value T
	Err   error
	Key   string
}

// Run runs tasks concurrently and returns results in completion order.
func Run(tasks ...func() error) []error {
	errs := make([]error, len(tasks))
	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		go func(i int, task func() error) {
			defer wg.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					errs[i] = fmt.Errorf("panic: %v", recovered)
				}
			}()
			errs[i] = task()
		}(i, task)
	}
	wg.Wait()
	return errs
}

// Map runs keyed tasks concurrently and returns keyed results.
func Map[T any](tasks map[string]func() (T, error)) map[string]Result[T] {
	out := make(map[string]Result[T], len(tasks))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for key, task := range tasks {
		wg.Add(1)
		go func(key string, task func() (T, error)) {
			defer wg.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					mu.Lock()
					var zero T
					out[key] = Result[T]{Key: key, Err: fmt.Errorf("panic: %v", recovered), Value: zero}
					mu.Unlock()
				}
			}()
			value, err := task()
			mu.Lock()
			out[key] = Result[T]{Key: key, Value: value, Err: err}
			mu.Unlock()
		}(key, task)
	}

	wg.Wait()
	return out
}

// Pool runs tasks with a limited number of workers.
func Pool(ctx context.Context, workers int, tasks []func(context.Context) error) []error {
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan int)
	errs := make([]error, len(tasks))
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				select {
				case <-ctx.Done():
					errs[idx] = ctx.Err()
					continue
				default:
				}
				func(i int) {
					defer func() {
						if recovered := recover(); recovered != nil {
							errs[i] = fmt.Errorf("panic: %v", recovered)
						}
					}()
					errs[i] = tasks[i](ctx)
				}(idx)
			}
		}()
	}

	go func() {
		defer close(jobs)
		for i := range tasks {
			select {
			case <-ctx.Done():
				return
			case jobs <- i:
			}
		}
	}()

	wg.Wait()
	return errs
}

// First returns the first successful result or the last error.
func First[T any](tasks ...func() (T, error)) (T, error) {
	type outcome struct {
		value T
		err   error
	}
	ch := make(chan outcome, len(tasks))
	for _, task := range tasks {
		go func(task func() (T, error)) {
			defer func() {
				if recovered := recover(); recovered != nil {
					var zero T
					ch <- outcome{value: zero, err: fmt.Errorf("panic: %v", recovered)}
				}
			}()
			value, err := task()
			ch <- outcome{value: value, err: err}
		}(task)
	}

	var zero T
	var last error
	for i := 0; i < len(tasks); i++ {
		result := <-ch
		if result.err == nil {
			return result.value, nil
		}
		last = result.err
	}
	return zero, last
}
