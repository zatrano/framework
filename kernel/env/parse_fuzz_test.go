package env_test

import (
	"testing"

	"github.com/zatrano/framework/v2/kernel/env"
)

func FuzzParse(f *testing.F) {
	f.Add([]byte("FOO=bar\n"))
	f.Add([]byte("# c\nexport A=b\n"))
	f.Add([]byte("A=\"x\\ny\"\n"))
	f.Add([]byte("A='q'\nB=$A\n"))
	f.Add([]byte("FOO=\"unterminated"))
	f.Add([]byte("NOEQ\n"))
	f.Add([]byte("\xef\xbb\xbfFOO=bar\r\n"))
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
