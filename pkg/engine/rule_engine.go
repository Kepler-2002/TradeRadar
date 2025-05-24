// pkg/engine/rule_engine.go
package engine

import (
	"fmt"
	"github.com/google/uuid"
	"log"
	"math"
	"time"

	"TradeRadar/pkg/model"
)

// RuleEngine 异动检测规则引擎
type RuleEngine struct {
	rules         map[string][]model.DetectionRule
	subscriptions map[string]*model.Subscription
	alertChan     chan<- model.AlertEvent
}

// NewRuleEngine 创建规则引擎
func NewRuleEngine(alertChan chan<- model.AlertEvent) *RuleEngine {
	return &RuleEngine{
		rules:         make(map[string][]model.DetectionRule),
		subscriptions: make(map[string]*model.Subscription),
		alertChan:     alertChan,
	}
}

// AddRule 添加规则
func (e *RuleEngine) AddRule(symbol string, rule model.DetectionRule) {
	if _, exists := e.rules[symbol]; !exists {
		e.rules[symbol] = make([]model.DetectionRule, 0)
	}
	e.rules[symbol] = append(e.rules[symbol], rule)
}

// ReloadRules 重新加载规则
func (e *RuleEngine) ReloadRules(rules map[string][]model.DetectionRule) {
	e.rules = rules
}

// Evaluate 评估股票行情是否触发规则
func (e *RuleEngine) Evaluate(quote model.StockQuote) {
	// 检查全局规则
	e.evaluateRules("*", quote)

	// 检查特定股票规则
	e.evaluateRules(quote.Symbol, quote)
}

// evaluateRules 评估特定规则集
func (e *RuleEngine) evaluateRules(symbol string, quote model.StockQuote) {
	rules, exists := e.rules[symbol]
	if !exists {
		return
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		var triggered bool
		var intensity float64
		var severity model.AlertSeverity
		var message string

		switch rule.Type {
		case model.AlertTypePriceVolatility:
			if math.Abs(quote.ChangePercent) >= rule.Threshold {
				triggered = true
				intensity = math.Abs(quote.ChangePercent) / rule.Threshold
				severity = e.calculateSeverity(intensity)
				message = fmt.Sprintf("%s 价格异动：涨跌幅 %.2f%%，超过阈值 %.2f%%",
					quote.Symbol, quote.ChangePercent, rule.Threshold)
			}
		case model.AlertTypeVolumeSpike:
			// 这里需要历史数据比较，暂时简化处理
			if quote.Volume >= rule.Threshold {
				triggered = true
				intensity = quote.Volume / rule.Threshold
				severity = e.calculateSeverity(intensity)
				message = fmt.Sprintf("%s 成交量异动：当前成交量 %.0f，超过阈值 %.0f",
					quote.Symbol, quote.Volume, rule.Threshold)
			}
		case model.AlertTypeNewsImpact:
			// 新闻异动检测将在后续实现
			continue
		}

		if triggered {
			// 为所有相关用户生成异动事件
			e.generateAlertsForUsers(quote, rule, intensity, severity, message)
		}
	}
}

// generateAlertsForUsers 为相关用户生成异动事件
func (e *RuleEngine) generateAlertsForUsers(quote model.StockQuote, rule model.DetectionRule, intensity float64, severity model.AlertSeverity, message string) {
	// 查找所有订阅了这个股票的用户
	for _, subscription := range e.subscriptions {
		// 检查这个订阅是否包含当前股票
		for _, symbol := range subscription.Symbols {
			if symbol == quote.Symbol {
				// 检查用户是否启用了这种类型的异动检测
				for _, alertRule := range subscription.AlertRules {
					if !alertRule.Enabled {
						continue
					}

					// 匹配规则类型并检查阈值
					if alertRule.Type == rule.Type && math.Abs(quote.ChangePercent) >= alertRule.Threshold {
						// 生成用户特定的异动事件
						alert := model.AlertEvent{
							ID:             uuid.New().String(),
							UserID:         subscription.UserID,
							SubscriptionID: subscription.ID,
							Symbol:         quote.Symbol,
							StockName:      quote.Name,
							Type:           rule.Type,
							Severity:       severity,
							Title:          fmt.Sprintf("%s %s", quote.Name, e.getAlertTitle(rule.Type)),
							Message:        message,
							QuoteData:      quote,
							AIAnalysis:     "", // 可以后续添加AI分析
							Intensity:      intensity,
							Threshold:      rule.Threshold,
							IsRead:         false,
							IsNotified:     false,
							CreatedAt:      time.Now(),
						}

						// 发送异动事件
						select {
						case e.alertChan <- alert:
							log.Printf("发送异动事件: 用户 %s, 股票 %s, 类型 %s",
								alert.UserID, alert.Symbol, alert.Type)
						default:
							log.Printf("警告: 异动事件通道已满，丢弃事件")
						}
					}
				}
				break // 找到匹配的股票后跳出内层循环
			}
		}
	}
}

// getAlertTitle 获取异动标题
func (e *RuleEngine) getAlertTitle(alertType model.AlertType) string {
	switch alertType {
	case model.AlertTypePriceVolatility:
		return "价格异动"
	case model.AlertTypeVolumeSpike:
		return "成交量异动"
	case model.AlertTypeNewsImpact:
		return "新闻异动"
	default:
		return "异动提醒"
	}
}

// calculateSeverity 计算异动严重程度
func (e *RuleEngine) calculateSeverity(intensity float64) model.AlertSeverity {
	if intensity >= 3.0 {
		return model.SeverityCritical
	} else if intensity >= 2.0 {
		return model.SeverityHigh
	} else if intensity >= 1.5 {
		return model.SeverityMedium
	}
	return model.SeverityLow
}

// AddSubscription 添加订阅到规则引擎
func (e *RuleEngine) AddSubscription(subscription *model.Subscription) {
	if subscription == nil {
		log.Printf("警告: 尝试添加空订阅")
		return
	}

	// 存储订阅信息
	e.subscriptions[subscription.ID] = subscription

	// 为订阅中的每个股票添加规则
	for _, symbol := range subscription.Symbols {
		// 转换AlertRule到DetectionRule并添加到规则引擎
		for _, alertRule := range subscription.AlertRules {
			if !alertRule.Enabled {
				continue // 跳过未启用的规则
			}

			// 创建检测规则
			detectionRule := model.DetectionRule{
				ID:          uuid.New().String(),
				Type:        alertRule.Type, // AlertRule.Type 现在已经是 AlertType
				Threshold:   alertRule.Threshold,
				Enabled:     alertRule.Enabled,
				Description: alertRule.Description,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			// 添加规则到引擎
			e.AddRule(symbol, detectionRule)
		}
	}

	log.Printf("成功添加订阅 %s，包含 %d 个股票", subscription.ID, len(subscription.Symbols))
}

// RemoveSubscription 移除订阅
func (e *RuleEngine) RemoveSubscription(subscriptionID string) {
	subscription, exists := e.subscriptions[subscriptionID]
	if !exists {
		log.Printf("警告: 尝试移除不存在的订阅 %s", subscriptionID)
		return
	}

	// 从订阅映射中删除
	delete(e.subscriptions, subscriptionID)

	// 清理相关规则（这里简化处理，实际可能需要更精确的规则管理）
	for _, symbol := range subscription.Symbols {
		// 重新构建该股票的规则（移除该订阅的规则）
		e.rebuildRulesForSymbol(symbol)
	}

	log.Printf("成功移除订阅 %s", subscriptionID)
}

// rebuildRulesForSymbol 重新构建指定股票的规则
func (e *RuleEngine) rebuildRulesForSymbol(symbol string) {
	// 清空该股票的现有规则
	e.rules[symbol] = make([]model.DetectionRule, 0)

	// 重新添加所有相关订阅的规则
	for _, subscription := range e.subscriptions {
		for _, subSymbol := range subscription.Symbols {
			if subSymbol == symbol {
				// 为这个股票重新添加规则
				for _, alertRule := range subscription.AlertRules {
					if !alertRule.Enabled {
						continue
					}

					detectionRule := model.DetectionRule{
						ID:          uuid.New().String(),
						Type:        alertRule.Type,
						Threshold:   alertRule.Threshold,
						Enabled:     alertRule.Enabled,
						Description: alertRule.Description,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					}

					e.AddRule(symbol, detectionRule)
				}
			}
		}
	}
}

// GetSubscriptions 获取所有订阅
func (e *RuleEngine) GetSubscriptions() map[string]*model.Subscription {
	return e.subscriptions
}

// GetSubscription 获取指定订阅
func (e *RuleEngine) GetSubscription(subscriptionID string) (*model.Subscription, bool) {
	subscription, exists := e.subscriptions[subscriptionID]
	return subscription, exists
}

// GetRules 获取指定股票的规则
func (e *RuleEngine) GetRules(symbol string) []model.DetectionRule {
	return e.rules[symbol]
}

// GetAllRules 获取所有规则
func (e *RuleEngine) GetAllRules() map[string][]model.DetectionRule {
	return e.rules
}

// UpdateRule 更新规则
func (e *RuleEngine) UpdateRule(symbol string, ruleID string, updates map[string]interface{}) error {
	rules, exists := e.rules[symbol]
	if !exists {
		return fmt.Errorf("股票 %s 不存在规则", symbol)
	}

	for i, rule := range rules {
		if rule.ID == ruleID {
			// 更新规则字段
			if threshold, ok := updates["threshold"].(float64); ok {
				e.rules[symbol][i].Threshold = threshold
			}
			if enabled, ok := updates["enabled"].(bool); ok {
				e.rules[symbol][i].Enabled = enabled
			}
			if description, ok := updates["description"].(string); ok {
				e.rules[symbol][i].Description = description
			}
			e.rules[symbol][i].UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("规则 %s 不存在", ruleID)
}

// ClearRules 清空所有规则
func (e *RuleEngine) ClearRules() {
	e.rules = make(map[string][]model.DetectionRule)
}

// GetRuleStats 获取规则统计信息
func (e *RuleEngine) GetRuleStats() map[string]interface{} {
	stats := make(map[string]interface{})

	totalRules := 0
	enabledRules := 0
	symbolCount := len(e.rules)

	typeCount := make(map[model.AlertType]int)

	for _, rules := range e.rules {
		for _, rule := range rules {
			totalRules++
			if rule.Enabled {
				enabledRules++
			}
			typeCount[rule.Type]++
		}
	}

	stats["total_rules"] = totalRules
	stats["enabled_rules"] = enabledRules
	stats["symbol_count"] = symbolCount
	stats["subscription_count"] = len(e.subscriptions)
	stats["type_distribution"] = typeCount

	return stats
}
