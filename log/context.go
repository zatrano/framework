package log

import (
	"fmt"
	"strings"
)

// Context is a logger with additional fields.
type Context struct {
	parent *Logger
	fields map[string]any
}

// With returns a contextual logger that includes the given fields.
func (l *Logger) With(fields map[string]any) *Context {
	cp := make(map[string]any, len(fields))
	for k, v := range fields {
		cp[k] = v
	}
	return &Context{parent: l, fields: cp}
}

// Share stores a field included on all subsequent log lines until flushed.
func (l *Logger) Share(key string, value any) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shared == nil {
		l.shared = make(map[string]any)
	}
	l.shared[key] = value
}

// FlushShared clears shared context fields.
func (l *Logger) FlushShared() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.shared = make(map[string]any)
}

// Debug logs with context fields.
func (c *Context) Debug(args ...any) { c.write(Debug, args...) }

// Info logs with context fields.
func (c *Context) Info(args ...any) { c.write(Info, args...) }

// Warning logs with context fields.
func (c *Context) Warning(args ...any) { c.write(Warning, args...) }

// Error logs with context fields.
func (c *Context) Error(args ...any) { c.write(Error, args...) }

// Infof logs a formatted message with context fields.
func (c *Context) Infof(format string, args ...any) {
	c.writef(Info, format, args...)
}

func (c *Context) write(level Level, args ...any) {
	if c == nil || c.parent == nil {
		return
	}
	if prefix := fieldsPrefix(c.fields); prefix != "" {
		args = append([]any{prefix}, args...)
	}
	c.parent.write(level, args...)
}

func (c *Context) writef(level Level, format string, args ...any) {
	if c == nil || c.parent == nil {
		return
	}
	if prefix := fieldsPrefix(c.fields); prefix != "" {
		format = prefix + " " + format
	}
	c.parent.writef(level, format, args...)
}

func fieldsPrefix(fields map[string]any) string {
	if len(fields) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fields))
	for k, v := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return "[" + strings.Join(parts, " ") + "]"
}
