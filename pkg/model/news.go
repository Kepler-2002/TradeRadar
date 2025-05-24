// pkg/model/news.go
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type NewsSentiment string

const (
	NewsSentimentPositive NewsSentiment = "positive"
	NewsSentimentNegative NewsSentiment = "negative"
	NewsSentimentNeutral  NewsSentiment = "neutral"
)

// NewsEvent 新闻事件
type NewsEvent struct {
	ID          string        `gorm:"type:uuid;primaryKey" json:"id"`
	Symbol      string        `gorm:"type:varchar(20);not null;index" json:"symbol"`
	Title       string        `gorm:"not null" json:"title"`
	Content     string        `gorm:"type:text" json:"content"`
	Summary     string        `gorm:"type:text" json:"summary"` // 新闻摘要
	Source      string        `json:"source"`
	Author      string        `json:"author"`
	URL         string        `gorm:"uniqueIndex" json:"url"`
	Sentiment   NewsSentiment `gorm:"type:varchar(20);default:'neutral'" json:"sentiment"`
	Impact      float64       `gorm:"default:0" json:"impact"`    // 影响程度 0-1
	Keywords    []string      `gorm:"type:jsonb" json:"keywords"` // 关键词
	PublishedAt time.Time     `gorm:"not null;index" json:"published_at"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`

	// 关联
	Stock Stock `gorm:"foreignKey:Symbol;references:Symbol" json:"stock,omitempty"`
}

func (n *NewsEvent) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	return nil
}
