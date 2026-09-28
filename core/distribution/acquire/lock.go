package acquire

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

var mutationGates sync.Map // canonical module root -> *mutationGate

type mutationGate struct {
	mu sync.Mutex
}

// lockMutation serializes go.mod / go.sum mutation for one module root.
// It does not lock Inspect, write a lockfile, or span other roots.
func lockMutation(ctx context.Context, root string) (func(), error) {
	if ctx == nil {
		return nil, fmt.Errorf("acquire: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("acquire: module root required")
	}
	key, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	key = filepath.Clean(key)
	actual, _ := mutationGates.LoadOrStore(key, &mutationGate{})
	g := actual.(*mutationGate)
	locked := make(chan struct{})
	go func() {
		g.mu.Lock()
		close(locked)
	}()
	select {
	case <-locked:
		return g.mu.Unlock, nil
	case <-ctx.Done():
		go func() {
			<-locked
			g.mu.Unlock()
		}()
		return nil, ctx.Err()
	}
}
