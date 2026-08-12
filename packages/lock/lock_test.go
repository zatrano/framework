package lock_test

import (
	"testing"
	"time"

	"github.com/zatrano/framework/packages/lock"
)

func TestLockAcquireRelease(t *testing.T) {
	m := lock.New()
	a := m.Get("job", time.Second)
	b := m.Get("job", time.Second)
	if !a.Acquire() {
		t.Fatal("a should acquire")
	}
	if b.Acquire() {
		t.Fatal("b should fail")
	}
	if !a.Release() {
		t.Fatal("release")
	}
	if !b.Acquire() {
		t.Fatal("b should acquire after release")
	}
	_ = b.Release()
}

func TestLockRun(t *testing.T) {
	m := lock.New()
	err := m.Run("demo", time.Second, func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}
