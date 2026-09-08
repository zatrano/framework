package kernel_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel"
)

type namedPhaseFail struct {
	name  string
	phase string
	err   error
}

func (p *namedPhaseFail) Name() string { return p.name }
func (p *namedPhaseFail) Register(contracts.App) error {
	if p.phase == "register" {
		return p.err
	}
	return nil
}
func (p *namedPhaseFail) Boot(contracts.App) error {
	if p.phase == "boot" {
		return p.err
	}
	return nil
}
func (p *namedPhaseFail) Start(contracts.App) error {
	if p.phase == "start" {
		return p.err
	}
	return nil
}
func (p *namedPhaseFail) Stop(context.Context) error { return nil }

func TestBootstrapRegisterFailureIdentifiesProviderAndPhase(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	cause := errors.New("connection refused")
	app.RegisterProviders(&namedPhaseFail{name: "redis", phase: "register", err: cause})
	err := app.Bootstrap()
	if err == nil {
		t.Fatal("expected register failure")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("must unwrap cause, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "boot failed") || !strings.Contains(msg, "provider redis") || !strings.Contains(msg, "phase register") {
		t.Fatalf("register error must identify provider and phase: %v", err)
	}
}

func TestBootstrapBootFailureIdentifiesProviderAndPhase(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	cause := errors.New("connection refused")
	app.RegisterProviders(&namedPhaseFail{name: "redis", phase: "boot", err: cause})
	err := app.Bootstrap()
	if err == nil {
		t.Fatal("expected boot failure")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("must unwrap cause, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "boot failed") || !strings.Contains(msg, "provider redis") || !strings.Contains(msg, "phase boot") {
		t.Fatalf("boot error must identify provider and phase: %v", err)
	}
}

func TestStartFailureIdentifiesProviderAndPhase(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	cause := errors.New("connection refused")
	app.RegisterProviders(&namedPhaseFail{name: "redis", phase: "start", err: cause})
	err := app.Start()
	if err == nil {
		t.Fatal("expected start failure")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("must unwrap cause, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "start failed") || !strings.Contains(msg, "provider redis") || !strings.Contains(msg, "phase start") {
		t.Fatalf("start error must identify provider and phase: %v", err)
	}
}
