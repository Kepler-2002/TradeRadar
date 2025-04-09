package main

import (
	"log"
	"os"

	"github.com/dewei/TradeRadar/pkg/api"
	"github.com/dewei/TradeRadar/pkg/collector"
	"github.com/dewei/TradeRadar/pkg/engine"
	"github.com/dewei/TradeRadar/pkg/repository"
)

func main() {
	log.Println("启动API服务...")

	// 获取配置
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	tushareAPIKey := os.Getenv("TUSHARE_API_KEY")
	if tushareAPIKey == "" {
		tushareAPIKey = "your_api_key_here" // 默认值，实际应从配置获取
	}
	
	tushareBaseURL := os.Getenv("TUSHARE_BASE_URL")
	if tushareBaseURL == "" {
		tushareBaseURL = "https://api.tushare.pro" // 默认值
	}
	
	// 创建数据仓库
	repo := repository.NewRepository()
	
	// 创建行情获取器
	quoteFetcher := collector.NewTushareAdapter(tushareAPIKey, tushareBaseURL)
	
	// 创建告警通道
	alertChan := make(chan model.AlertEvent, 100)
	
	// 创建规则引擎
	ruleEngine := engine.NewRuleEngine(alertChan)
	
	// 启动告警处理
	go processAlerts(alertChan, repo)
	
	// 创建API处理程序
	handlers := api.NewHandlers(quoteFetcher, ruleEngine, repo)
	
	// 创建并启动服务器
	server := api.NewServer(port)
	server.SetupRoutes(handlers)
	server.Start()
}

// 处理告警事件
func processAlerts(alertChan <-chan model.AlertEvent, repo *repository.Repository) {
	for alert := range alertChan {
		log.Printf("检测到异动: 股票=%s, 类型=%d, 强度=%.2f\n", 
			alert.Symbol, alert.Type, alert.Intensity)
		
		// 保存异动事件
		if err := repo.SaveAlert(alert); err != nil {
			log.Printf("保存异动事件失败: %v\n", err)
		}
		
		// 实际项目中应该将告警发送到消息队列
	}
}