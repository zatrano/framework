package fn

import (
	"sync"
	"time"
)

// Debounce returns a function that delays fn until wait has elapsed since the last call.
func Debounce(wait time.Duration, fn func()) func() {
	var (
		mu    sync.Mutex
		timer *time.Timer
	)
	return func() {
		mu.Lock()
		defer mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(wait, fn)
	}
}

// Throttle returns a function that invokes fn at most once per interval.
func Throttle(interval time.Duration, fn func()) func() {
	var (
		mu        sync.Mutex
		last      time.Time
		scheduled bool
	)
	return func() {
		mu.Lock()
		defer mu.Unlock()
		now := time.Now()
		if last.IsZero() || now.Sub(last) >= interval {
			last = now
			fn()
			return
		}
		if scheduled {
			return
		}
		scheduled = true
		delay := interval - now.Sub(last)
		time.AfterFunc(delay, func() {
			mu.Lock()
			scheduled = false
			last = time.Now()
			mu.Unlock()
			fn()
		})
	}
}

// Once returns a function that runs fn at most once.
func Once(fn func()) func() {
	var (
		once sync.Once
	)
	return func() {
		once.Do(fn)
	}
}
