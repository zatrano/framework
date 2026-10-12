package http

import "testing"

func TestJSONKeepsSecurityHeadersInline(t *testing.T) {
	resp := JSON(map[string]string{"message": "Hello, World!"})
	if resp.headers != nil {
		t.Fatal("JSON must not allocate a header map")
	}
	resp.Header("X-Frame-Options", "SAMEORIGIN")
	resp.Header("X-Content-Type-Options", "nosniff")
	resp.Header("Referrer-Policy", "strict-origin-when-cross-origin")
	resp.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
	resp.Header("X-Request-Id", "0123456789abcdef0123456789abcdef")
	if resp.headers != nil {
		t.Fatal("five headers must stay in the inline slots")
	}
	if resp.nInline != 5 {
		t.Fatalf("inline headers = %d", resp.nInline)
	}
	if resp.GetHeader("X-Frame-Options") != "SAMEORIGIN" {
		t.Fatalf("frame header = %q", resp.GetHeader("X-Frame-Options"))
	}
	if resp.headers != nil {
		t.Fatal("GetHeader must not flush inline headers into a map")
	}
}

func BenchmarkJSONWithSecurityHeaders(b *testing.B) {
	payload := map[string]string{"message": "Hello, World!"}
	const id = "0123456789abcdef0123456789abcdef"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resp := JSON(payload)
		resp.Header("X-Frame-Options", "SAMEORIGIN")
		resp.Header("X-Content-Type-Options", "nosniff")
		resp.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		resp.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		resp.Header("X-Request-Id", id)
		if resp.nInline != 5 {
			b.Fatal(resp.nInline)
		}
	}
}
