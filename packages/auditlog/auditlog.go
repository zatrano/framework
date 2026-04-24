// Package auditlog — kurumsal denetim kaydı.
// Kim, ne zaman, neyi, nasıl değiştirdi bilgisini DB'ye yazar.
// GDPR / SOC2 / ISO27001 uyumu için gereklidir.
package auditlog

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/packages/requestid"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Action — denetim eylemi türleri
type Action string

const (
	ActionCreate Action = "CREATE"
	ActionUpdate Action = "UPDATE"
	ActionDelete Action = "DELETE"
	ActionLogin  Action = "LOGIN"
	ActionLogout Action = "LOGOUT"
	ActionView   Action = "VIEW"
	ActionExport Action = "EXPORT"
	ActionFailed Action = "FAILED"
)

// AuditLog — DB tablosu
type AuditLog struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
	UserID     *uint          `gorm:"index" json:"user_id,omitempty"`
	UserEmail  string         `gorm:"size:100;index" json:"user_email,omitempty"`
	Action     Action         `gorm:"size:20;index" json:"action"`
	Resource   string         `gorm:"size:50;index" json:"resource"`  // tablo adı veya entity
	ResourceID *uint          `gorm:"index" json:"resource_id,omitempty"`
	OldValues  string         `gorm:"type:text" json:"old_values,omitempty"` // JSON
	NewValues  string         `gorm:"type:text" json:"new_values,omitempty"` // JSON
	IP         string         `gorm:"size:45" json:"ip,omitempty"`
	UserAgent  string         `gorm:"size:300" json:"user_agent,omitempty"`
	RequestID  string         `gorm:"size:36;index" json:"request_id,omitempty"`
	Extra      string         `gorm:"type:text" json:"extra,omitempty"` // ek bilgi JSON
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// Entry — log kaydı oluşturmak için builder.
type Entry struct {
	UserID     *uint
	UserEmail  string
	Action     Action
	Resource   string
	ResourceID *uint
	OldValues  interface{}
	NewValues  interface{}
	IP         string
	UserAgent  string
	Extra      map[string]interface{}
}

// Log — audit kaydını arka planda DB'ye yazar (non-blocking).
// ctx içinde request_id varsa otomatik eklenir.
func Log(ctx context.Context, e Entry) {
	go func() {
		record := &AuditLog{
			UserID:    e.UserID,
			UserEmail: e.UserEmail,
			Action:    e.Action,
			Resource:  e.Resource,
			ResourceID: e.ResourceID,
			IP:        e.IP,
			UserAgent: e.UserAgent,
			RequestID: requestid.FromContext(ctx),
		}
		if e.OldValues != nil {
			if b, err := json.Marshal(e.OldValues); err == nil {
				record.OldValues = string(b)
			}
		}
		if e.NewValues != nil {
			if b, err := json.Marshal(e.NewValues); err == nil {
				record.NewValues = string(b)
			}
		}
		if e.Extra != nil {
			if b, err := json.Marshal(e.Extra); err == nil {
				record.Extra = string(b)
			}
		}
		db := databaseconfig.GetDB()
		if err := db.WithContext(ctx).Create(record).Error; err != nil {
			logconfig.Log.Warn("Audit log yazma hatası",
				zap.String("action", string(e.Action)),
				zap.String("resource", e.Resource),
				zap.Error(err))
		}
	}()
}

// LogLogin — giriş denemelerini kaydeder.
func LogLogin(ctx context.Context, userID *uint, email, ip, ua string, success bool) {
	action := ActionLogin
	if !success {
		action = ActionFailed
	}
	Log(ctx, Entry{
		UserID:    userID,
		UserEmail: email,
		Action:    action,
		Resource:  "auth",
		IP:        ip,
		UserAgent: ua,
	})
}

// LogCRUD — CRUD işlemlerini kaydeder.
func LogCRUD(ctx context.Context, userID *uint, email string, action Action,
	resource string, resourceID *uint, oldVal, newVal interface{}, ip string) {
	Log(ctx, Entry{
		UserID:     userID,
		UserEmail:  email,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		OldValues:  oldVal,
		NewValues:  newVal,
		IP:         ip,
	})
}

// Migrate — audit_logs tablosunu oluşturur.
// database/migrations/schema.go içinde çağrılmalıdır.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&AuditLog{})
}
