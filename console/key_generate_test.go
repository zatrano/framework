package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestKeyGenerateSeedsFromExample(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte("APP_NAME=Demo\nAPP_KEY=\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := &KeyGenerateCommand{app: kernel.NewApplication(dir)}
	if err := cmd.Handle(nil); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "APP_NAME=Demo") {
		t.Fatalf("seeded .env lost example keys:\n%s", text)
	}
	if !strings.Contains(text, "APP_KEY=base64:") {
		t.Fatalf("expected generated APP_KEY:\n%s", text)
	}
}

func TestKeyGenerateUpdatesExistingEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("APP_NAME=Demo\nAPP_KEY=\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := &KeyGenerateCommand{app: kernel.NewApplication(dir)}
	if err := cmd.Handle(nil); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "APP_KEY=\n") || strings.Contains(text, "APP_KEY=\r") {
		t.Fatalf("APP_KEY still empty:\n%s", text)
	}
	if !strings.Contains(text, "APP_KEY=base64:") {
		t.Fatalf("expected generated APP_KEY:\n%s", text)
	}
}

func TestKeyGenerateMissingExample(t *testing.T) {
	cmd := &KeyGenerateCommand{app: kernel.NewApplication(t.TempDir())}
	err := cmd.Handle(nil)
	if err == nil || !strings.Contains(err.Error(), ".env not found") || !strings.Contains(err.Error(), "Next:") {
		t.Fatalf("got %v", err)
	}
}

func TestSeedEnvFromExampleLeavesExisting(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("APP_KEY=keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte("APP_KEY=\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := seedEnvFromExample(dir); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "APP_KEY=keep\n" {
		t.Fatalf("existing .env was rewritten: %q", body)
	}
}
