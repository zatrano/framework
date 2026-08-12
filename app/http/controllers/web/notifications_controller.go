package web

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/core/flash"
	"github.com/zatrano/framework/core/http"
	"github.com/zatrano/framework/core/notification"
)

// NotificationsController serves web notification inbox and send forms.
type NotificationsController struct {
	App *core.Application
}

// Index shows recent in-app notifications.
func (c *NotificationsController) Index(req *http.Request) *http.Response {
	id := firstNonEmpty(req.Query("notifiable_id"), req.Input("notifiable_id"), "demo")
	store := c.App.Notifications().Store()
	var items []notification.Record
	if store != nil {
		items, _ = store.ListFor(id, 50)
	}
	return http.View("notifications/index", map[string]any{
		"notifiableID": id,
		"items":        items,
	})
}

// SendForm shows the single-send form.
func (c *NotificationsController) SendForm(req *http.Request) *http.Response {
	return http.View("notifications/send", map[string]any{})
}

// Send handles single notification submit.
func (c *NotificationsController) Send(req *http.Request) *http.Response {
	channels := splitCSV(firstNonEmpty(req.Input("channels"), "database"))
	recipient := notification.Recipient{
		ID:    req.Input("id"),
		Email: req.Input("email"),
		Phone: req.Input("phone"),
		Name:  req.Input("name"),
	}
	msg := notification.Message{
		Channels: channels,
		Subject:  req.Input("subject"),
		Body:     req.Input("body"),
	}
	if err := c.App.Notifications().Send(recipient, msg); err != nil {
		flash.Error(req, err.Error())
		return http.Redirect("/notifications/send")
	}
	flash.Success(req, "Notification sent.")
	id := fmt.Sprint(recipient.NotificationID())
	return http.Redirect("/notifications?notifiable_id=" + id)
}

// BulkForm shows bulk import form.
func (c *NotificationsController) BulkForm(req *http.Request) *http.Response {
	return http.View("notifications/bulk", map[string]any{})
}

// Bulk handles CSV/XLSX import send.
func (c *NotificationsController) Bulk(req *http.Request) *http.Response {
	uploaded, err := req.File("file")
	if err != nil {
		flash.Error(req, "CSV or XLSX file is required.")
		return http.Redirect("/notifications/bulk")
	}
	src, err := uploaded.File()
	if err != nil {
		flash.Error(req, err.Error())
		return http.Redirect("/notifications/bulk")
	}
	defer src.Close()
	raw, err := io.ReadAll(src)
	if err != nil {
		flash.Error(req, err.Error())
		return http.Redirect("/notifications/bulk")
	}
	recipients, err := notification.ImportRecipientsBytes(uploaded.Name(), raw)
	if err != nil {
		flash.Error(req, err.Error())
		return http.Redirect("/notifications/bulk")
	}
	channels := splitCSV(firstNonEmpty(req.Input("channels"), "database"))
	msg := notification.Message{
		Channels: channels,
		Subject:  req.Input("subject"),
		Body:     req.Input("body"),
	}
	result := c.App.Notifications().SendMany(recipients, msg)
	flash.Success(req, "Bulk send completed: "+strconv.Itoa(result.Sent)+"/"+strconv.Itoa(result.Total))
	return http.Redirect("/notifications")
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
