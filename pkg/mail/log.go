package mail

import (
	"context"
	"go.uber.org/zap"
)

// LogDriver is a mail driver that logs messages instead of sending them.
// Useful for development and testing.
type LogDriver struct {
	logger *zap.Logger
}

// NewLogDriver creates a log-only mail driver.
func NewLogDriver(logger *zap.Logger) *LogDriver {
	return &LogDriver{logger: logger}
}

var _ Driver = (*LogDriver)(nil)

func (d *LogDriver) Name() string { return "log" }

func (d *LogDriver) Send(_ context.Context, msg *Message) error {
	d.logger.Info("mail (log driver)",
		zap.String("from", msg.From.String()),
		zap.String("to", joinAddrs(msg.To)),
		zap.String("subject", msg.Subject),
		zap.Int("attachments", len(msg.Attachments)),
		zap.Bool("has_html", msg.HTMLBody != ""),
		zap.Bool("has_text", msg.TextBody != ""),
	)
	return nil
}
