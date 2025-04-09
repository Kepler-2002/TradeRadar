package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"TradeRadar/pkg/collector"
	"TradeRadar/pkg/config"
	"TradeRadar/pkg/messaging"
)

func main() {
	log.Println("启动数据采集服务...")

	// 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = config.GetDefaultConfigPath()
	}
	
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v\n", err)
	}
	
	// 创建Tushare适配器
	tushare := collector.NewTushareAdapter(
		cfg.DataSources.Tushare.APIKey,
		cfg.DataSources.Tushare.BaseURL,
	)
	
	// 连接NATS
	natsClient, err := messaging.NewNATSClient(
		cfg.NATS.URL,
		cfg.NATS.ClusterID,
		cfg.NATS.ClientID+"-collector",
	)
	if err != nil {
		log.Fatalf("连接NATS失败: %v\n", err)
	}
	defer natsClient.Close()
	
	// 获取股票代码列表
	stockCodes := getStockCodes()
	
	// 启动定时采集
	stopChan := make(chan struct{})
	go startCollection(tushare, natsClient, stockCodes, stopChan)
	
	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	log.Println("正在关闭数据采集服务...")
	close(stopChan)
	time.Sleep(1 * time.Second) // 等待采集任务完成
}

// 获取股票代码列表
func getStockCodes() []string {
	// 实际项目中应该从配置或数据库获取
	codesStr := os.Getenv("STOCK_CODES")
	if codesStr == "" {
		codesStr = "000001.SZ,600000.SH,601318.SH" // 默认股票列表
	}
	
	return strings.Split(codesStr, ",")
}

// 启动定时采集
func startCollection(
	fetcher collector.QuoteFetcher,
	natsClient *messaging.NATSClient,
	stockCodes []string,
	stopChan <-chan struct{},
) {
	ticker := time.NewTicker(5 * time.Second) // 每5秒采集一次
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// 采集行情数据
			quotes, err := fetcher.FetchRealtime(stockCodes)
			if err != nil {
				log.Printf("采集行情数据失败: %v\n", err)
				continue
			}
			
			// 发布行情数据
			for _, quote := range quotes {
				if err := natsClient.PublishQuote(quote); err != nil {
					log.Printf("发布行情数据失败: %v\n", err)
				}
			}
			
			log.Printf("已采集并发布%d条行情数据\n", len(quotes))
			
		case <-stopChan:
			log.Println("停止采集任务")
			return
		}
	}
}