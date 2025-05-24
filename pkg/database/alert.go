// pkg/database/alert.go
package database

import (
	"TradeRadar/pkg/model"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type AlertDB struct {
	db *gorm.DB
}

func (t *TimescaleDB) Alert() *AlertDB {
	return &AlertDB{db: t.db}
}

func (a *AlertDB) Save(alert *model.AlertEvent) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		// 保存异动事件
		if err := tx.Create(alert).Error; err != nil {
			return fmt.Errorf("保存异动事件失败: %w", err)
		}

		// 更新订阅的最后异动时间
		if alert.SubscriptionID != "" {
			if err := tx.Model(&model.Subscription{}).
				Where("id = ?", alert.SubscriptionID).
				Update("last_alert_at", time.Now()).Error; err != nil {
				return fmt.Errorf("更新订阅最后异动时间失败: %w", err)
			}
		}

		return nil
	})
}

func (a *AlertDB) GetByID(alertID string) (*model.AlertEvent, error) {
	var alert model.AlertEvent
	err := a.db.Preload("User").Preload("Stock").Preload("Subscription").
		First(&alert, "id = ?", alertID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("异动事件不存在")
		}
		return nil, fmt.Errorf("获取异动事件失败: %w", err)
	}
	return &alert, nil
}

func (a *AlertDB) GetByUserID(userID string, limit, offset int, onlyUnread bool) ([]*model.AlertEvent, error) {
	var alerts []*model.AlertEvent
	query := a.db.Where("user_id = ?", userID)

	if onlyUnread {
		query = query.Where("is_read = ?", false)
	}

	err := query.Preload("Stock").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&alerts).Error

	if err != nil {
		return nil, fmt.Errorf("查询用户异动事件失败: %w", err)
	}
	return alerts, nil
}

func (a *AlertDB) GetBySymbol(userID, symbol string, limit int) ([]*model.AlertEvent, error) {
	var alerts []*model.AlertEvent
	query := a.db.Where("user_id = ? AND symbol = ?", userID, symbol)

	err := query.Preload("Stock").
		Order("created_at DESC").
		Limit(limit).
		Find(&alerts).Error

	if err != nil {
		return nil, fmt.Errorf("查询股票异动事件失败: %w", err)
	}
	return alerts, nil
}

func (a *AlertDB) GetByType(userID string, alertType model.AlertType, limit int) ([]*model.AlertEvent, error) {
	var alerts []*model.AlertEvent
	err := a.db.Where("user_id = ? AND type = ?", userID, alertType).
		Preload("Stock").
		Order("created_at DESC").
		Limit(limit).
		Find(&alerts).Error

	if err != nil {
		return nil, fmt.Errorf("查询指定类型异动事件失败: %w", err)
	}
	return alerts, nil
}

func (a *AlertDB) GetBySeverity(userID string, severity model.AlertSeverity, limit int) ([]*model.AlertEvent, error) {
	var alerts []*model.AlertEvent
	err := a.db.Where("user_id = ? AND severity = ?", userID, severity).
		Preload("Stock").
		Order("created_at DESC").
		Limit(limit).
		Find(&alerts).Error

	if err != nil {
		return nil, fmt.Errorf("查询指定严重程度异动事件失败: %w", err)
	}
	return alerts, nil
}

func (a *AlertDB) GetByTimeRange(userID string, startTime, endTime time.Time, limit int) ([]*model.AlertEvent, error) {
	var alerts []*model.AlertEvent
	err := a.db.Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startTime, endTime).
		Preload("Stock").
		Order("created_at DESC").
		Limit(limit).
		Find(&alerts).Error

	if err != nil {
		return nil, fmt.Errorf("查询时间范围异动事件失败: %w", err)
	}
	return alerts, nil
}

func (a *AlertDB) MarkAsRead(alertID string) error {
	return a.db.Model(&model.AlertEvent{}).
		Where("id = ?", alertID).
		Update("is_read", true).Error
}

func (a *AlertDB) MarkAsNotified(alertID string) error {
	return a.db.Model(&model.AlertEvent{}).
		Where("id = ?", alertID).
		Update("is_notified", true).Error
}

func (a *AlertDB) GetUnreadCount(userID string) (int64, error) {
	var count int64
	err := a.db.Model(&model.AlertEvent{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (a *AlertDB) GetStatsByUser(userID string, days int) (map[string]int64, error) {
	stats := make(map[string]int64)
	startTime := time.Now().AddDate(0, 0, -days)

	// 按类型统计
	var typeStats []struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}

	err := a.db.Model(&model.AlertEvent{}).
		Select("type, COUNT(*) as count").
		Where("user_id = ? AND created_at >= ?", userID, startTime).
		Group("type").
		Find(&typeStats).Error

	if err != nil {
		return nil, fmt.Errorf("统计异动事件失败: %w", err)
	}

	for _, stat := range typeStats {
		stats[stat.Type] = stat.Count
	}

	return stats, nil
}
