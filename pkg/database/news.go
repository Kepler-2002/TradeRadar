// pkg/database/news.go
package database

import (
	"TradeRadar/pkg/model"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type NewsDB struct {
	db *gorm.DB
}

func (t *TimescaleDB) News() *NewsDB {
	return &NewsDB{db: t.db}
}

func (n *NewsDB) Save(news *model.NewsEvent) error {
	return n.db.Save(news).Error
}

func (n *NewsDB) GetBySymbol(symbol string, limit int) ([]*model.NewsEvent, error) {
	var newsEvents []*model.NewsEvent
	err := n.db.Where("symbol = ?", symbol).
		Order("published_at DESC").
		Limit(limit).
		Find(&newsEvents).Error

	if err != nil {
		return nil, fmt.Errorf("查询股票新闻失败: %w", err)
	}
	return newsEvents, nil
}

func (n *NewsDB) GetRecent(limit int) ([]*model.NewsEvent, error) {
	var newsEvents []*model.NewsEvent
	err := n.db.Order("published_at DESC").
		Limit(limit).
		Find(&newsEvents).Error

	if err != nil {
		return nil, fmt.Errorf("查询最新新闻失败: %w", err)
	}
	return newsEvents, nil
}

func (n *NewsDB) GetBySentiment(sentiment model.NewsSentiment, limit int) ([]*model.NewsEvent, error) {
	var newsEvents []*model.NewsEvent
	err := n.db.Where("sentiment = ?", sentiment).
		Order("published_at DESC").
		Limit(limit).
		Find(&newsEvents).Error

	if err != nil {
		return nil, fmt.Errorf("查询新闻失败: %w", err)
	}
	return newsEvents, nil
}

func (n *NewsDB) GetByTimeRange(startTime, endTime time.Time, limit int) ([]*model.NewsEvent, error) {
	var newsEvents []*model.NewsEvent
	err := n.db.Where("published_at BETWEEN ? AND ?", startTime, endTime).
		Order("published_at DESC").
		Limit(limit).
		Find(&newsEvents).Error

	if err != nil {
		return nil, fmt.Errorf("查询时间范围新闻失败: %w", err)
	}
	return newsEvents, nil
}

// 根据影响程度获取新闻
func (n *NewsDB) GetByImpact(minImpact float64, limit int) ([]*model.NewsEvent, error) {
	var newsEvents []*model.NewsEvent
	err := n.db.Where("impact >= ?", minImpact).
		Order("impact DESC, published_at DESC").
		Limit(limit).
		Find(&newsEvents).Error

	if err != nil {
		return nil, fmt.Errorf("查询高影响新闻失败: %w", err)
	}
	return newsEvents, nil
}

// 检查新闻是否已存在（根据URL去重）
func (n *NewsDB) ExistsByURL(url string) (bool, error) {
	var count int64
	err := n.db.Model(&model.NewsEvent{}).Where("url = ?", url).Count(&count).Error
	return count > 0, err
}
