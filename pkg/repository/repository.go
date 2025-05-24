package repository

import (
	"fmt"
	"sync"
	"time"
	"github.com/google/uuid"

	"TradeRadar/pkg/model"
)

// Repository 数据仓库
type Repository struct {
	// 实际项目中应该使用数据库连接
	// 这里使用内存存储作为示例
	subscriptions    map[string][]Subscription
	userSubscriptions map[string][]*model.Subscription // 新增：用户订阅映射
	subscriptionByID  map[string]*model.Subscription    // 新增：按ID索引的订阅
	alerts           []model.AlertEvent
	mutex            sync.RWMutex
}

// Subscription 订阅信息
type Subscription struct {
	UserID    string
	Symbol    string
	Rules     []model.DetectionRule
	CreatedAt time.Time
}

// NewRepository 创建新的数据仓库
func NewRepository() *Repository {
	return &Repository{
		subscriptions:     make(map[string][]Subscription),
		userSubscriptions: make(map[string][]*model.Subscription),
		subscriptionByID:  make(map[string]*model.Subscription),
		alerts:           make([]model.AlertEvent, 0),
	}
}

// SaveSubscription 保存订阅信息
func (r *Repository) SaveSubscription(userID string, symbols []string, rules []model.DetectionRule) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	// 为每个股票创建订阅
	for _, symbol := range symbols {
		sub := Subscription{
			UserID:    userID,
			Symbol:    symbol,
			Rules:     rules,
			CreatedAt: time.Now(),
		}
		
		r.subscriptions[symbol] = append(r.subscriptions[symbol], sub)
	}
	
	return nil
}

// GetSubscriptions 获取股票的所有订阅
func (r *Repository) GetSubscriptions(symbol string) []Subscription {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	return r.subscriptions[symbol]
}

// SaveAlert 保存异动事件
func (r *Repository) SaveAlert(alert model.AlertEvent) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.alerts = append(r.alerts, alert)
	
	// 实际项目中应该保存到数据库
	return nil
}

// GetAlertHistory 获取异动历史
func (r *Repository) GetAlertHistory(symbol string, limit int) ([]model.AlertEvent, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var result []model.AlertEvent
	
	// 如果没有指定股票，返回所有异动
	if symbol == "" {
		if len(r.alerts) <= limit {
			return r.alerts, nil
		}
		return r.alerts[len(r.alerts)-limit:], nil
	}
	
	// 筛选指定股票的异动
	for i := len(r.alerts) - 1; i >= 0 && len(result) < limit; i-- {
		if r.alerts[i].Symbol == symbol {
			result = append(result, r.alerts[i])
		}
	}
	
	return result, nil
}

// LoadActiveRules 加载活跃规则
func (r *Repository) LoadActiveRules() map[string][]model.DetectionRule {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	// 模拟从数据库加载规则
	rules := make(map[string][]model.DetectionRule)
	
	// 全局规则
	rules["*"] = []model.DetectionRule{
		{
			Type:      model.AlertPriceVolatility,
			Threshold: 5.0,
		},
		{
			Type:      model.AlertVolumeSpike,
			Threshold: 1000000,
		},
	}
	
	// 特定股票规则
	rules["000001.SZ"] = []model.DetectionRule{
		{
			Type:      model.AlertPriceVolatility,
			Threshold: 3.0, // 更敏感的阈值
		},
	}
	
	return rules
}

// SaveSubscriptionModel 保存订阅模型
func (r *Repository) SaveSubscriptionModel(subscription *model.Subscription) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 生成ID
	if subscription.ID == "" {
		subscription.ID = uuid.New().String()
	}

	// 保存到用户订阅映射
	r.userSubscriptions[subscription.UserID] = append(r.userSubscriptions[subscription.UserID], subscription)
	
	// 保存到ID索引
	r.subscriptionByID[subscription.ID] = subscription

	// 为了兼容现有代码，也保存到原有格式
	for _, symbol := range subscription.Symbols {
		// 转换AlertRule到DetectionRule
		var rules []model.DetectionRule
		for _, alertRule := range subscription.AlertRules {
			var alertType model.AlertType
			switch alertRule.Type {
			case "price_volatility":
				alertType = model.AlertPriceVolatility
			case "volume_spike":
				alertType = model.AlertVolumeSpike
			case "news_alert":
				alertType = model.AlertNewsImpact
			default:
				alertType = model.AlertPriceVolatility
			}
			
			if alertRule.Enabled {
				rules = append(rules, model.DetectionRule{
					Type:      alertType,
					Threshold: alertRule.Threshold,
				})
			}
		}

		sub := Subscription{
			UserID:    subscription.UserID,
			Symbol:    symbol,
			Rules:     rules,
			CreatedAt: subscription.CreatedAt,
		}
		
		r.subscriptions[symbol] = append(r.subscriptions[symbol], sub)
	}

	return nil
}

// GetUserSubscriptions 获取用户的所有订阅
func (r *Repository) GetUserSubscriptions(userID string) ([]*model.Subscription, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.userSubscriptions[userID], nil
}

// UpdateSubscription 更新订阅
func (r *Repository) UpdateSubscription(subscription *model.Subscription) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 检查订阅是否存在
	existingSub, exists := r.subscriptionByID[subscription.ID]
	if !exists {
		return fmt.Errorf("订阅不存在")
	}

	// 更新订阅信息
	subscription.CreatedAt = existingSub.CreatedAt // 保持创建时间不变
	r.subscriptionByID[subscription.ID] = subscription

	// 更新用户订阅列表
	userSubs := r.userSubscriptions[subscription.UserID]
	for i, sub := range userSubs {
		if sub.ID == subscription.ID {
			userSubs[i] = subscription
			break
		}
	}

	return nil
}

// DeleteSubscription 删除订阅
func (r *Repository) DeleteSubscription(subscriptionID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 获取订阅信息
	subscription, exists := r.subscriptionByID[subscriptionID]
	if !exists {
		return fmt.Errorf("订阅不存在")
	}

	// 从ID索引中删除
	delete(r.subscriptionByID, subscriptionID)

	// 从用户订阅列表中删除
	userSubs := r.userSubscriptions[subscription.UserID]
	for i, sub := range userSubs {
		if sub.ID == subscriptionID {
			r.userSubscriptions[subscription.UserID] = append(userSubs[:i], userSubs[i+1:]...)
			break
		}
	}

	// 从符号订阅中删除（为了兼容现有代码）
	for _, symbol := range subscription.Symbols {
		symbolSubs := r.subscriptions[symbol]
		for i, sub := range symbolSubs {
			if sub.UserID == subscription.UserID {
				r.subscriptions[symbol] = append(symbolSubs[:i], symbolSubs[i+1:]...)
				break
			}
		}
	}

	return nil
}