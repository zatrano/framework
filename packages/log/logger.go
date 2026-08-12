package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Level represents a log severity.
type Level int

const (
	Debug Level = iota
	Info
	Warning
	Error
)

// Logger writes structured-ish log lines.
type Logger struct {
	mu     sync.Mutex
	level  Level
	logger *log.Logger
	file   *os.File
	shared map[string]any
}

// New creates a logger that writes to stdout and an optional file.
func New(level string, filePath string) (*Logger, error) {
	writers := []io.Writer{os.Stdout}
	var file *os.File
	if filePath != "" {
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			return nil, err
		}
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		file = f
		writers = append(writers, f)
	}

	return &Logger{
		level:  parseLevel(level),
		logger: log.New(io.MultiWriter(writers...), "", log.LstdFlags),
		file:   file,
	}, nil
}

// Close closes the log file if present.
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Debug logs a debug message.
func (l *Logger) Debug(args ...any) { l.write(Debug, args...) }

// Info logs an info message.
func (l *Logger) Info(args ...any) { l.write(Info, args...) }

// Warning logs a warning message.
func (l *Logger) Warning(args ...any) { l.write(Warning, args...) }

// Error logs an error message.
func (l *Logger) Error(args ...any) { l.write(Error, args...) }

// Debugf logs a formatted debug message.
func (l *Logger) Debugf(format string, args ...any) { l.writef(Debug, format, args...) }

// Infof logs a formatted info message.
func (l *Logger) Infof(format string, args ...any) { l.writef(Info, format, args...) }

// Warningf logs a formatted warning message.
func (l *Logger) Warningf(format string, args ...any) { l.writef(Warning, format, args...) }

// Errorf logs a formatted error message.
func (l *Logger) Errorf(format string, args ...any) { l.writef(Error, format, args...) }

func (l *Logger) write(level Level, args ...any) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	prefix := sharedPrefix(l.shared)
	if prefix != "" {
		args = append([]any{prefix}, args...)
	}
	l.logger.Println(append([]any{levelPrefix(level)}, args...)...)
}

func (l *Logger) writef(level Level, format string, args ...any) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	prefix := sharedPrefix(l.shared)
	msg := fmt.Sprintf(format, args...)
	if prefix != "" {
		l.logger.Println(levelPrefix(level), prefix, msg)
		return
	}
	l.logger.Println(levelPrefix(level), msg)
}

func sharedPrefix(shared map[string]any) string {
	if len(shared) == 0 {
		return ""
	}
	parts := make([]string, 0, len(shared))
	for k, v := range shared {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func parseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return Debug
	case "info":
		return Info
	case "warning", "warn":
		return Warning
	case "error":
		return Error
	default:
		return Debug
	}
}

func levelPrefix(level Level) string {
	switch level {
	case Debug:
		return "[DEBUG]"
	case Info:
		return "[INFO]"
	case Warning:
		return "[WARN]"
	case Error:
		return "[ERROR]"
	default:
		return "[LOG]"
	}
}
