package env_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel/env"
)

func TestParseCommentsQuotesExportEmpty(t *testing.T) {
	src := `
# comment
export APP_NAME=ZATRANO
APP_KEY=
APP_DEBUG = true
GREETING="hello # world"
SINGLE='keep $HOME raw'
SPACED=  value  
INLINE=bar  # trailing
`
	got, err := env.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if got["APP_NAME"] != "ZATRANO" {
		t.Fatalf("APP_NAME=%q", got["APP_NAME"])
	}
	if got["APP_KEY"] != "" {
		t.Fatalf("APP_KEY=%q", got["APP_KEY"])
	}
	if got["APP_DEBUG"] != "true" {
		t.Fatalf("APP_DEBUG=%q", got["APP_DEBUG"])
	}
	if got["GREETING"] != "hello # world" {
		t.Fatalf("GREETING=%q", got["GREETING"])
	}
	if got["SINGLE"] != "keep $HOME raw" {
		t.Fatalf("SINGLE=%q", got["SINGLE"])
	}
	if got["SPACED"] != "value" {
		t.Fatalf("SPACED=%q", got["SPACED"])
	}
	if got["INLINE"] != "bar" {
		t.Fatalf("INLINE=%q", got["INLINE"])
	}
}

func TestParseCRLFAndBOM(t *testing.T) {
	src := "\xef\xbb\xbfFOO=one\r\nBAR=two\rBAZ=three\n"
	got, err := env.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if got["FOO"] != "one" || got["BAR"] != "two" || got["BAZ"] != "three" {
		t.Fatalf("%v", got)
	}
}

func TestParseEscapesAndMultiline(t *testing.T) {
	src := "A=\"line\\nnext\"\nB=\"say \\\"hi\\\"\"\nC=\"one\ntwo\"\n"
	got, err := env.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if got["A"] != "line\nnext" {
		t.Fatalf("A=%q", got["A"])
	}
	if got["B"] != `say "hi"` {
		t.Fatalf("B=%q", got["B"])
	}
	if got["C"] != "one\ntwo" {
		t.Fatalf("C=%q", got["C"])
	}
}

func TestParseExpansionFileThenProcess(t *testing.T) {
	t.Setenv("ZATRANO_ENV_OS", "from-os")
	src := `
HOST=localhost
URL=http://${HOST}:8080
OSVAL=$ZATRANO_ENV_OS
MISSING=$ZATRANO_ENV_DOES_NOT_EXIST
`
	got, err := env.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if got["URL"] != "http://localhost:8080" {
		t.Fatalf("URL=%q", got["URL"])
	}
	if got["OSVAL"] != "from-os" {
		t.Fatalf("OSVAL=%q", got["OSVAL"])
	}
	if got["MISSING"] != "" {
		t.Fatalf("MISSING=%q", got["MISSING"])
	}
}

func TestParseExpansionDoesNotExplode(t *testing.T) {
	got, err := env.Parse([]byte("X=$X$X\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got["X"]) > 64<<10 {
		t.Fatalf("expansion grew to %d bytes", len(got["X"]))
	}
}

func TestParseMalformed(t *testing.T) {
	if _, err := env.Parse([]byte("NOEQUALS\n")); err == nil {
		t.Fatal("expected error")
	}
	if _, err := env.Parse([]byte("FOO=\"unterminated\n")); err == nil {
		t.Fatal("expected unterminated quote")
	}
}

func TestParseExampleFile(t *testing.T) {
	_, this, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	path := filepath.Join(filepath.Dir(this), "..", "..", ".env.example")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := env.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got["APP_NAME"] != "ZATRANO" {
		t.Fatalf("APP_NAME=%q", got["APP_NAME"])
	}
	if !strings.Contains(string(data), "APP_KEY=") {
		t.Fatal("example")
	}
	if _, ok := got["APP_KEY"]; !ok {
		t.Fatal("empty APP_KEY must still be present")
	}
	for _, key := range []string{
		"SESSION_DRIVER", "CACHE_STORE", "QUEUE_CONNECTION", "REDIS_HOST",
		"MAIL_MAILER", "STRIPE_SECRET_KEY", "AI_DRIVER", "MONGO_URI",
		"LOG_CHANNEL", "DB_CONNECTION",
	} {
		if _, ok := got[key]; ok {
			t.Fatalf("framework .env.example must be kernel-only, found %s", key)
		}
	}
}
