package shorturl_test

import (
	"testing"
	"time"

	"github.com/zatrano/framework/packages/shorturl"
)

func TestShortURL(t *testing.T) {
	m := shorturl.New("http://localhost:8080")
	link, err := m.Create("https://example.com/docs", time.Hour)
	if err != nil || link.Code == "" {
		t.Fatalf("%+v err=%v", link, err)
	}
	got, err := m.Resolve(link.Code)
	if err != nil || got.URL != "https://example.com/docs" || got.Hits != 1 {
		t.Fatalf("%+v err=%v", got, err)
	}
	if m.ShortURL(link.Code) != "http://localhost:8080/s/"+link.Code {
		t.Fatal(m.ShortURL(link.Code))
	}
}
