package http_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/kernel/http"
)

func TestTrackCommitWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	w := http.TrackCommit(rec)
	if w.Committed() {
		t.Fatal("expected uncommitted")
	}
	w.WriteHeader(stdhttp.StatusCreated)
	if !w.Committed() || rec.Code != stdhttp.StatusCreated {
		t.Fatalf("committed=%v code=%d", w.Committed(), rec.Code)
	}
	w.WriteHeader(stdhttp.StatusInternalServerError)
	if rec.Code != stdhttp.StatusCreated {
		t.Fatalf("second WriteHeader changed status to %d", rec.Code)
	}
}

func TestTrackCommitWriteCommitsOK(t *testing.T) {
	rec := httptest.NewRecorder()
	w := http.TrackCommit(rec)
	if _, err := w.Write([]byte("hi")); err != nil {
		t.Fatal(err)
	}
	if !w.Committed() || rec.Code != stdhttp.StatusOK {
		t.Fatalf("committed=%v code=%d", w.Committed(), rec.Code)
	}
}
