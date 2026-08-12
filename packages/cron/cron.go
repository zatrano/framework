package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Expression is a 5-field cron schedule (min hour dom month dow).
type Expression struct {
	raw    string
	fields [5]string
}

// Parse parses a cron expression or alias (@hourly, @daily, @weekly, @monthly, @yearly, @reboot skipped).
func Parse(expr string) (*Expression, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("cron: empty expression")
	}
	switch strings.ToLower(expr) {
	case "@yearly", "@annually":
		expr = "0 0 1 1 *"
	case "@monthly":
		expr = "0 0 1 * *"
	case "@weekly":
		expr = "0 0 * * 0"
	case "@daily", "@midnight":
		expr = "0 0 * * *"
	case "@hourly":
		expr = "0 * * * *"
	case "@every_minute":
		expr = "* * * * *"
	}
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("cron: expected 5 fields, got %d", len(parts))
	}
	e := &Expression{raw: expr}
	for i := 0; i < 5; i++ {
		e.fields[i] = parts[i]
	}
	return e, nil
}

// String returns the normalized expression.
func (e *Expression) String() string {
	if e == nil {
		return ""
	}
	return e.raw
}

// Matches reports whether t satisfies the expression.
func (e *Expression) Matches(t time.Time) bool {
	if e == nil {
		return false
	}
	return matchField(e.fields[0], t.Minute(), 0, 59) &&
		matchField(e.fields[1], t.Hour(), 0, 23) &&
		matchField(e.fields[2], t.Day(), 1, 31) &&
		matchField(e.fields[3], int(t.Month()), 1, 12) &&
		matchField(e.fields[4], int(t.Weekday()), 0, 6)
}

// Next returns the next matching time after from (minute resolution, max scan).
func (e *Expression) Next(from time.Time, limit ...int) (time.Time, bool) {
	max := 366 * 24 * 60
	if len(limit) > 0 && limit[0] > 0 {
		max = limit[0]
	}
	t := from.Truncate(time.Minute).Add(time.Minute)
	for i := 0; i < max; i++ {
		if e.Matches(t) {
			return t, true
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, false
}

func matchField(field string, value, min, max int) bool {
	if field == "*" {
		return true
	}
	if strings.Contains(field, ",") {
		for _, part := range strings.Split(field, ",") {
			if matchField(part, value, min, max) {
				return true
			}
		}
		return false
	}
	if strings.Contains(field, "/") {
		parts := strings.SplitN(field, "/", 2)
		base := parts[0]
		step, _ := strconv.Atoi(parts[1])
		if step <= 0 {
			step = 1
		}
		if base == "*" {
			return (value-min)%step == 0
		}
		if strings.Contains(base, "-") {
			rangeParts := strings.SplitN(base, "-", 2)
			start, _ := strconv.Atoi(rangeParts[0])
			end, _ := strconv.Atoi(rangeParts[1])
			if value < start || value > end {
				return false
			}
			return (value-start)%step == 0
		}
		start, _ := strconv.Atoi(base)
		return value >= start && (value-start)%step == 0
	}
	if strings.Contains(field, "-") {
		parts := strings.SplitN(field, "-", 2)
		start, _ := strconv.Atoi(parts[0])
		end, _ := strconv.Atoi(parts[1])
		return value >= start && value <= end
	}
	n, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return n == value
}
