package httpclient_test

import (
	"testing"

	"github.com/zatrano/framework/packages/httpclient"
)

func TestFakeClient(t *testing.T) {
	client := httpclient.Fake(httpclient.FakeResponse{
		Status: 200,
		Body:   `{"hello":"world"}`,
	})
	resp, err := client.AsJSON().Get("https://example.test/api")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK() || resp.String() != `{"hello":"world"}` {
		t.Fatalf("unexpected %#v", resp)
	}
}
