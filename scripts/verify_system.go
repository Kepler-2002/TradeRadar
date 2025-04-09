package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"TradeRadar/pkg/collector"
	"TradeRadar/pkg/config"
	"TradeRadar/pkg/engine"
	"TradeRadar/pkg/messaging"
	"TradeRadar/pkg/model"
	"TradeRadar/pkg/repository"
)

func main() {
	log.Println("开始系统验证...")

	// 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/dev/app.yaml"
	}
	
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v\n", err)
	}
	
	// 创建仓库
	repo := repository.NewRepository()
	
	// 创建告警通道和规则引擎
	alertChan := make(chan model.AlertEvent, 100)
	ruleEngine := engine.NewRuleEngine(alertChan)
	
	// 加载规则
	rules := repo.LoadActiveRules()
	ruleEngine.ReloadRules(rules)
	
	// 创建NATS客户端
	natsClient, err := messaging.NewNATSClient(
		cfg.NATS.URL,
		cfg.NATS.ClusterID,
		cfg.NATS.ClientID+"-verifier",
	)
	if err != nil {
		log.Printf("连接NATS失败: %v，跳过NATS相关测试\n", err)
	} else {
		defer natsClient.Close()
	}
	
	// 创建Tushare适配器
	tushare := collector.NewTushareAdapter(
		cfg.DataSources.Tushare.APIKey,
		cfg.DataSources.Tushare.BaseURL,
	)
	
	// 启动告警处理
	go func() {
		for alert := range alertChan {
			log.Printf("收到告警: 股票=%s, 类型=%d, 强度=%.2f\n", 
				alert.Symbol, alert.Type, alert.Intensity)
		}
	}()
	
	// 测试数据采集
	testDataCollection(tushare)
	
	// 测试规则引擎
	testRuleEngine(ruleEngine)
	
	// 测试仓库
	testRepository(repo)
	
	// 测试NATS（如果可用）
	if natsClient != nil {
		testNATS(natsClient)
	}
	
	log.Println("系统验证完成")
}

// 测试数据采集
func testDataCollection(fetcher collector.QuoteFetcher) {
	log.Println("测试数据采集...")
	
	codes := []string{"000001.SZ", "600000.SH"}
	quotes, err := fetcher.FetchRealtime(codes)
	if err != nil {
		log.Printf("数据采集失败: %v\n", err)
		return
	}
	
	log.Printf("成功获取%d条行情数据\n", len(quotes))
	for _, quote := range quotes {
		log.Printf("股票: %s, 价格: %.2f, 涨跌幅: %.2f%%\n", 
			quote.Symbol, quote.Price, quote.ChangePercent)
	}
}

// 测试规则引擎
func testRuleEngine(engine *engine.RuleEngine) {
	log.Println("测试规则引擎...")
	
	// 创建一个模拟的大幅波动行情
	quote := model.StockQuote{
		Symbol:        "000001.SZ",
		Name:          "平安银行",
		Price:         15.8,
		Open:          15.0,
		High:          16.0,
		Low:           14.9,
		Volume:        2000000,
		ChangePercent: 6.5, // 超过5%的阈值
		Timestamp:     time.Now(),
	}
	
	// 评估规则
	engine.Evaluate(quote)
	log.Println("规则评估完成，检查是否有告警输出")
	
	// 等待一下，确保告警处理完成
	time.Sleep(1 * time.Second)
}

// 测试仓库
func testRepository(repo *repository.Repository) {
	log.Println("测试数据仓库...")
	
	// 测试保存订阅
	userID := "test_user_001"
	symbols := []string{"000001.SZ", "600000.SH"}
	rules := []model.DetectionRule{
		{
			Type:      model.AlertPriceVolatility,
			Threshold: 4.0,
		},
	}
	
	err := repo.SaveSubscription(userID, symbols, rules)
	if err != nil {
		log.Printf("保存订阅失败: %v\n", err)
	} else {
		log.Println("保存订阅成功")
	}
	
	// 测试获取订阅
	subs := repo.GetSubscriptions("000001.SZ")
	log.Printf("获取到%d个订阅\n", len(subs))
	
	// 测试保存告警
	alert := model.AlertEvent{
		Symbol:    "000001.SZ",
		Type:      model.AlertPriceVolatility,
		Intensity: 1.3,
		Timestamp: time.Now(),
	}
	
	err = repo.SaveAlert(alert)
	if err != nil {
		log.Printf("保存告警失败: %v\n", err)
	} else {
		log.Println("保存告警成功")
	}
	
	// 测试获取告警历史
	alerts, err := repo.GetAlertHistory("000001.SZ", 10)
	if err != nil {
		log.Printf("获取告警历史失败: %v\n", err)
	} else {
		log.Printf("获取到%d条告警历史\n", len(alerts))
	}
}

// 测试NATS
func testNATS(client *messaging.NATSClient) {
	log.Println("测试NATS消息队列...")
	
	// 创建一个测试行情
	quote := model.StockQuote{
		Symbol:        "000001.SZ",
		Name:          "平安银行",
		Price:         15.8,
		ChangePercent: 2.5,
		Volume:        1000000,
		Timestamp:     time.Now(),
	}
	
	// 发布行情
	err := client.PublishQuote(quote)
	if err != nil {
		log.Printf("发布行情失败: %v\n", err)
	} else {
		log.Println("发布行情成功")
	}
	
	// 创建一个测试告警
	alert := model.AlertEvent{
		Symbol:    "000001.SZ",
		Type:      model.AlertVolumeSpike,
		Intensity: 1.2,
		Timestamp: time.Now(),
	}
	
	// 发布告警
	err = client.PublishAlert(alert)
	if err != nil {
		log.Printf("发布告警失败: %v\n", err)
	} else {
		log.Println("发布告警成功")
	}
	
	// 订阅行情，验证是否能收到
	receivedQuote := false
	quoteSub, err := client.SubscribeQuotes(func(q model.StockQuote) {
		log.Printf("收到行情: %s, 价格: %.2f\n", q.Symbol, q.Price)
		receivedQuote = true
	})
	
	if err != nil {
		log.Printf("订阅行情失败: %v\n", err)
	} else {
		defer quoteSub.Close()
		
		// 再次发布一条行情
		time.Sleep(1 * time.Second)
		quote.Price = 16.0
		client.PublishQuote(quote)
		
		// 等待接收
		time.Sleep(2 * time.Second)
		
		if receivedQuote {
			log.Println("成功接收到行情数据")
		} else {
			log.Println("未接收到行情数据")
		}
	}
}