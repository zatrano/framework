package broadcasting_test

import (
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/broadcasting"
	"github.com/zatrano/framework/packages/http"
)

func TestChannelAuthorization(t *testing.T) {
	mgr := broadcasting.NewManager("null", map[string]broadcasting.Broadcaster{
		"null": broadcasting.NullBroadcaster{},
	})
	mgr.Channel("public", func(req *http.Request, channel string) bool { return true })
	mgr.Channel("private.*", func(req *http.Request, channel string) bool { return false })

	req := http.NewRequest(httptest.NewRequest("POST", "/broadcasting/auth", nil))
	ok, err := mgr.Channels().Authorize(req, "public")
	if err != nil || !ok {
		t.Fatalf("public auth failed ok=%v err=%v", ok, err)
	}
	ok, err = mgr.Channels().Authorize(req, "private.user.1")
	if err != nil || ok {
		t.Fatalf("private should deny ok=%v err=%v", ok, err)
	}
}
