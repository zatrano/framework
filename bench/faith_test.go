package bench

import (
	"bytes"
	"testing"
	"time"
)

func TestFaithDeadlineForwards(t *testing.T) {
	c, err := newFaith(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.SetReadDeadline(time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := c.SetWriteDeadline(time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := c.SetDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}
	n, err := c.Write([]byte("mem"))
	if err != nil || n != 3 {
		t.Fatalf("write n=%d err=%v", n, err)
	}
}
