package kernel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
	klog "github.com/zatrano/framework/v3/core/kernel/log"
)

func setOrUnset(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
	if value == "" {
		os.Unsetenv(key)
	}
}

func poolWarningLog(t *testing.T, value string) string {
	t.Helper()
	setOrUnset(t, "HTTP_POOL_REQUESTS", value)
	path := filepath.Join(t.TempDir(), "app.log")
	lg, err := klog.New("debug", path)
	if err != nil {
		t.Fatal(err)
	}
	app := &Application{logger: lg}
	app.warnPoolEnv()
	_ = lg.Close()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestHTTPPoolRequestsIsDeprecatedAndIgnored(t *testing.T) {
	for _, value := range []string{"true", "false", "1"} {
		out := poolWarningLog(t, value)
		if strings.Count(out, "HTTP_POOL_REQUESTS is deprecated and ignored") != 1 {
			t.Fatalf("HTTP_POOL_REQUESTS=%q: want one deprecation warning, log:\n%s", value, out)
		}
		if http.RequestPooling() {
			t.Fatalf("HTTP_POOL_REQUESTS=%q turned request pooling on", value)
		}
	}
	if out := poolWarningLog(t, ""); strings.Contains(out, "HTTP_POOL_REQUESTS") {
		t.Fatalf("unset HTTP_POOL_REQUESTS logged a warning:\n%s", out)
	}
}

func TestHTTPPoolRequestsDoesNotEnablePooling(t *testing.T) {
	t.Setenv("HTTP_POOL_REQUESTS", "true")
	http.SetRequestPooling(true, true)
	if http.RequestPooling() {
		t.Fatal("SetRequestPooling(true) enabled pooling")
	}
	a := http.AcquireRequest(nil)
	http.ReleaseRequest(a)
	b := http.AcquireRequest(nil)
	if a == b {
		t.Fatal("AcquireRequest returned a released request")
	}
}
