// pkg/model/user.go
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID          string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username    string     `gorm:"uniqueIndex;not null" json:"username"`
	Email       string     `gorm:"uniqueIndex" json:"email"`
	Phone       string     `gorm:"uniqueIndex" json:"phone"`
	Nickname    string     `json:"nickname"`
	Avatar      string     `json:"avatar"`
	Status      int        `gorm:"default:1;index" json:"status"` // 1:正常 0:禁用
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at"`

	// 关联关系
	Subscriptions  []Subscription       `gorm:"foreignKey:UserID" json:"subscriptions,omitempty"`
	Alerts         []AlertEvent         `gorm:"foreignKey:UserID" json:"alerts,omitempty"`
	Notifications  []NotificationRecord `gorm:"foreignKey:UserID" json:"notifications,omitempty"`
	DailySummaries []DailySummary       `gorm:"foreignKey:UserID" json:"daily_summaries,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}
