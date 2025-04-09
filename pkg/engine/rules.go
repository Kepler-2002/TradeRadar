package engine

import (
	"math"
	"time"

	"TradeRadar/pkg/model"
)

// RuleEngine 异动检测规则引擎
type RuleEngine struct {
	rules     map[string][]model.DetectionRule
	alertChan chan<- model.AlertEvent
}

// NewRuleEngine 创建规则引擎
func NewRuleEngine(alertChan chan<- model.AlertEvent) *RuleEngine {
	return &RuleEngine{
		rules:     make(map[string][]model.DetectionRule),
		alertChan: alertChan,
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
		var triggered bool
		var intensity float64

		switch rule.Type {
		case model.AlertPriceVolatility:
			if math.Abs(quote.ChangePercent) >= rule.Threshold {
				triggered = true
				intensity = math.Abs(quote.ChangePercent) / rule.Threshold
			}
		case model.AlertVolumeSpike:
			// 这里需要历史数据比较，暂时简化处理
			if quote.Volume >= rule.Threshold {
				triggered = true
				intensity = quote.Volume / rule.Threshold
			}
		}

		if triggered {
			e.alertChan <- model.AlertEvent{
				Symbol:    quote.Symbol,
				Type:      rule.Type,
				Intensity: intensity,
				Timestamp: time.Now(),
			}
		}
	}
}

// calculateIntensity 计算异动强度
func calculateIntensity(quote model.StockQuote) float64 {
	// 简单实现，实际可能需要更复杂的计算
	return math.Abs(quote.ChangePercent)
}