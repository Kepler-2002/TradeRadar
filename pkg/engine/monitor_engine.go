// pkg/engine/monitor_engine.go (更新部分)
package engine

import (
	"TradeRadar/pkg/collector"
	"TradeRadar/pkg/llm"
	"TradeRadar/pkg/model"
	"fmt"
	"log"
	"time"
)

// MonitorEngine 完整的监控引擎
type MonitorEngine struct {
	ruleEngine    *RuleEngine
	collector     collector.QuoteFetcher
	newsCollector collector.NewsCollector // 使用接口
	llmClient     *llm.LLMClient
	alertChan     chan<- model.AlertEvent
}

// NewMonitorEngine 创建监控引擎
func NewMonitorEngine(
	alertChan chan<- model.AlertEvent,
	quoteFetcher collector.QuoteFetcher,
	newsCollector collector.NewsCollector, // 注入NewsCollector
	llmClient *llm.LLMClient,
) *MonitorEngine {
	return &MonitorEngine{
		ruleEngine:    NewRuleEngine(alertChan),
		collector:     quoteFetcher,
		newsCollector: newsCollector,
		llmClient:     llmClient,
		alertChan:     alertChan,
	}
}

// StartMonitoring 开始监控
func (e *MonitorEngine) StartMonitoring() error {
	// 启动新闻收集器
	if err := e.newsCollector.StartCollecting(); err != nil {
		return fmt.Errorf("启动新闻收集器失败: %w", err)
	}

	// 订阅新闻更新，自动检测新闻异动
	e.newsCollector.SubscribeNewsUpdates(e.handleNewsUpdate)

	// 启动定时行情检查
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.checkAllSubscriptions()
		}
	}
}

// handleNewsUpdate 处理新闻更新
func (e *MonitorEngine) handleNewsUpdate(news *model.NewsEvent) {
	// 检查是否有用户订阅了这个股票的新闻
	subscriptions := e.ruleEngine.GetSubscriptions()

	for _, subscription := range subscriptions {
		if subscription.Status != model.SubscriptionStatusActive {
			continue
		}

		// 检查用户是否订阅了这个股票或者是通用新闻
		shouldNotify := false
		if news.Symbol == "" {
			// 通用新闻，检查是否有新闻异动规则
			shouldNotify = e.hasNewsAlertRule(subscription)
		} else {
			// 特定股票新闻
			for _, symbol := range subscription.Symbols {
				if symbol == news.Symbol {
					shouldNotify = e.hasNewsAlertRule(subscription)
					break
				}
			}
		}

		if shouldNotify && e.isSignificantNews(news) {
			// 生成新闻异动事件
			alert := model.AlertEvent{
				UserID:         subscription.UserID,
				SubscriptionID: subscription.ID,
				Symbol:         news.Symbol,
				StockName:      "", // 需要从股票信息中获取
				Type:           model.AlertTypeNewsImpact,
				Severity:       e.calculateNewsSeverity(news),
				Title:          "重要新闻提醒",
				Message:        fmt.Sprintf("检测到重要新闻: %s", news.Title),
				AIAnalysis:     e.generateNewsAnalysis(news),
				Intensity:      news.Impact,
				Threshold:      0.5, // 新闻异动阈值
				IsRead:         false,
				IsNotified:     false,
				CreatedAt:      time.Now(),
			}

			e.alertChan <- alert
		}
	}
}

// hasNewsAlertRule 检查是否有新闻异动规则
func (e *MonitorEngine) hasNewsAlertRule(subscription *model.Subscription) bool {
	for _, rule := range subscription.AlertRules {
		if rule.Type == model.AlertTypeNewsImpact && rule.Enabled {
			return true
		}
	}
	return false
}

// checkNewsAlerts 检查新闻异动 (更新原有方法)
func (e *MonitorEngine) checkNewsAlerts(userID string, symbols []string) {
	// 获取最近的新闻
	for _, symbol := range symbols {
		news, err := e.newsCollector.GetLatestNews(symbol)
		if err != nil {
			log.Printf("获取新闻失败: %v", err)
			continue
		}

		for _, newsItem := range news {
			if e.isSignificantNews(newsItem) {
				// 生成新闻异动事件
				alert := model.AlertEvent{
					UserID:     userID,
					Symbol:     symbol,
					Type:       model.AlertTypeNewsImpact,
					Severity:   e.calculateNewsSeverity(newsItem),
					Title:      "重要新闻提醒",
					Message:    fmt.Sprintf("检测到 %s 相关重要新闻", symbol),
					AIAnalysis: e.generateNewsAnalysis(newsItem),
					CreatedAt:  time.Now(),
				}

				e.alertChan <- alert
			}
		}
	}
}

// isSignificantNews 判断是否为重要新闻
func (e *MonitorEngine) isSignificantNews(news *model.NewsEvent) bool {
	return news.Impact > 0.7 || news.Sentiment == model.NewsSentimentNegative
}

// calculateNewsSeverity 计算新闻严重程度
func (e *MonitorEngine) calculateNewsSeverity(news *model.NewsEvent) model.AlertSeverity {
	if news.Impact >= 0.9 {
		return model.SeverityCritical
	} else if news.Impact >= 0.7 {
		return model.SeverityHigh
	} else if news.Impact >= 0.5 {
		return model.SeverityMedium
	}
	return model.SeverityLow
}

// generateNewsAnalysis 生成新闻分析
func (e *MonitorEngine) generateNewsAnalysis(news *model.NewsEvent) string {
	if e.llmClient == nil {
		return "AI分析功能暂不可用"
	}

	prompt := fmt.Sprintf("请分析这条新闻对股票 %s 的影响：%s\n内容：%s",
		news.Symbol, news.Title, news.Summary)
	analysis, err := e.llmClient.GenerateAnalysis(prompt)
	if err != nil {
		return "AI分析生成失败"
	}
	return analysis
}
