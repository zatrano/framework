package log_test

import (
	"os"
	"path/filepath"
	"testing"

	zlog "github.com/zatrano/framework/v2/kernel/log"
)

func TestLoggerContext(t *testing.T) {
	logger, err := zlog.New("debug", "")
	if err != nil {
		t.Fatal(err)
	}
	logger.Share("request_id", "abc")
	logger.With(map[string]any{"user": "ada"}).Infof("hello %s", "world")
	logger.FlushShared()
	logger.Info("done")
}

func TestLoggerLevelsFileAndContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	logger, err := zlog.New("info", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logger.Close() })
	logger.Debug("hidden")
	logger.Info("shown")
	logger.Warning("w")
	logger.Error("e")
	logger.Debugf("d %s", "x")
	logger.Infof("i %s", "x")
	logger.Warningf("w %s", "x")
	logger.Errorf("e %s", "x")
	logger.Share("k", "v")
	logger.Info("shared")
	ctx := logger.With(map[string]any{"user": "ada"})
	ctx.Debug("d")
	ctx.Info("i")
	ctx.Warning("w")
	ctx.Error("e")
	ctx.Infof("fmt %d", 1)
	var nilLog *zlog.Logger
	nilLog.Share("x", 1)
	nilLog.FlushShared()
	var nilCtx *zlog.Context
	nilCtx.Info("noop")
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 {
		t.Fatalf("log file: %v %q", err, raw)
	}
	stdout, err := zlog.New("warn", "")
	if err != nil {
		t.Fatal(err)
	}
	stdout.Debug("skip")
	stdout.Warning("ok")
	errorLog, err := zlog.New("error", "")
	if err != nil {
		t.Fatal(err)
	}
	errorLog.Warning("skip")
	errorLog.Error("ok")
	nested, err := zlog.New("debug", filepath.Join(t.TempDir(), "nested", "more", "x.log"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nested.Close() })
}
