// pkg/database/subscription.go
package database

import (
	"TradeRadar/pkg/model"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type SubscriptionDB struct {
	db *gorm.DB
}

func (t *TimescaleDB) Subscription() *SubscriptionDB {
	return &SubscriptionDB{db: t.db}
}

func (s *SubscriptionDB) Save(subscription *model.Subscription) error {
	return s.db.Save(subscription).Error
}

func (s *SubscriptionDB) GetByID(subscriptionID string) (*model.Subscription, error) {
	var subscription model.Subscription
	err := s.db.Preload("User").First(&subscription, "id = ?", subscriptionID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("订阅不存在")
		}
		return nil, fmt.Errorf("获取订阅信息失败: %w", err)
	}
	return &subscription, nil
}

func (s *SubscriptionDB) GetByUserID(userID string) ([]*model.Subscription, error) {
	var subscriptions []*model.Subscription
	err := s.db.Where("user_id = ? AND status != ?", userID, model.SubscriptionStatusCancelled).
		Order("created_at DESC").
		Find(&subscriptions).Error

	if err != nil {
		return nil, fmt.Errorf("查询用户订阅失败: %w", err)
	}
	return subscriptions, nil
}

func (s *SubscriptionDB) GetActiveByUserID(userID string) ([]*model.Subscription, error) {
	var subscriptions []*model.Subscription
	err := s.db.Where("user_id = ? AND status = ?", userID, model.SubscriptionStatusActive).
		Order("created_at DESC").
		Find(&subscriptions).Error

	if err != nil {
		return nil, fmt.Errorf("查询用户活跃订阅失败: %w", err)
	}
	return subscriptions, nil
}

func (s *SubscriptionDB) GetBySymbol(symbol string) ([]*model.Subscription, error) {
	var subscriptions []*model.Subscription
	err := s.db.Where("symbols @> ?", fmt.Sprintf(`["%s"]`, symbol)).
		Where("status = ?", model.SubscriptionStatusActive).
		Find(&subscriptions).Error

	if err != nil {
		return nil, fmt.Errorf("查询股票订阅失败: %w", err)
	}
	return subscriptions, nil
}

func (s *SubscriptionDB) Delete(subscriptionID string) error {
	return s.db.Model(&model.Subscription{}).
		Where("id = ?", subscriptionID).
		Update("status", model.SubscriptionStatusCancelled).Error
}

func (s *SubscriptionDB) UpdateStatus(subscriptionID string, status model.SubscriptionStatus) error {
	return s.db.Model(&model.Subscription{}).
		Where("id = ?", subscriptionID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

func (s *SubscriptionDB) UpdateLastAlert(subscriptionID string) error {
	return s.db.Model(&model.Subscription{}).
		Where("id = ?", subscriptionID).
		Update("last_alert_at", time.Now()).Error
}

func (s *SubscriptionDB) GetAllActive() ([]*model.Subscription, error) {
	var subscriptions []*model.Subscription
	err := s.db.Where("status = ?", model.SubscriptionStatusActive).
		Preload("User").
		Find(&subscriptions).Error

	if err != nil {
		return nil, fmt.Errorf("查询所有活跃订阅失败: %w", err)
	}
	return subscriptions, nil
}

func (s *SubscriptionDB) GetExpiringSoon(days int) ([]*model.Subscription, error) {
	// 如果有过期逻辑的话
	var subscriptions []*model.Subscription
	cutoffTime := time.Now().AddDate(0, 0, days)

	err := s.db.Where("status = ? AND last_alert_at < ?",
		model.SubscriptionStatusActive, cutoffTime).
		Find(&subscriptions).Error

	if err != nil {
		return nil, fmt.Errorf("查询即将过期订阅失败: %w", err)
	}
	return subscriptions, nil
}
