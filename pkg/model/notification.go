// pkg/model/notification.go
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

// NotificationRecord 通知记录
type NotificationRecord struct {
	ID        string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    string     `gorm:"type:uuid;not null;index" json:"user_id"`
	AlertID   string     `gorm:"type:uuid;not null;index" json:"alert_id"`
	Type      string     `gorm:"type:varchar(20);not null" json:"type"` // email, sms, push, webhook
	Title     string     `gorm:"not null" json:"title"`
	Content   string     `json:"content"`
	Status    string     `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending, sent, failed
	SentAt    *time.Time `json:"sent_at"`
	Error     string     `json:"error"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	// 关联
	User  User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Alert AlertEvent `gorm:"foreignKey:AlertID" json:"alert,omitempty"`
}

func (n *NotificationRecord) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	return nil
}
