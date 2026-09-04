package log_test

import (
	"testing"

	zlog "github.com/zatrano/framework/kernel/log"
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
