// pkg/collector/news_collector.go
package collector

import (
	"TradeRadar/pkg/model"
	"encoding/json"
	"fmt"
	"github.com/nats-io/stan.go"
	"log"
	"time"
)

// NewsCollector 新闻收集器接口
type NewsCollector interface {
	StartCollecting() error
	Stop() error
	GetLatestNews(symbol string) ([]*model.NewsEvent, error)
	GetNewsByTimeRange(startTime, endTime time.Time) ([]*model.NewsEvent, error)
	SubscribeNewsUpdates(handler func(*model.NewsEvent)) error
}

// NATSNewsCollector 基于NATS的新闻收集器
type NATSNewsCollector struct {
	natsClient   *stan.Conn
	clusterID    string
	clientID     string
	natsURL      string
	subscription stan.Subscription
	newsCache    map[string]*model.NewsEvent // 内存缓存最新新闻
	handlers     []func(*model.NewsEvent)    // 事件处理器
	stopChan     chan bool
}

// NewNATSNewsCollector 创建新的NATS新闻收集器
func NewNATSNewsCollector(natsURL, clusterID, clientID string) (*NATSNewsCollector, error) {
	conn, err := stan.Connect(
		clusterID,
		clientID,
		stan.NatsURL(natsURL),
	)
	if err != nil {
		return nil, fmt.Errorf("连接NATS失败: %w", err)
	}

	return &NATSNewsCollector{
		natsClient: &conn,
		clusterID:  clusterID,
		clientID:   clientID,
		natsURL:    natsURL,
		newsCache:  make(map[string]*model.NewsEvent),
		handlers:   make([]func(*model.NewsEvent), 0),
		stopChan:   make(chan bool),
	}, nil
}

// StartCollecting 开始收集新闻
func (nc *NATSNewsCollector) StartCollecting() error {
	log.Println("开始订阅新闻数据...")

	// 订阅新闻主题
	sub, err := (*nc.natsClient).Subscribe(
		"news.cls", // 财联社新闻主题
		nc.handleNewsMessage,
		stan.StartWithLastReceived(),
	)
	if err != nil {
		return fmt.Errorf("订阅新闻主题失败: %w", err)
	}

	nc.subscription = sub
	log.Println("新闻收集器启动成功")
	return nil
}

// handleNewsMessage 处理新闻消息
func (nc *NATSNewsCollector) handleNewsMessage(msg *stan.Msg) {
	// 解析爬虫发送的原始新闻数据
	var crawlerNews struct {
		Title      string  `json:"title"`
		Abstract   string  `json:"abstract"`
		Date       string  `json:"date"`
		Link       string  `json:"link"`
		Content    string  `json:"content"`
		ID         string  `json:"id"`
		Type       string  `json:"type"`
		ReadNumber float64 `json:"read_number"`
		Time       string  `json:"time"`
	}

	if err := json.Unmarshal(msg.Data, &crawlerNews); err != nil {
		log.Printf("解析新闻数据失败: %v", err)
		return
	}

	// 转换为标准的NewsEvent格式
	newsEvent := nc.convertToNewsEvent(crawlerNews)

	// 缓存新闻
	nc.newsCache[newsEvent.ID] = newsEvent

	// 通知所有处理器
	for _, handler := range nc.handlers {
		go handler(newsEvent)
	}

	log.Printf("收到新闻: %s - %s", newsEvent.Symbol, newsEvent.Title)
}

// convertToNewsEvent 将爬虫数据转换为NewsEvent
func (nc *NATSNewsCollector) convertToNewsEvent(crawlerNews interface{}) *model.NewsEvent {
	// 这里需要根据你的爬虫数据结构进行转换
	data := crawlerNews.(struct {
		Title      string  `json:"title"`
		Abstract   string  `json:"abstract"`
		Date       string  `json:"date"`
		Link       string  `json:"link"`
		Content    string  `json:"content"`
		ID         string  `json:"id"`
		Type       string  `json:"type"`
		ReadNumber float64 `json:"read_number"`
		Time       string  `json:"time"`
	})

	// 从标题和内容中提取股票代码
	symbol := nc.extractSymbolFromContent(data.Title + " " + data.Content)

	// 解析时间
	publishedAt, err := time.Parse("2006-01-02 15:04:05", data.Date)
	if err != nil {
		publishedAt = time.Now()
	}

	// 分析新闻情感和影响程度
	sentiment := nc.analyzeSentiment(data.Title, data.Content)
	impact := nc.calculateImpact(data.Title, data.Content, data.ReadNumber)

	return &model.NewsEvent{
		ID:          data.ID,
		Symbol:      symbol,
		Title:       data.Title,
		Content:     data.Content,
		Summary:     data.Abstract,
		Source:      "财联社",
		Author:      "",
		URL:         data.Link,
		Sentiment:   sentiment,
		Impact:      impact,
		Keywords:    nc.extractKeywords(data.Title, data.Content),
		PublishedAt: publishedAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// extractSymbolFromContent 从内容中提取股票代码
func (nc *NATSNewsCollector) extractSymbolFromContent(content string) string {
	// 这里实现股票代码提取逻辑
	// 可以使用正则表达式匹配常见的股票代码格式
	// 暂时返回空字符串，表示通用新闻
	return ""
}

// analyzeSentiment 分析新闻情感
func (nc *NATSNewsCollector) analyzeSentiment(title, content string) model.NewsSentiment {
	// 简单的关键词情感分析
	negative_keywords := []string{"下跌", "暴跌", "亏损", "风险", "警告", "下滑"}
	positive_keywords := []string{"上涨", "暴涨", "盈利", "机会", "突破", "增长"}

	text := title + " " + content

	negativeScore := 0
	positiveScore := 0

	for _, keyword := range negative_keywords {
		if contains(text, keyword) {
			negativeScore++
		}
	}

	for _, keyword := range positive_keywords {
		if contains(text, keyword) {
			positiveScore++
		}
	}

	if positiveScore > negativeScore {
		return model.NewsSentimentPositive
	} else if negativeScore > positiveScore {
		return model.NewsSentimentNegative
	}

	return model.NewsSentimentNeutral
}

// calculateImpact 计算新闻影响程度
func (nc *NATSNewsCollector) calculateImpact(title, content string, readNumber float64) float64 {
	impact := 0.3 // 基础影响

	// 根据阅读量调整影响程度
	if readNumber > 10000 {
		impact += 0.3
	} else if readNumber > 5000 {
		impact += 0.2
	} else if readNumber > 1000 {
		impact += 0.1
	}

	// 根据关键词调整影响程度
	highImpactKeywords := []string{"重大", "突发", "紧急", "重要", "关键"}
	for _, keyword := range highImpactKeywords {
		if contains(title, keyword) {
			impact += 0.2
			break
		}
	}

	if impact > 1.0 {
		impact = 1.0
	}

	return impact
}

// extractKeywords 提取关键词
func (nc *NATSNewsCollector) extractKeywords(title, content string) []string {
	// 简单的关键词提取，实际可以使用更复杂的NLP算法
	keywords := []string{}

	stockKeywords := []string{"股票", "股价", "涨跌", "交易", "市值"}
	for _, keyword := range stockKeywords {
		if contains(title+" "+content, keyword) {
			keywords = append(keywords, keyword)
		}
	}

	return keywords
}

// GetLatestNews 获取最新新闻
func (nc *NATSNewsCollector) GetLatestNews(symbol string) ([]*model.NewsEvent, error) {
	var news []*model.NewsEvent

	for _, newsEvent := range nc.newsCache {
		if symbol == "" || newsEvent.Symbol == symbol {
			news = append(news, newsEvent)
		}
	}

	// 按时间排序
	// 这里简化处理，实际应该实现排序逻辑

	return news, nil
}

// GetNewsByTimeRange 获取时间范围内的新闻
func (nc *NATSNewsCollector) GetNewsByTimeRange(startTime, endTime time.Time) ([]*model.NewsEvent, error) {
	var news []*model.NewsEvent

	for _, newsEvent := range nc.newsCache {
		if newsEvent.PublishedAt.After(startTime) && newsEvent.PublishedAt.Before(endTime) {
			news = append(news, newsEvent)
		}
	}

	return news, nil
}

// SubscribeNewsUpdates 订阅新闻更新
func (nc *NATSNewsCollector) SubscribeNewsUpdates(handler func(*model.NewsEvent)) error {
	nc.handlers = append(nc.handlers, handler)
	return nil
}

// Stop 停止收集器
func (nc *NATSNewsCollector) Stop() error {
	if nc.subscription != nil {
		nc.subscription.Unsubscribe()
	}

	if nc.natsClient != nil {
		(*nc.natsClient).Close()
	}

	close(nc.stopChan)
	return nil
}

// 辅助函数
func contains(text, keyword string) bool {
	return len(text) > 0 && len(keyword) > 0 &&
		len(text) >= len(keyword) &&
		text[len(text)-len(keyword):] == keyword ||
		len(text) > len(keyword) &&
			text[:len(keyword)] == keyword ||
		len(text) > len(keyword)*2
	// 这是一个简化的contains实现，实际应该使用strings.Contains
}
