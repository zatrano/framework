package pkgmanager

import (
	"os"
	"strings"
	"testing"
)

func TestEnablementCommandsShareIntentionalDifferences(t *testing.T) {
	acquireSrc, err := os.ReadFile("package_acquire.go")
	if err != nil {
		t.Fatal(err)
	}
	acquireView, err := os.ReadFile("package_acquire_view.go")
	if err != nil {
		t.Fatal(err)
	}
	cmdSrc, err := os.ReadFile("package_cmd.go")
	if err != nil {
		t.Fatal(err)
	}
	enableSrc, err := os.ReadFile("package_enable.go")
	if err != nil {
		t.Fatal(err)
	}
	acquireText := string(acquireSrc) + "\n" + string(acquireView)
	cmdText := string(cmdSrc)
	enableText := string(enableSrc)

	if !strings.Contains(acquireText, "enablePackage(") || !strings.Contains(acquireText, "wireEnablement(") {
		t.Fatal("package:acquire --enable must reuse enablePackage and wireEnablement")
	}
	if strings.Contains(acquireText, "publishPackage(") {
		t.Fatal("package:acquire --enable must not publish stubs; package:install owns stubs")
	}
	if !strings.Contains(cmdText, `return "package:install"`) || !strings.Contains(cmdText, "publishPackage(") {
		t.Fatal("package:install remains enablement plus stub publication")
	}
	if !strings.Contains(enableText, `return "package:enable"`) {
		t.Fatal("package:enable must remain a distinct command")
	}

	enableIdx := strings.Index(enableText, "func (c *PackageEnableCommand) Handle")
	if enableIdx < 0 {
		t.Fatal("missing enable handler")
	}
	enableBody := enableText[enableIdx:]
	if !strings.Contains(enableBody, `fmt.Printf("Note: %v\n", err)`) {
		t.Fatal("package:enable treats wire failure as a Note, not a command error")
	}
	if strings.Contains(acquireText, `fmt.Printf("Note:`) {
		t.Fatal("package:acquire --enable must not swallow wire failure as a Note")
	}
	if !strings.Contains(acquireText, "cliFailed(ExitEnablement") && !strings.Contains(acquireText, "cliErr(ExitEnablement, err)") {
		t.Fatal("package:acquire --enable must classify wire/enablement failure distinctly from acquisition success")
	}
}
