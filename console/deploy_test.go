package console

import (
	"os"
	"strings"
	"testing"
)

func TestDeployBuildTargetsGeneratedApp(t *testing.T) {
	raw, err := os.ReadFile("deploy.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	if strings.Contains(src, "./cmd/zatrano") {
		t.Fatal("deploy:build must not compile the host CLI as the application")
	}
	if !strings.Contains(src, `"./cmd/app"`) {
		t.Fatal("deploy:build must compile the generated application entrypoint")
	}
}
