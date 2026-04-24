package models

import "time"

// ContactMessage web sitesi iletişim formundan gelen mesajlar (anonim gönderim; BaseModel kullanılmaz).
type ContactMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	Name      string     `gorm:"size:255;not null" json:"name"`
	Email     string     `gorm:"size:255;not null;index" json:"email"`
	Phone     string     `gorm:"size:50" json:"phone,omitempty"`
	Subject   string     `gorm:"size:100" json:"subject,omitempty"`
	Message   string     `gorm:"type:text;not null" json:"message"`
	IP        string     `gorm:"size:45" json:"ip,omitempty"`
	UserAgent string     `gorm:"size:500" json:"user_agent,omitempty"`
	ReadAt    *time.Time `gorm:"index" json:"read_at,omitempty"`
}

func (ContactMessage) TableName() string {
	return "contact_messages"
}
