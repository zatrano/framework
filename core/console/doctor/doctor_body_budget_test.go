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
