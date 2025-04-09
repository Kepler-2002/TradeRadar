package repository

import (
	"fmt"
	"sync"
	"time"

	"github.com/dewei/TradeRadar/pkg/model"
)

// Repository 数据仓库
type Repository struct {
	// 实际项目中应该使用数据库连接
	// 这里使用内存存储作为示例
	subscriptions map[string][]Subscription
	alerts        []model.AlertEvent
	mutex         sync.RWMutex
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
		subscriptions: make(map[string][]Subscription),
		alerts:        make([]model.AlertEvent, 0),
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