package http

import "testing"

func TestCloneStaysValidAfterRelease(t *testing.T) {
	ConfigureRequestPool(false)
	req := testRequest("POST", "/docs?q=ab", []byte("name=Ada"))
	req.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	req.SetHeader("X-Trace", "abc")
	req.Set("k", "v")
	clone := req.Clone()
	req.ctx.Path[1] = 'X'
	ReleaseRequest(req)

	if requestFreedPoison {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("original did not panic")
				}
			}()
			_ = req.Path()
		}()
	}
	if clone.Path() != "/docs" {
		t.Fatalf("path=%q", clone.Path())
	}
	if clone.Query("q") != "ab" {
		t.Fatalf("query=%q", clone.Query("q"))
	}
	value, ok := clone.HeaderValue("X-Trace")
	if !ok || value != "abc" {
		t.Fatalf("header=%q ok=%v", value, ok)
	}
	body, err := clone.Body()
	if err != nil || string(body) != "name=Ada" {
		t.Fatalf("body=%q err=%v", body, err)
	}
	if clone.Get("k") != "v" {
		t.Fatalf("attr=%v", clone.Get("k"))
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if clone.Path() != "/docs" || clone.QueryString() != "q=ab" {
			t.Errorf("async clone path=%q query=%q", clone.Path(), clone.QueryString())
		}
	}()
	<-done
}
