package notification

import (
	"fmt"
	"strings"
)

// Recipient is a transport-agnostic notification target used by bulk/central sends.
type Recipient struct {
	ID    string
	Email string
	Phone string
	Name  string
	Push  string
	Meta  map[string]string
}

// RouteNotificationFor implements Notifiable.
func (r Recipient) RouteNotificationFor(channel string) string {
	switch strings.ToLower(channel) {
	case "mail":
		return r.Email
	case "sms":
		return r.Phone
	case "push":
		if r.Push != "" {
			return r.Push
		}
		if r.ID != "" {
			return "user:" + r.ID
		}
		return ""
	case "broadcast", "database":
		if r.ID != "" {
			return r.ID
		}
		if r.Email != "" {
			return r.Email
		}
		return r.Phone
	default:
		return ""
	}
}

// NotificationID implements Notifiable.
func (r Recipient) NotificationID() any {
	if r.ID != "" {
		return r.ID
	}
	if r.Email != "" {
		return r.Email
	}
	if r.Phone != "" {
		return r.Phone
	}
	return "anonymous"
}

// NotifiableType returns a stable type label for polymorphic storage.
func (r Recipient) NotifiableType() string {
	if t := r.Meta["notifiable_type"]; t != "" {
		return t
	}
	return "recipient"
}

// RecipientFromMap builds a Recipient from imported row keys.
// Accepted keys (case-insensitive): id, email, phone/sms/mobile, name, push/token.
func RecipientFromMap(row map[string]string) (Recipient, error) {
	normalized := map[string]string{}
	for k, v := range row {
		key := strings.ToLower(strings.TrimSpace(k))
		normalized[key] = strings.TrimSpace(v)
	}
	r := Recipient{
		ID:    first(normalized, "id", "user_id", "notifiable_id"),
		Email: first(normalized, "email", "mail", "e-mail"),
		Phone: first(normalized, "phone", "sms", "mobile", "tel"),
		Name:  first(normalized, "name", "full_name", "fullname"),
		Push:  first(normalized, "push", "token", "device_token"),
		Meta:  row,
	}
	if r.ID == "" && r.Email == "" && r.Phone == "" {
		return Recipient{}, fmt.Errorf("notification: row needs id, email, or phone")
	}
	return r, nil
}

func first(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := m[k]; v != "" {
			return v
		}
	}
	return ""
}
