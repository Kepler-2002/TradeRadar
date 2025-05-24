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
	analysis, err := e.llmClient.GenerateNewsAnalysis(news.Symbol, news.Content)
	if err != nil {
		return "AI分析生成失败"
	}
	return analysis
}

// checkAllSubscriptions 检查所有订阅的状态和触发条件
func (e *MonitorEngine) checkAllSubscriptions() {
	subscriptions := e.ruleEngine.GetSubscriptions()
	now := time.Now()

	for _, subscription := range subscriptions {
		// 检查订阅状态
		if !e.isSubscriptionValid(subscription) {
			continue
		}

		// 更新最后检查时间
		subscription.LastCheckedAt = &now

		// 检查每个股票的异动规则
		for _, symbol := range subscription.Symbols {
			// 检查行情异动
			if e.collector != nil {
				e.checkQuoteAlerts(subscription, symbol)
			}

			// 检查新闻异动
			if e.newsCollector != nil {
				e.checkNewsAlerts(subscription.UserID, []string{symbol})
			}
		}

		// 检查订阅级别的规则
		e.checkSubscriptionRules(subscription)
	}
}

// checkQuoteAlerts 检查行情异动
func (e *MonitorEngine) checkQuoteAlerts(subscription *model.Subscription, symbol string) {
	if e.collector == nil {
		return
	}

	// 获取当前行情
	quote, err := e.collector.FetchQuote(symbol)
	if err != nil {
		log.Printf("获取 %s 行情失败: %v", symbol, err)
		return
	}

	// 检查订阅中的各种异动规则
	for _, rule := range subscription.AlertRules {
		if !rule.Enabled {
			continue
		}

		switch rule.Type {
		case model.AlertTypePriceChange:
			e.checkPriceChangeRule(subscription, quote, rule)
		case model.AlertTypeVolumeSpike:
			e.checkVolumeRule(subscription, quote, rule)
		case model.AlertTypePriceLevel:
			e.checkPriceLevelRule(subscription, quote, rule)
			// 可以继续添加其他规则类型
		}
	}
}

// checkPriceChangeRule 检查价格变动规则
func (e *MonitorEngine) checkPriceChangeRule(subscription *model.Subscription, quote *model.StockQuote, rule model.AlertRule) {
	// 检查涨跌幅是否超过阈值
	if abs(quote.ChangePercent) >= rule.Threshold {
		alert := model.AlertEvent{
			UserID:         subscription.UserID,
			SubscriptionID: subscription.ID,
			Symbol:         quote.Symbol,
			StockName:      quote.Name,
			Type:           model.AlertTypePriceChange,
			Severity:       e.calculatePriceSeverity(quote.ChangePercent, rule.Threshold),
			Title:          "价格异动提醒",
			Message: fmt.Sprintf("%s 涨跌幅达到 %.2f%%，触发预设阈值 %.2f%%",
				quote.Name, quote.ChangePercent, rule.Threshold),
			AIAnalysis: e.generatePriceAnalysis(quote),
			Intensity:  abs(quote.ChangePercent) / rule.Threshold,
			Threshold:  rule.Threshold,
			IsRead:     false,
			IsNotified: false,
			CreatedAt:  time.Now(),
		}

		log.Printf("触发价格异动: %s 涨跌幅 %.2f%%", quote.Symbol, quote.ChangePercent)
		e.alertChan <- alert
	}
}

// checkVolumeRule 检查成交量规则
func (e *MonitorEngine) checkVolumeRule(subscription *model.Subscription, quote *model.StockQuote, rule model.AlertRule) {
	// 这里需要获取历史成交量来计算是否为异常放量
	// 简化处理：假设成交量比平均值高出rule.Threshold倍就算异动
	// 实际实现中您可能需要从数据库或其他数据源获取历史数据

	// 模拟：如果成交量超过某个基准值就触发
	if quote.Volume > rule.Threshold {
		alert := model.AlertEvent{
			UserID:         subscription.UserID,
			SubscriptionID: subscription.ID,
			Symbol:         quote.Symbol,
			StockName:      quote.Name,
			Type:           model.AlertTypeVolumeSpike,
			Severity:       model.SeverityMedium,
			Title:          "成交量异动提醒",
			Message:        fmt.Sprintf("%s 成交量达到 %.0f，超过预设阈值", quote.Name, quote.Volume),
			AIAnalysis:     e.generateVolumeAnalysis(quote),
			Intensity:      quote.Volume / rule.Threshold,
			Threshold:      rule.Threshold,
			IsRead:         false,
			IsNotified:     false,
			CreatedAt:      time.Now(),
		}

		log.Printf("触发成交量异动: %s 成交量 %.0f", quote.Symbol, quote.Volume)
		e.alertChan <- alert
	}
}

// checkPriceLevelRule 检查价格水平规则
func (e *MonitorEngine) checkPriceLevelRule(subscription *model.Subscription, quote *model.StockQuote, rule model.AlertRule) {
	// 检查是否突破关键价位
	if quote.Price >= rule.Threshold {
		alert := model.AlertEvent{
			UserID:         subscription.UserID,
			SubscriptionID: subscription.ID,
			Symbol:         quote.Symbol,
			StockName:      quote.Name,
			Type:           model.AlertTypePriceLevel,
			Severity:       model.SeverityMedium,
			Title:          "价位突破提醒",
			Message: fmt.Sprintf("%s 价格达到 %.2f，突破关键价位 %.2f",
				quote.Name, quote.Price, rule.Threshold),
			AIAnalysis: e.generatePriceLevelAnalysis(quote, rule.Threshold),
			Intensity:  quote.Price / rule.Threshold,
			Threshold:  rule.Threshold,
			IsRead:     false,
			IsNotified: false,
			CreatedAt:  time.Now(),
		}

		log.Printf("触发价位突破: %s 价格 %.2f", quote.Symbol, quote.Price)
		e.alertChan <- alert
	}
}

// checkSubscriptionRules 检查订阅级别的规则
func (e *MonitorEngine) checkSubscriptionRules(subscription *model.Subscription) {
	now := time.Now()

	// 检查最后检查时间
	if subscription.LastCheckedAt != nil {
		timeSinceLastCheck := now.Sub(*subscription.LastCheckedAt)

		// 如果超过10分钟未检查，记录警告
		if timeSinceLastCheck > 10*time.Minute {
			log.Printf("警告: 订阅 %s 超过10分钟未检查", subscription.ID)
		}
	}

	// 检查订阅即将过期
	if subscription.ExpiresAt != nil {
		timeUntilExpiry := subscription.ExpiresAt.Sub(now)

		// 如果订阅将在7天内过期，生成提醒
		if timeUntilExpiry > 0 && timeUntilExpiry <= 7*24*time.Hour {
			alert := model.AlertEvent{
				UserID:         subscription.UserID,
				SubscriptionID: subscription.ID,
				Type:           model.AlertTypeSystem,
				Severity:       model.SeverityMedium,
				Title:          "订阅即将过期提醒",
				Message:        fmt.Sprintf("您的订阅将在 %.1f 天后过期", timeUntilExpiry.Hours()/24),
				IsRead:         false,
				IsNotified:     false,
				CreatedAt:      now,
			}

			e.alertChan <- alert
		}
	}

	//// 检查订阅的使用情况
	//if subscription.QuotaLimit > 0 && subscription.QuotaUsed >= subscription.QuotaLimit {
	//	alert := model.AlertEvent{
	//		UserID:         subscription.UserID,
	//		SubscriptionID: subscription.ID,
	//		Type:           model.AlertTypeSystem,
	//		Severity:       model.SeverityHigh,
	//		Title:          "订阅配额已用尽提醒",
	//		Message:        "您的订阅配额已用尽，请考虑升级订阅",
	//		IsRead:         false,
	//		IsNotified:     false,
	//		CreatedAt:      now,
	//	}
	//
	//	e.alertChan <- alert
	//}
}

// 辅助方法

// abs 返回浮点数的绝对值
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// calculatePriceSeverity 根据涨跌幅计算严重程度
func (e *MonitorEngine) calculatePriceSeverity(changePercent, threshold float64) model.AlertSeverity {
	absChange := abs(changePercent)
	if absChange >= threshold*2 {
		return model.SeverityCritical
	} else if absChange >= threshold*1.5 {
		return model.SeverityHigh
	} else if absChange >= threshold {
		return model.SeverityMedium
	}
	return model.SeverityLow
}

// generatePriceAnalysis 生成价格分析
func (e *MonitorEngine) generatePriceAnalysis(quote *model.StockQuote) string {
	if e.llmClient == nil {
		return fmt.Sprintf("股票 %s 当前价格 %.2f，涨跌幅 %.2f%%",
			quote.Symbol, quote.Price, quote.ChangePercent)
	}

	// 调用LLM生成分析
	analysis, err := e.llmClient.GenerateStockAnalysis(
		quote.Symbol,
		quote.Name,
		quote.Price,
		quote.ChangePercent,
	)
	if err != nil {
		return fmt.Sprintf("股票 %s 出现价格异动，当前价格 %.2f", quote.Symbol, quote.Price)
	}
	return analysis
}

// generateVolumeAnalysis 生成成交量分析
func (e *MonitorEngine) generateVolumeAnalysis(quote *model.StockQuote) string {
	if e.llmClient == nil {
		return fmt.Sprintf("股票 %s 成交量异常，当前成交量 %.0f", quote.Symbol, quote.Volume)
	}

	// 这里可以调用LLM的成交量分析方法
	return fmt.Sprintf("股票 %s 出现成交量异动，需要关注", quote.Symbol)
}

// generatePriceLevelAnalysis 生成价位分析
func (e *MonitorEngine) generatePriceLevelAnalysis(quote *model.StockQuote, threshold float64) string {
	if e.llmClient == nil {
		return fmt.Sprintf("股票 %s 突破关键价位 %.2f，当前价格 %.2f",
			quote.Symbol, threshold, quote.Price)
	}

	// 这里可以调用LLM的价位分析方法
	return fmt.Sprintf("股票 %s 价格突破重要关口", quote.Symbol)
}

// isSubscriptionValid 检查订阅是否有效
func (e *MonitorEngine) isSubscriptionValid(subscription *model.Subscription) bool {
	// 检查订阅状态
	if subscription.Status != model.SubscriptionStatusActive {
		return false
	}

	// 检查是否过期
	if subscription.ExpiresAt != nil && time.Now().After(*subscription.ExpiresAt) {
		log.Printf("订阅 %s 已过期", subscription.ID)
		subscription.Status = model.SubscriptionStatusExpired
		return false
	}

	return true
}

// AddSubscription 添加新的订阅
func (e *MonitorEngine) AddSubscription(subscription *model.Subscription) error {
	if subscription == nil {
		return fmt.Errorf("订阅对象不能为空")
	}

	// 验证订阅的基本信息
	if err := e.validateSubscription(subscription); err != nil {
		return fmt.Errorf("订阅验证失败: %w", err)
	}

	// 添加到规则引擎
	e.ruleEngine.AddSubscription(subscription)

	log.Printf("成功添加订阅 %s，包含 %d 个股票", subscription.ID, len(subscription.Symbols))
	return nil
}

// validateSubscription 验证订阅信息
func (e *MonitorEngine) validateSubscription(subscription *model.Subscription) error {
	if subscription.UserID == "" {
		return fmt.Errorf("用户ID不能为空")
	}
	if len(subscription.Symbols) == 0 {
		return fmt.Errorf("订阅必须包含至少一个股票")
	}
	if len(subscription.AlertRules) == 0 {
		return fmt.Errorf("订阅必须包含至少一个异动规则")
	}
	return nil
}

// UpdateSubscription 更新订阅信息
func (e *MonitorEngine) UpdateSubscription(subscription *model.Subscription) error {
	if subscription == nil {
		return fmt.Errorf("订阅对象不能为空")
	}

	// 验证订阅是否存在
	existingSub, exists := e.ruleEngine.GetSubscription(subscription.ID)
	if !exists {
		return fmt.Errorf("订阅 %s 不存在", subscription.ID)
	}

	// 验证更新的订阅信息
	if err := e.validateSubscription(subscription); err != nil {
		return fmt.Errorf("订阅验证失败: %w", err)
	}

	// 移除旧的订阅
	e.ruleEngine.RemoveSubscription(existingSub.ID)

	// 添加新的订阅
	e.ruleEngine.AddSubscription(subscription)

	log.Printf("成功更新订阅 %s", subscription.ID)
	return nil
}

// RemoveSubscription 移除订阅
func (e *MonitorEngine) RemoveSubscription(subscriptionID string) error {
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID不能为空")
	}

	// 验证订阅是否存在
	if _, exists := e.ruleEngine.GetSubscription(subscriptionID); !exists {
		return fmt.Errorf("订阅 %s 不存在", subscriptionID)
	}

	// 从规则引擎中移除订阅
	e.ruleEngine.RemoveSubscription(subscriptionID)

	log.Printf("成功移除订阅 %s", subscriptionID)
	return nil
}
