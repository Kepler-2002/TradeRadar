// pkg/model/summary.go
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

// DailySummary 每日总结
type DailySummary struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      string    `gorm:"type:uuid;not null;index" json:"user_id"`
	Date        time.Time `gorm:"type:date;not null;index" json:"date"`
	AlertCount  int       `gorm:"default:0" json:"alert_count"`
	TopSymbols  []string  `gorm:"type:jsonb" json:"top_symbols"`     // 最活跃股票
	Summary     string    `gorm:"type:text" json:"summary"`          // AI生成的总结
	IsGenerated bool      `gorm:"default:false" json:"is_generated"` // 是否已生成
	IsSent      bool      `gorm:"default:false" json:"is_sent"`      // 是否已发送
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (d *DailySummary) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// 复合索引
func (DailySummary) TableName() string {
	return "daily_summaries"
}
