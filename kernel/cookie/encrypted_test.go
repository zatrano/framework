package cookie_test

import (
	crand "crypto/rand"
	"errors"
	"strings"
	"testing"

	"github.com/zatrano/framework/kernel/cookie"
	"github.com/zatrano/framework/kernel/encryption"
)

func TestQueueEncryptedReturnsEncryptError(t *testing.T) {
	enc, err := encryption.New(strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	old := crand.Reader
	t.Cleanup(func() { crand.Reader = old })
	crand.Reader = readerFunc(func([]byte) (int, error) {
		return 0, errors.New("entropy exhausted")
	})
	jar := cookie.NewJar()
	if err := jar.QueueEncrypted(enc, "sid", "secret", 10); err == nil {
		t.Fatal("QueueEncrypted must return encrypt error")
	}
	if len(jar.Apply()) != 0 {
		t.Fatal("failed QueueEncrypted must not queue a cookie")
	}
}

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }
