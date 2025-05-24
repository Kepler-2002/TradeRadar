// pkg/model/subscription.go (更新)
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusPaused    SubscriptionStatus = "paused"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
)

type Subscription struct {
	ID          string             `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      string             `gorm:"type:uuid;not null;index" json:"user_id"`
	Name        string             `gorm:"not null" json:"name"`
	Description string             `gorm:"type:text" json:"description"`
	Symbols     []string           `gorm:"type:jsonb" json:"symbols"`
	AlertRules  []AlertRule        `gorm:"type:jsonb" json:"alert_rules"` // 现在可以引用 AlertRule
	Status      SubscriptionStatus `gorm:"type:varchar(20);default:'active';index" json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	LastAlertAt *time.Time         `json:"last_alert_at"`

	// 关联
	User   User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Alerts []AlertEvent `gorm:"foreignKey:SubscriptionID" json:"alerts,omitempty"`
}

func (s *Subscription) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}
