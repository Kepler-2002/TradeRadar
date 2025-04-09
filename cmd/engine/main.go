package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dewei/TradeRadar/pkg/config"
	"github.com/dewei/TradeRadar/pkg/database"
	"github.com/dewei/TradeRadar/pkg/engine"
	"github.com/dewei/TradeRadar/pkg/messaging"
	"github.com/dewei/TradeRadar/pkg/model"
	"github.com/dewei/TradeRadar/pkg/scheduler"
)

func main() {
	log.Println("启动异动检测引擎...")

	// 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = config.GetDefaultConfigPath()
	}
	
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v\n", err)
	}
	
	// 连接数据库
	db, err := database.NewTimescaleDB(cfg)
	if err != nil {
		log.Fatalf("连接数据库失败: %v\n", err)
	}
	defer db.Close()
	
	// 连接NATS
	natsClient, err := messaging.NewNATSClient(
		cfg.NATS.URL,
		cfg.NATS.ClusterID,
		cfg.NATS.ClientID+"-engine",
	)
	if err != nil {
		log.Fatalf("连接NATS失败: %v\n", err)
	}
	defer natsClient.Close()
	
	// 创建告警通道
	alertChan := make(chan model.AlertEvent, 100)
	
	// 创建规则引擎
	ruleEngine := engine.NewRuleEngine(alertChan)
	
	// 订阅行情数据
	quoteSub, err := natsClient.SubscribeQuotes(func(quote model.StockQuote) {
		// 评估行情是否触发规则
		ruleEngine.Evaluate(quote)
		
		// 保存行情数据到数据库
		if err := db.SaveQuote(quote.Symbol, quote.Price, quote.Volume, quote.Timestamp); err != nil {
			log.Printf("保存行情数据失败: %v\n", err)
		}
	})
	if err != nil {
		log.Fatalf("订阅行情数据失败: %v\n", err)
	}
	defer quoteSub.Close()
	
	// 启动告警处理
	go processAlerts(alertChan, db, natsClient)
	
	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	log.Println("正在关闭异动检测引擎...")
}

// 处理告警事件
func processAlerts(alertChan <-chan model.AlertEvent, db *database.TimescaleDB, natsClient *messaging.NATSClient) {
	for alert := range alertChan {
		log.Printf("检测到异动: 股票=%s, 类型=%d, 强度=%.2f\n", 
			alert.Symbol, alert.Type, alert.Intensity)
		
		// 保存异动事件到数据库
		if err := db.SaveAlert(alert.Symbol, int(alert.Type), alert.Intensity, alert.Timestamp); err != nil {
			log.Printf("保存异动事件失败: %v\n", err)
		}
		
		// 发布异动事件到NATS
		if err := natsClient.PublishAlert(alert); err != nil {
			log.Printf("发布异动事件失败: %v\n", err)
		}
	}
}