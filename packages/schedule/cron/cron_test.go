package cron_test

import (
	"testing"
	"time"

	"github.com/zatrano/framework/packages/schedule/cron"
)

func TestCronMatches(t *testing.T) {
	expr, err := cron.Parse("*/5 * * * *")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 8, 1, 12, 10, 0, 0, time.UTC)
	if !expr.Matches(at) {
		t.Fatal("expected match at :10")
	}
	if expr.Matches(at.Add(time.Minute)) {
		t.Fatal("should not match :11")
	}
	hourly, _ := cron.Parse("@hourly")
	if !hourly.Matches(time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("@hourly")
	}
	next, ok := expr.Next(at)
	if !ok || next.Minute() != 15 {
		t.Fatalf("next=%v ok=%v", next, ok)
	}
}
