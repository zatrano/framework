package http_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

func bindRequest(payload string) string {
	return fmt.Sprintf(
		"POST /echo HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		len(payload), payload,
	)
}

func TestBindJSONDoesNotAliasDecodedStrings(t *testing.T) {
	const payload = `{"message":"Hello, World!"}`
	var decoded map[string]any
	status, body, err := http.ServeConnForTest(func(ctx *rawhttp.Ctx) {
		req := http.NewRequest(ctx)
		if err := req.BindJSON(&decoded); err != nil {
			t.Errorf("BindJSON: %v", err)
			return
		}
		raw := ctx.Body()
		if len(raw) == 0 {
			t.Error("empty connection buffer")
			return
		}
		owned, err := req.Body()
		if err != nil {
			t.Errorf("Body: %v", err)
			return
		}
		owned[0] = '!'
		if raw[0] == '!' {
			t.Error("Body aliases the connection buffer")
		}
		raw[0] = 'X'
		if decoded["message"] != "Hello, World!" {
			t.Errorf("decoded string aliases the connection buffer: %#v", decoded)
		}
		ctx.SetStatusCode(200)
		ctx.SetBodyString("ok")
	}, bindRequest(payload))
	if err != nil {
		t.Fatal(err)
	}
	if status != 200 || string(body) != "ok" {
		t.Fatalf("status=%d body=%q", status, body)
	}
	if decoded["message"] != "Hello, World!" {
		t.Fatalf("decoded %#v", decoded)
	}
}

func TestBindJSONRejectsOversizedBody(t *testing.T) {
	payload := `{"message":"Hello, World!"}`
	status, _, err := http.ServeConnForTest(func(ctx *rawhttp.Ctx) {
		req := http.NewRequest(ctx)
		req.SetMaxBodyBytes(4)
		var m map[string]any
		if err := req.BindJSON(&m); err != http.ErrBodyTooLarge {
			t.Errorf("err %v", err)
		}
		if _, err := req.Body(); err != http.ErrBodyTooLarge {
			t.Errorf("Body err %v", err)
		}
		ctx.SetStatusCode(413)
		ctx.SetBodyString("big")
	}, bindRequest(payload))
	if err != nil {
		t.Fatal(err)
	}
	if status != 413 {
		t.Fatalf("status %d", status)
	}
}

func BenchmarkUnmarshalInPlace(b *testing.B) {
	raw := []byte(`{"message":"Hello, World!"}`)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCopyThenUnmarshal(b *testing.B) {
	raw := []byte(`{"message":"Hello, World!"}`)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		owned := append([]byte(nil), raw...)
		var m map[string]any
		if err := json.Unmarshal(owned, &m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBindJSON(b *testing.B) {
	payload := `{"message":"Hello, World!"}`
	raw := bindRequest(payload)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := http.ServeConnForTest(func(ctx *rawhttp.Ctx) {
			req := http.NewRequest(ctx)
			var m map[string]any
			if err := req.BindJSON(&m); err != nil {
				b.Fatal(err)
			}
			ctx.SetStatusCode(200)
			ctx.SetBodyString("ok")
		}, raw)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONCopiesBody(b *testing.B) {
	payload := `{"message":"Hello, World!"}`
	raw := bindRequest(payload)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := http.ServeConnForTest(func(ctx *rawhttp.Ctx) {
			req := http.NewRequest(ctx)
			var m map[string]any
			if err := req.JSON(&m); err != nil {
				b.Fatal(err)
			}
			ctx.SetStatusCode(200)
			ctx.SetBodyString("ok")
		}, raw)
		if err != nil {
			b.Fatal(err)
		}
	}
}
