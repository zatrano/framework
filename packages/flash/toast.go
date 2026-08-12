package flash

import (
	"html"
	"strings"

	"github.com/zatrano/framework/packages/http"
)

const KeyToast = "flash_toasts"

// Toast is a UI toast message.
type Toast struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// PushToast queues a toast for the next request.
func PushToast(req *http.Request, typ, message string) {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" {
		typ = "info"
	}
	message = strings.TrimSpace(message)
	if message == "" || req == nil {
		return
	}
	sess := req.Session()
	if sess == nil {
		return
	}
	list := pullToastList(sess.Get(KeyToast))
	list = append(list, Toast{Type: typ, Message: message})
	sess.Flash(KeyToast, list)
}

// ToastSuccess queues a success toast.
func ToastSuccess(req *http.Request, message string) { PushToast(req, "success", message) }

// ToastError queues an error toast.
func ToastError(req *http.Request, message string) { PushToast(req, "error", message) }

// ToastInfo queues an info toast.
func ToastInfo(req *http.Request, message string) { PushToast(req, "info", message) }

// Toasts returns toasts for the current request (from flash).
func Toasts(req *http.Request) []Toast {
	if req == nil || req.Session() == nil {
		return nil
	}
	return pullToastList(req.Session().Get(KeyToast))
}

// RenderToasts returns a minimal HTML snippet for flashed toasts.
func RenderToasts(req *http.Request) string {
	items := Toasts(req)
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="zatrano-toasts" role="status">`)
	for _, t := range items {
		b.WriteString(`<div class="zatrano-toast zatrano-toast--`)
		b.WriteString(html.EscapeString(t.Type))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(t.Message))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func pullToastList(raw any) []Toast {
	switch v := raw.(type) {
	case []Toast:
		out := make([]Toast, len(v))
		copy(out, v)
		return out
	case []any:
		out := make([]Toast, 0, len(v))
		for _, item := range v {
			if t, ok := item.(Toast); ok {
				out = append(out, t)
			}
		}
		return out
	default:
		return nil
	}
}
