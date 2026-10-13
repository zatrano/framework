package http_test

import (
	"encoding/json"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
)

func TestSetJSONCodecIsOptIn(t *testing.T) {
	sample := map[string]string{"html": "<tag>&"}
	want, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	if string(http.JSON(sample).Content()) != string(want) {
		t.Fatal("default codec drifted from encoding/json")
	}
	http.SetJSONCodec(
		func(any) ([]byte, error) { return []byte(`{"codec":1}`), nil },
		func([]byte, any) error { return nil },
	)
	t.Cleanup(func() { http.SetJSONCodec(nil, nil) })
	if string(http.JSON(sample).Content()) != `{"codec":1}` {
		t.Fatalf("codec not used: %s", http.JSON(sample).Content())
	}
	http.SetJSONCodec(nil, nil)
	if string(http.JSON(sample).Content()) != string(want) {
		t.Fatal("clearing the codec did not restore encoding/json")
	}
}
