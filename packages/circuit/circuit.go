package circuit

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// State is the breaker state.
type State int

const (
	Closed State = iota
	Open
	HalfOpen
)

func (s State) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrOpen is returned when the circuit is open.
var ErrOpen = errors.New("circuit breaker is open")

// Settings configures a breaker.
type Settings struct {
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
}

// Breaker is a consecutive-failure circuit breaker.
type Breaker struct {
	mu               sync.Mutex
	name             string
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	state            State
	failures         int
	successes        int
	openedAt         time.Time
}

// Manager holds named breakers.
type Manager struct {
	mu       sync.Mutex
	breakers map[string]*Breaker
	defaults Settings
}

// New creates a breaker manager.
func New(defaults ...Settings) *Manager {
	s := Settings{FailureThreshold: 5, SuccessThreshold: 2, Timeout: 30 * time.Second}
	if len(defaults) > 0 {
		if defaults[0].FailureThreshold > 0 {
			s.FailureThreshold = defaults[0].FailureThreshold
		}
		if defaults[0].SuccessThreshold > 0 {
			s.SuccessThreshold = defaults[0].SuccessThreshold
		}
		if defaults[0].Timeout > 0 {
			s.Timeout = defaults[0].Timeout
		}
	}
	return &Manager{breakers: make(map[string]*Breaker), defaults: s}
}

// Breaker returns (or creates) a named breaker.
func (m *Manager) Breaker(name string) *Breaker {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.breakers[name]; ok {
		return b
	}
	b := &Breaker{
		name:             name,
		failureThreshold: m.defaults.FailureThreshold,
		successThreshold: m.defaults.SuccessThreshold,
		timeout:          m.defaults.Timeout,
		state:            Closed,
	}
	m.breakers[name] = b
	return b
}

// Execute runs fn through the named breaker.
func (m *Manager) Execute(name string, fn func() error) error {
	return m.Breaker(name).Execute(fn)
}

// Allow reports whether a call may proceed.
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.transitionLocked()
	return b.state != Open
}

// Execute runs fn if allowed.
func (b *Breaker) Execute(fn func() error) error {
	if !b.Allow() {
		return fmt.Errorf("%w: %s", ErrOpen, b.name)
	}
	err := fn()
	if err != nil {
		b.RecordFailure()
		return err
	}
	b.RecordSuccess()
	return nil
}

// RecordSuccess records a successful call.
func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	if b.state == HalfOpen {
		b.successes++
		if b.successes >= b.successThreshold {
			b.state = Closed
			b.successes = 0
		}
		return
	}
	b.state = Closed
}

// RecordFailure records a failed call.
func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.successes = 0
	b.failures++
	if b.state == HalfOpen || b.failures >= b.failureThreshold {
		b.state = Open
		b.openedAt = time.Now()
		b.failures = 0
	}
}

// State returns the current state.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.transitionLocked()
	return b.state
}

// Stats returns breaker diagnostics.
func (b *Breaker) Stats() map[string]any {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.transitionLocked()
	return map[string]any{
		"name":     b.name,
		"state":    b.state.String(),
		"failures": b.failures,
	}
}

func (b *Breaker) transitionLocked() {
	if b.state == Open && time.Since(b.openedAt) >= b.timeout {
		b.state = HalfOpen
		b.successes = 0
	}
}
