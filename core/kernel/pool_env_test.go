package kernel

import (
	"os"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
)

func TestPoolDebugFollowsAppDebugDefault(t *testing.T) {
	cases := []struct {
		appEnv, appDebug string
		want             bool
	}{
		{"", "", true},
		{"local", "", true},
		{"testing", "", true},
		{"production", "", false},
		{" Production ", "", false},
		{"production", "true", true},
		{"local", "false", false},
		{"production", "false", false},
	}
	for _, c := range cases {
		setOrUnset(t, "APP_ENV", c.appEnv)
		setOrUnset(t, "APP_DEBUG", c.appDebug)
		if got := poolDebugFromEnv(); got != c.want {
			t.Fatalf("APP_ENV=%q APP_DEBUG=%q: pool debug %v, want %v", c.appEnv, c.appDebug, got, c.want)
		}
	}
}

func setOrUnset(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
	if value == "" {
		os.Unsetenv(key)
	}
}

func TestHTTPPoolRequestsIsOffUntilSetTrue(t *testing.T) {
	prev, had := os.LookupEnv("HTTP_POOL_REQUESTS")
	t.Cleanup(func() {
		if had {
			os.Setenv("HTTP_POOL_REQUESTS", prev)
		} else {
			os.Unsetenv("HTTP_POOL_REQUESTS")
		}
		http.SetRequestPooling(false, false)
	})

	os.Unsetenv("HTTP_POOL_REQUESTS")
	off := NewApplication(t.TempDir())
	t.Cleanup(func() {
		if c, ok := off.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
	if snap := off.ensureBodyLimits(); snap.poolRequests || http.RequestPooling() {
		t.Fatal("unset HTTP_POOL_REQUESTS left the request pool on")
	}

	os.Setenv("HTTP_POOL_REQUESTS", "true")
	on := NewApplication(t.TempDir())
	t.Cleanup(func() {
		if c, ok := on.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
	if snap := on.ensureBodyLimits(); !snap.poolRequests || !http.RequestPooling() {
		t.Fatal("HTTP_POOL_REQUESTS=true left the request pool off")
	}
}
