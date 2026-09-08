package console

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel"
)

type failStartProvider struct{ err error }

func (p *failStartProvider) Register(contracts.App) error { return nil }
func (p *failStartProvider) Boot(contracts.App) error     { return nil }
func (p *failStartProvider) Start(contracts.App) error    { return p.err }
func (p *failStartProvider) Stop(context.Context) error   { return nil }

func TestClassifyRuntimeErrorCodes(t *testing.T) {
	boot := fmt.Errorf("%w: %w", kernel.ErrRuntimeBoot, errors.New("listen"))
	shut := fmt.Errorf("%w: %w", kernel.ErrRuntimeShutdown, errors.New("stop"))
	cases := []struct {
		err  error
		want int
	}{
		{nil, ExitSuccess},
		{boot, ExitRuntimeBoot},
		{shut, ExitRuntimeShutdown},
		{context.Canceled, ExitRuntimeCanceled},
		{fmt.Errorf("%w: %w", kernel.ErrRuntimeBoot, context.Canceled), ExitRuntimeCanceled},
		{context.DeadlineExceeded, ExitRuntimeTimeout},
		{fmt.Errorf("%w: %w", kernel.ErrRuntimeShutdown, context.DeadlineExceeded), ExitRuntimeTimeout},
		{errors.New("plain"), ExitGeneral},
	}
	for _, c := range cases {
		got := CodeFromError(classifyRuntimeError(c.err))
		if got != c.want {
			t.Errorf("err=%v code=%d want %d", c.err, got, c.want)
		}
	}
}

func TestRuntimeCancellationIsNotAcquireCancellation(t *testing.T) {
	err := classifyRuntimeError(context.Canceled)
	if CodeFromError(err) == ExitCanceled {
		t.Fatal("runtime cancellation must not use acquire ExitCanceled=7")
	}
	if CodeFromError(err) != ExitRuntimeCanceled {
		t.Fatalf("got %d", CodeFromError(err))
	}
	acquire := classifyContextError(context.Canceled)
	if CodeFromError(acquire) != ExitCanceled {
		t.Fatal("acquire cancellation stays ExitCanceled=7")
	}
}

func TestAcquisitionExitCodesUnchanged(t *testing.T) {
	if ExitUsage != 2 || ExitResolution != 3 || ExitPlanning != 4 || ExitAcquisition != 5 || ExitEnablement != 6 || ExitCanceled != 7 {
		t.Fatalf("acquisition codes drifted: 2=%d 3=%d 4=%d 5=%d 6=%d 7=%d",
			ExitUsage, ExitResolution, ExitPlanning, ExitAcquisition, ExitEnablement, ExitCanceled)
	}
	if ExitRuntimeBoot != 20 || ExitRuntimeShutdown != 21 || ExitRuntimeCanceled != 22 || ExitRuntimeTimeout != 23 {
		t.Fatal("runtime codes must be 20–23")
	}
}

func TestServeCommandClassifiesRunErrors(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	app.RegisterProviders(&failStartProvider{err: errors.New("nope")})
	cmd := &ServeCommand{app: app}
	t.Cleanup(func() {
		if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
	err := cmd.Handle([]string{"--port", "0"})
	if CodeFromError(err) != ExitRuntimeBoot {
		t.Fatalf("serve Start failure must be ExitRuntimeBoot, code=%d err=%v", CodeFromError(err), err)
	}
	if CodeFromError(err) == ExitAcquisition || CodeFromError(err) == ExitCanceled {
		t.Fatal("runtime errors must never become acquisition codes")
	}
}
