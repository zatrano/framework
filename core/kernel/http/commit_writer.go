package http

import (
	stdhttp "net/http"
)

// CommitWriter tracks whether headers or body have been written so panic
// recovery can avoid rewriting a committed response as 500.
type CommitWriter struct {
	stdhttp.ResponseWriter
	committed bool
}

// TrackCommit wraps w. If w is already a *CommitWriter, it is returned as-is.
func TrackCommit(w stdhttp.ResponseWriter) *CommitWriter {
	if cw, ok := w.(*CommitWriter); ok {
		return cw
	}
	return &CommitWriter{ResponseWriter: w}
}

// Committed reports whether WriteHeader or Write has been invoked.
func (w *CommitWriter) Committed() bool {
	return w != nil && w.committed
}

// Unwrap returns the underlying writer (http.ResponseController).
func (w *CommitWriter) Unwrap() stdhttp.ResponseWriter {
	if w == nil {
		return nil
	}
	return w.ResponseWriter
}

func (w *CommitWriter) WriteHeader(code int) {
	if w == nil || w.ResponseWriter == nil {
		return
	}
	if w.committed {
		return
	}
	w.committed = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *CommitWriter) Write(p []byte) (int, error) {
	if w == nil || w.ResponseWriter == nil {
		return 0, nil
	}
	if !w.committed {
		w.WriteHeader(stdhttp.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func (w *CommitWriter) Flush() {
	if w == nil {
		return
	}
	if !w.committed {
		w.WriteHeader(stdhttp.StatusOK)
	}
	if f, ok := w.ResponseWriter.(stdhttp.Flusher); ok {
		f.Flush()
	}
}
