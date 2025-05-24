// pkg/model/alert.go
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

// AlertType 异动类型枚举
type AlertType string

const (
	AlertTypePriceVolatility AlertType = "price_volatility"
	AlertTypeVolumeSpike     AlertType = "volume_spike"
	AlertTypeNewsImpact      AlertType = "news_alert"
	AlertTypeSystem         AlertType = "system_alert"    // 系统提醒类型
	AlertTypePriceChange    AlertType = "price_change"    // 价格变动
	AlertTypePriceLevel     AlertType = "price_level"     // 价格水平
)

// AlertSeverity 异动严重程度
type AlertSeverity string

const (
	SeverityLow      AlertSeverity = "low"
	SeverityMedium   AlertSeverity = "medium"
	SeverityHigh     AlertSeverity = "high"
	SeverityCritical AlertSeverity = "critical"
)

// AlertRule 异动规则
type AlertRule struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	Type        AlertType `gorm:"type:varchar(30);not null;index" json:"type"`  // 使用统一的枚举
	Threshold   float64   `gorm:"type:decimal(10,4);not null" json:"threshold"` // 阈值
	Enabled     bool      `gorm:"default:true" json:"enabled"`                  // 是否启用
	Description string    `gorm:"type:text" json:"description"`                 // 规则描述
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (a *AlertRule) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// AlertEvent 异动事件
type AlertEvent struct {
	ID             string        `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         string        `gorm:"type:uuid;not null;index" json:"user_id"`
	SubscriptionID string        `gorm:"type:uuid;index" json:"subscription_id"` // 关联订阅
	Symbol         string        `gorm:"type:varchar(20);index" json:"symbol"`           // 允许为空，用于系统提醒
	StockName      string        `json:"stock_name"`                                     // 允许为空，用于系统提醒
	Type           AlertType     `gorm:"type:varchar(30);not null;index" json:"type"`
	Severity       AlertSeverity `gorm:"type:varchar(20);not null;index" json:"severity"`
	Title          string        `gorm:"not null" json:"title"`        // 异动标题
	Message        string        `gorm:"type:text" json:"message"`     // 异动消息
	QuoteData      *StockQuote   `gorm:"type:jsonb" json:"quote_data,omitempty"`        // 修改为指针，允许为空
	AIAnalysis     string        `gorm:"type:text" json:"ai_analysis,omitempty"`
	Intensity      float64       `gorm:"type:decimal(8,4);default:0" json:"intensity"` // 异动强度
	Threshold      float64       `gorm:"type:decimal(10,4)" json:"threshold"`          // 触发阈值
	IsRead         bool          `gorm:"default:false;index" json:"is_read"`           // 是否已读
	IsNotified     bool          `gorm:"default:false;index" json:"is_notified"`       // 是否已通知
	CreatedAt      time.Time     `gorm:"index:idx_created_at" json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	ExpireAt       *time.Time    `gorm:"index" json:"expire_at,omitempty"`              // 提醒过期时间

	// 关联关系
	User         User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
	Stock        Stock        `gorm:"foreignKey:Symbol;references:Symbol" json:"stock,omitempty"`

	// 通知记录
	Notifications []NotificationRecord `gorm:"foreignKey:AlertID" json:"notifications,omitempty"`
}

func (a *AlertEvent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// 自定义表名和索引
func (AlertEvent) TableName() string {
	return "alert_events"
}

// 如果需要额外的复合索引，可以在迁移时添加
func (AlertEvent) Indexes() []string {
	return []string{
		"idx_user_created", // 用户+创建时间复合索引
		"idx_symbol_type",  // 股票代码+类型复合索引
	}
}
