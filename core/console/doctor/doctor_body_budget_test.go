package doctor

import (
	"path/filepath"
	"testing"
)

func TestDoctorBodyLimitOverBudget(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "routes.go"), `package http

func routes(r interface{ BodyLimit(int64) }) {
	r.BodyLimit(300 << 20)
}
`)
	assertDoctorRule(t, root, "APP-HTTP-006")
}

func TestDoctorTrustedProxyShareHint(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "http", "routes.go"), "package http\n")
	writeDoctorFile(t, root, ".env", "APP_KEY=test\n")
	assertDoctorRule(t, root, "APP-HTTP-007")

	set := t.TempDir()
	writeDoctorFile(t, set, filepath.Join("app", "http", "routes.go"), "package http\n")
	writeDoctorFile(t, set, ".env", "TRUSTED_PROXIES=10.0.0.1\n")
	if hasDoctorRule(mustDoctor(t, set), "APP-HTTP-007") {
		t.Fatal("TRUSTED_PROXIES must silence APP-HTTP-007")
	}
	off := t.TempDir()
	writeDoctorFile(t, off, filepath.Join("app", "http", "routes.go"), "package http\n")
	writeDoctorFile(t, off, ".env", "HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT=-1\n")
	if hasDoctorRule(mustDoctor(t, off), "APP-HTTP-007") {
		t.Fatal("a negative per-client share must silence APP-HTTP-007")
	}
}
