package env

import (
	"strings"
	"testing"
)

func TestIntOrBlankUsesFallback(t *testing.T) {
	t.Setenv("APP_PORT", "  ")
	n, err := IntOr("APP_PORT", 8080)
	if err != nil || n != 8080 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestIntOrParsesInteger(t *testing.T) {
	t.Setenv("APP_PORT", "3000")
	n, err := IntOr("APP_PORT", 8080)
	if err != nil || n != 3000 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestIntOrRejectsNonInteger(t *testing.T) {
	t.Setenv("APP_PORT", "abc")
	_, err := IntOr("APP_PORT", 8080)
	if err == nil {
		t.Fatal("expected configuration error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "APP_PORT") || !strings.Contains(msg, "integer") || !strings.Contains(msg, "abc") {
		t.Fatalf("want named type error, got %v", err)
	}
}

func TestIntOrDoesNotEchoSecrets(t *testing.T) {
	t.Setenv("DB_PASSWORD", "hunter2-not-an-int")
	_, err := IntOr("DB_PASSWORD", 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "hunter2") {
		t.Fatalf("secret leaked: %v", err)
	}
	if !strings.Contains(err.Error(), "DB_PASSWORD") || !strings.Contains(err.Error(), "integer") {
		t.Fatalf("must still name the variable: %v", err)
	}
}

func TestSensitiveDetectsCredentialKeys(t *testing.T) {
	for _, key := range []string{"DB_PASSWORD", "API_TOKEN", "APP_KEY", "Authorization"} {
		if !Sensitive(key) {
			t.Errorf("%s should be sensitive", key)
		}
	}
	if Sensitive("APP_PORT") || Sensitive("APP_ENV") {
		t.Fatal("APP_PORT/APP_ENV are not credentials")
	}
}
