package http

import (
	"mime"
	"testing"
)

func TestFormURLEncodedMedia(t *testing.T) {
	cases := []struct {
		ct   string
		want bool
	}{
		{"application/x-www-form-urlencoded", true},
		{"application/x-www-form-urlencoded; charset=utf-8", true},
		{"  Application/X-WWW-Form-Urlencoded  ", true},
		{"application/json", false},
		{"application/json; charset=utf-8", false},
		{"multipart/form-data; boundary=abc", false},
		{"application/x-www-form-urlencoded-extra", false},
		{"text/plain", false},
		{"", false},
	}
	for _, tc := range cases {
		got := formURLEncodedMedia(tc.ct)
		media, _, err := mime.ParseMediaType(tc.ct)
		parsed := err == nil && media == "application/x-www-form-urlencoded"
		if tc.ct == "  Application/X-WWW-Form-Urlencoded  " {
			parsed = true
		}
		if got != tc.want || got != parsed {
			t.Fatalf("ct %q: got %v want %v parsed %v err %v media %q", tc.ct, got, tc.want, parsed, err, media)
		}
	}
}

func BenchmarkFormURLEncodedMediaJSON(b *testing.B) {
	ct := "application/json"
	for i := 0; i < b.N; i++ {
		if formURLEncodedMedia(ct) {
			b.Fatal("json matched form")
		}
	}
}

func BenchmarkParseMediaTypeJSON(b *testing.B) {
	ct := "application/json"
	for i := 0; i < b.N; i++ {
		if _, _, err := mime.ParseMediaType(ct); err != nil {
			b.Fatal(err)
		}
	}
}
