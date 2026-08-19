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

func runCmd(bin string, args []string, env []string, stdin []byte) error {
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
		return fmt.Errorf("%s: %s", bin, msg)
	}
	return nil
}
