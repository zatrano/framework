package fuzz_test

import (
	"testing"

	"github.com/zatrano/framework/kernel/env"
)

func FuzzDotenvParse(f *testing.F) {
	f.Add([]byte("FOO=bar\n# c\nexport A=b\n"))
	f.Add([]byte("A=\"x\\ny\"\nB='q'\nC=$A\n"))
	f.Add([]byte("FOO=\"unterminated"))
	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("panic: %v", rec)
			}
		}()
		if len(data) > 8192 {
			return
		}
		_, _ = env.Parse(data)
	})
}
