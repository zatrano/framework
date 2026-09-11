package console

import (
	"os"
	"path/filepath"
)

// seedEnvFromExample copies .env.example to .env when .env is missing.
// Existing .env files are left unchanged.
func seedEnvFromExample(root string) error {
	envPath := filepath.Join(root, ".env")
	if _, err := os.Stat(envPath); !os.IsNotExist(err) {
		return err
	}
	examplePath := filepath.Join(root, ".env.example")
	raw, err := os.ReadFile(examplePath)
	if err != nil {
		return err
	}
	return os.WriteFile(envPath, raw, 0o644)
}
