package api

import (
	"io"
	"strconv"
	"strings"

	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/core/http"
	"github.com/zatrano/framework/core/notification"
)

// NotificationsController exposes central notification APIs.
type NotificationsController struct {
	App *core.Application
}

// Send delivers a single notification across selected channels.
// POST /api/notifications/send
func (c *NotificationsController) Send(req *http.Request) *http.Response {
	channels := splitCSV(firstNonEmpty(req.Input("channels"), "database"))
	recipient := notification.Recipient{
		ID:    req.Input("id"),
		Email: req.Input("email"),
		Phone: req.Input("phone"),
		Name:  req.Input("name"),
		Push:  req.Input("push"),
	}
	if recipient.ID == "" && recipient.Email == "" && recipient.Phone == "" {
		return http.JSON(map[string]any{"message": "id, email, or phone is required"}).Status(422)
	}
	msg := notification.Message{
		Channels: channels,
		Subject:  req.Input("subject"),
		Body:     req.Input("body"),
	}
	if err := c.App.Notifications().Send(recipient, msg); err != nil {
		return http.JSON(map[string]any{"message": err.Error()}).Status(500)
	}
	return http.JSON(map[string]any{"status": "sent", "channels": channels})
}

// Bulk sends notifications to recipients imported from CSV/XLSX.
// POST /api/notifications/bulk (multipart: file, channels, subject, body)
func (c *NotificationsController) Bulk(req *http.Request) *http.Response {
	uploaded, err := req.File("file")
	if err != nil {
		return http.JSON(map[string]any{"message": "file is required (.csv or .xlsx)"}).Status(422)
	}
	src, err := uploaded.File()
	if err != nil {
		return http.JSON(map[string]any{"message": err.Error()}).Status(500)
	}
	defer src.Close()
	raw, err := io.ReadAll(src)
	if err != nil {
		return http.JSON(map[string]any{"message": err.Error()}).Status(500)
	}
	recipients, err := notification.ImportRecipientsBytes(uploaded.Name(), raw)
	if err != nil {
		return http.JSON(map[string]any{"message": err.Error()}).Status(422)
	}
	channels := splitCSV(firstNonEmpty(req.Input("channels"), "database"))
	msg := notification.Message{
		Channels: channels,
		Subject:  req.Input("subject"),
		Body:     req.Input("body"),
	}
	result := c.App.Notifications().SendMany(recipients, msg)
	return http.JSON(map[string]any{
		"status":   "completed",
		"channels": channels,
		"result":   result,
	})
}

// Index lists in-app notifications for a notifiable.
// GET /api/notifications?notifiable_id=&unread=1
func (c *NotificationsController) Index(req *http.Request) *http.Response {
	store := c.App.Notifications().Store()
	if store == nil {
		return http.JSON(map[string]any{"message": "notification store is unavailable"}).Status(503)
	}
	id := firstNonEmpty(req.Query("notifiable_id"), req.Input("notifiable_id"))
	if id == "" {
		return http.JSON(map[string]any{"message": "notifiable_id is required"}).Status(422)
	}
	limit, _ := strconv.Atoi(req.Query("limit", "50"))
	var (
		items []notification.Record
		err   error
	)
	if req.Query("unread") == "1" || req.Query("unread") == "true" {
		items, err = store.UnreadFor(id, limit)
	} else {
		items, err = store.ListFor(id, limit)
	}
	if err != nil {
		return http.JSON(map[string]any{"message": err.Error()}).Status(500)
	}
	return http.JSON(map[string]any{"data": items})
}

// MarkRead marks one notification as read.
// POST /api/notifications/{id}/read
func (c *NotificationsController) MarkRead(req *http.Request) *http.Response {
	store := c.App.Notifications().Store()
	if store == nil {
		return http.JSON(map[string]any{"message": "notification store is unavailable"}).Status(503)
	}
	id, _ := strconv.ParseInt(req.RouteParams()["id"], 10, 64)
	notifiableID := firstNonEmpty(req.Input("notifiable_id"), req.Query("notifiable_id"))
	if id == 0 || notifiableID == "" {
		return http.JSON(map[string]any{"message": "id and notifiable_id are required"}).Status(422)
	}
	if err := store.MarkAsRead(id, notifiableID); err != nil {
		return http.JSON(map[string]any{"message": err.Error()}).Status(500)
	}
	return http.JSON(map[string]any{"status": "read"})
}

// MarkAllRead marks all notifications as read for a notifiable.
// POST /api/notifications/read-all
func (c *NotificationsController) MarkAllRead(req *http.Request) *http.Response {
	store := c.App.Notifications().Store()
	if store == nil {
		return http.JSON(map[string]any{"message": "notification store is unavailable"}).Status(503)
	}
	notifiableID := req.Input("notifiable_id")
	if notifiableID == "" {
		return http.JSON(map[string]any{"message": "notifiable_id is required"}).Status(422)
	}
	if err := store.MarkAllRead(notifiableID); err != nil {
		return http.JSON(map[string]any{"message": err.Error()}).Status(500)
	}
	return http.JSON(map[string]any{"status": "read"})
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
