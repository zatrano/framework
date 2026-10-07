package kernel

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/kernel/http"
)

func TestShutdownHooksRunBeforeStopAndDoNotAbort(t *testing.T) {
	http.ResetShutdownState()
	t.Cleanup(http.ResetShutdownState)

	app := NewApplication(t.TempDir())
	t.Cleanup(func() {
		if app.logger != nil {
			_ = app.logger.Close()
		}
	})
	prov := &shutdownOrderProvider{}
	app.RegisterProviders(prov)
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ln := serveTestServer(t, srv)
	addr := ln.Addr().String()
	prov.addr = addr

	var mu sync.Mutex
	var order []string
	http.RegisterShutdownHook("order", func(context.Context) error {
		c, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			return err
		}
		_ = c.Close()
		mu.Lock()
		order = append(order, "hook")
		mu.Unlock()
		return nil
	})
	http.RegisterShutdownHook("panic", func(context.Context) error {
		panic("hook blew up")
	})
	http.RegisterShutdownHook("fail", func(context.Context) error {
		return errors.New("hook failed")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.gracefulShutdown(ctx, srv); err != nil {
		t.Fatal(err)
	}
	if len(order) != 1 || order[0] != "hook" {
		t.Fatalf("order %v", order)
	}
	if !prov.stopped {
		t.Fatal("Stop did not run")
	}
	if prov.stillOpen {
		t.Fatal("listener was still accepting during Stop")
	}
}

func TestShutdownHooksAreIdempotentAndIgnoreLateRegistration(t *testing.T) {
	http.ResetShutdownState()
	t.Cleanup(http.ResetShutdownState)

	var n int
	http.RegisterShutdownHook("count", func(context.Context) error {
		n++
		return nil
	})
	if err := http.RunShutdownHooks(context.Background()); len(err) != 0 {
		t.Fatal(err)
	}
	if err := http.RunShutdownHooks(context.Background()); len(err) != 0 || n != 1 {
		t.Fatalf("second call n=%d err=%v", n, err)
	}
	http.RegisterShutdownHook("late", func(context.Context) error {
		n += 10
		return nil
	})
	if err := http.RunShutdownHooks(context.Background()); len(err) != 0 || n != 1 {
		t.Fatalf("late hook ran: n=%d err=%v", n, err)
	}
}

type shutdownOrderProvider struct {
	addr      string
	stopped   bool
	stillOpen bool
}

func (p *shutdownOrderProvider) Register(contracts.App) error { return nil }
func (p *shutdownOrderProvider) Boot(contracts.App) error     { return nil }
func (p *shutdownOrderProvider) Start(contracts.App) error    { return nil }
func (p *shutdownOrderProvider) Stop(context.Context) error {
	p.stopped = true
	c, err := net.DialTimeout("tcp", p.addr, 300*time.Millisecond)
	if err == nil {
		_ = c.Close()
		p.stillOpen = true
	}
	return nil
}
