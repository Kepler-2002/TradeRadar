// pkg/collector/news_collector.go
package collector

import (
	"TradeRadar/pkg/messaging"
	"TradeRadar/pkg/model"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// NATSNewsCollector 基于NATS的新闻收集器
type NATSNewsCollector struct {
	natsClient *messaging.NATSClient
	newsCache  map[string]*model.NewsEvent // 内存缓存最新新闻
	handlers   []func(*model.NewsEvent)    // 事件处理器
	mu         sync.RWMutex                // 保护共享资源
}

// CrawlerNewsData 爬虫发送的原始新闻数据结构
type CrawlerNewsData struct {
	Title      string  `json:"title"`
	Abstract   string  `json:"abstract"`
	Date       string  `json:"date"`
	Link       string  `json:"link"`
	Content    string  `json:"content"`
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Author     string  `json:"author"`
	ReadNumber float64 `json:"read_number"`
	Time       string  `json:"time"`
}

// NewNATSNewsCollector 创建新的NATS新闻收集器
func NewNATSNewsCollector(natsClient *messaging.NATSClient) *NATSNewsCollector {
	return &NATSNewsCollector{
		natsClient: natsClient,
		newsCache:  make(map[string]*model.NewsEvent),
		handlers:   make([]func(*model.NewsEvent), 0),
	}
}

// StartCollecting 开始收集新闻
func (nc *NATSNewsCollector) StartCollecting() error {
	log.Println("开始订阅新闻数据...")

	// 订阅财联社新闻主题
	err := nc.natsClient.Subscribe(
		"NEWS_STREAM",        // Stream名称
		"news-collector",     // Consumer名称
		"news.cls",           // 过滤主题
		nc.handleNewsMessage, // 消息处理器
	)
	if err != nil {
		return fmt.Errorf("订阅新闻主题失败: %w", err)
	}

	log.Println("新闻收集器启动成功")
	return nil
}

// handleNewsMessage 处理新闻消息
func (nc *NATSNewsCollector) handleNewsMessage(data []byte) error {
	// 解析爬虫发送的原始新闻数据
	var crawlerNews CrawlerNewsData
	if err := json.Unmarshal(data, &crawlerNews); err != nil {
		return fmt.Errorf("解析新闻数据失败: %w", err)
	}

	// 验证必要字段
	if crawlerNews.ID == "" || crawlerNews.Title == "" {
		return fmt.Errorf("新闻数据缺少必要字段: ID=%s, Title=%s", crawlerNews.ID, crawlerNews.Title)
	}

	// 转换为标准的NewsEvent格式
	newsEvent := nc.convertToNewsEvent(crawlerNews)

	// 线程安全地缓存新闻
	nc.mu.Lock()
	nc.newsCache[newsEvent.ID] = newsEvent
	nc.mu.Unlock()

	// 通知所有处理器
	nc.mu.RLock()
	handlers := make([]func(*model.NewsEvent), len(nc.handlers))
	copy(handlers, nc.handlers)
	nc.mu.RUnlock()

	for _, handler := range handlers {
		go func(h func(*model.NewsEvent)) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("新闻处理器异常: %v", r)
				}
			}()
			h(newsEvent)
		}(handler)
	}

	log.Printf("收到新闻: %s - %s (来源: %s)", newsEvent.Symbol, newsEvent.Title, newsEvent.Source)
	return nil
}

// convertToNewsEvent 将爬虫数据转换为NewsEvent
func (nc *NATSNewsCollector) convertToNewsEvent(crawlerNews CrawlerNewsData) *model.NewsEvent {
	// 从标题和内容中提取股票代码
	symbol := nc.extractSymbolFromContent(crawlerNews.Title + " " + crawlerNews.Content)

	// 解析时间
	publishedAt := nc.parsePublishedTime(crawlerNews.Date)

	// 分析新闻情感和影响程度
	sentiment := nc.analyzeSentiment(crawlerNews.Title, crawlerNews.Content)
	impact := nc.calculateImpact(crawlerNews.Title, crawlerNews.Content, crawlerNews.ReadNumber)

	return &model.NewsEvent{
		ID:          crawlerNews.ID,
		Symbol:      symbol,
		Title:       crawlerNews.Title,
		Content:     crawlerNews.Content,
		Summary:     crawlerNews.Abstract,
		Source:      "财联社",
		Author:      crawlerNews.Author,
		URL:         crawlerNews.Link,
		Sentiment:   sentiment,
		Impact:      impact,
		Keywords:    nc.extractKeywords(crawlerNews.Title, crawlerNews.Content),
		PublishedAt: publishedAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// parsePublishedTime 解析发布时间
func (nc *NATSNewsCollector) parsePublishedTime(dateStr string) time.Time {
	// 尝试多种时间格式
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	log.Printf("无法解析时间格式: %s，使用当前时间", dateStr)
	return time.Now()
}

// extractSymbolFromContent 从内容中提取股票代码
func (nc *NATSNewsCollector) extractSymbolFromContent(content string) string {
	// A股代码格式：6位数字
	stockPattern := regexp.MustCompile(`(\d{6})`)
	matches := stockPattern.FindAllStringSubmatch(content, -1)

	if len(matches) > 0 {
		// 验证是否为有效的股票代码范围
		code := matches[0][1]
		if nc.isValidStockCode(code) {
			return code
		}
	}

	// 如果没找到具体股票代码，根据内容类型返回分类
	if strings.Contains(content, "A股") {
		return "A股"
	}
	if strings.Contains(content, "港股") {
		return "港股"
	}
	if strings.Contains(content, "美股") {
		return "美股"
	}

	return "通用" // 通用新闻
}

// isValidStockCode 验证是否为有效的股票代码
func (nc *NATSNewsCollector) isValidStockCode(code string) bool {
	if len(code) != 6 {
		return false
	}

	// A股主要交易所代码范围
	// 上交所：60xxxx, 68xxxx
	// 深交所：00xxxx, 30xxxx
	if strings.HasPrefix(code, "60") || strings.HasPrefix(code, "68") ||
		strings.HasPrefix(code, "00") || strings.HasPrefix(code, "30") {
		return true
	}

	return false
}

// analyzeSentiment 分析新闻情感
func (nc *NATSNewsCollector) analyzeSentiment(title, content string) model.NewsSentiment {
	negative_keywords := []string{
		"下跌", "暴跌", "亏损", "风险", "警告", "下滑", "跌幅", "利空",
		"失败", "困难", "问题", "危机", "担忧", "恶化", "下降",
	}
	positive_keywords := []string{
		"上涨", "暴涨", "盈利", "机会", "突破", "增长", "涨幅", "利好",
		"成功", "机遇", "收益", "优势", "改善", "提升", "上升",
	}

	text := strings.ToLower(title + " " + content)
	negativeScore := 0
	positiveScore := 0

	for _, keyword := range negative_keywords {
		if strings.Contains(text, keyword) {
			negativeScore++
		}
	}

	for _, keyword := range positive_keywords {
		if strings.Contains(text, keyword) {
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
	switch {
	case readNumber > 100000:
		impact += 0.5
	case readNumber > 50000:
		impact += 0.4
	case readNumber > 10000:
		impact += 0.3
	case readNumber > 5000:
		impact += 0.2
	case readNumber > 1000:
		impact += 0.1
	}

	// 根据关键词调整影响程度
	highImpactKeywords := []string{
		"重大", "突发", "紧急", "重要", "关键", "重磅", "首次", "历史性",
		"突破", "创新高", "跌停", "涨停", "停牌", "复牌",
	}

	text := title + " " + content
	for _, keyword := range highImpactKeywords {
		if strings.Contains(text, keyword) {
			impact += 0.2
			break
		}
	}

	// 限制最大值
	if impact > 1.0 {
		impact = 1.0
	}

	return impact
}

// extractKeywords 提取关键词
func (nc *NATSNewsCollector) extractKeywords(title, content string) []string {
	keywords := []string{}
	text := title + " " + content

	// 股票相关关键词
	stockKeywords := []string{
		"股票", "股价", "涨跌", "交易", "市值", "成交量", "换手率",
		"PE", "PB", "ROE", "EPS", "分红", "业绩", "财报",
	}

	// 行业关键词
	industryKeywords := []string{
		"科技", "金融", "医药", "地产", "汽车", "消费", "能源",
		"AI", "人工智能", "芯片", "新能源", "5G", "区块链",
	}

	// 政策关键词
	policyKeywords := []string{
		"政策", "监管", "法规", "央行", "降准", "降息", "IPO",
		"注册制", "退市", "并购", "重组",
	}

	allKeywords := append(append(stockKeywords, industryKeywords...), policyKeywords...)

	for _, keyword := range allKeywords {
		if strings.Contains(text, keyword) {
			keywords = append(keywords, keyword)
		}
	}

	return keywords
}

// GetLatestNews 获取最新新闻
func (nc *NATSNewsCollector) GetLatestNews(symbol string) ([]*model.NewsEvent, error) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	var news []*model.NewsEvent
	for _, newsEvent := range nc.newsCache {
		if symbol == "" || newsEvent.Symbol == symbol {
			news = append(news, newsEvent)
		}
	}

	// 按时间排序（最新的在前）
	sort.Slice(news, func(i, j int) bool {
		return news[i].PublishedAt.After(news[j].PublishedAt)
	})

	return news, nil
}

// GetNewsByTimeRange 获取时间范围内的新闻
func (nc *NATSNewsCollector) GetNewsByTimeRange(startTime, endTime time.Time) ([]*model.NewsEvent, error) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	var news []*model.NewsEvent
	for _, newsEvent := range nc.newsCache {
		if newsEvent.PublishedAt.After(startTime) && newsEvent.PublishedAt.Before(endTime) {
			news = append(news, newsEvent)
		}
	}

	// 按时间排序
	sort.Slice(news, func(i, j int) bool {
		return news[i].PublishedAt.After(news[j].PublishedAt)
	})

	return news, nil
}

// SubscribeNewsUpdates 订阅新闻更新
func (nc *NATSNewsCollector) SubscribeNewsUpdates(handler func(*model.NewsEvent)) error {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.handlers = append(nc.handlers, handler)
	log.Printf("新增新闻更新处理器，当前处理器数量: %d", len(nc.handlers))
	return nil
}

// Stop 停止收集器
func (nc *NATSNewsCollector) Stop() error {
	log.Println("正在停止新闻收集器...")

	// 删除消费者
	if err := nc.natsClient.DeleteConsumer("NEWS_STREAM", "news-collector"); err != nil {
		log.Printf("删除新闻消费者失败: %v", err)
	}

	// 清理缓存
	nc.mu.Lock()
	nc.newsCache = make(map[string]*model.NewsEvent)
	nc.handlers = make([]func(*model.NewsEvent), 0)
	nc.mu.Unlock()

	log.Println("新闻收集器已停止")
	return nil
}
