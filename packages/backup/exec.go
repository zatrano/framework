package backup

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func lookPath(names ...string) (string, error) {
	var tried []string
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
		tried = append(tried, name)
	}
	return "", fmt.Errorf("backup: required tool not found in PATH (tried: %s)", strings.Join(tried, ", "))
}

func runCmd(bin string, args []string, env []string, stdin []byte, secrets ...string) error {
	cmd := exec.Command(bin, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s: %s", bin, redactSecrets(msg, secrets...))
	}
	return nil
}

func redactSecrets(msg string, secrets ...string) string {
	for _, s := range secrets {
		if s == "" {
			continue
		}
		msg = strings.ReplaceAll(msg, s, "***")
	}
	return msg
}

// withSecretFile writes content to a 0600 temp file, runs fn, then removes the file.
func withSecretFile(pattern string, content []byte, fn func(path string) error) error {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return err
	}
	path := f.Name()
	defer func() { _ = os.Remove(path) }()
	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	_ = os.Chmod(path, 0o600)
	return fn(path)
}
