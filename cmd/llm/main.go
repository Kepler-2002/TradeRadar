package main

import (
	"TradeRadar/pkg/model"
	"log"
	"os"
	"os/signal"
	"syscall"

	"TradeRadar/pkg/config"
	"TradeRadar/pkg/llm"
	"TradeRadar/pkg/messaging"
)

func main() {
	log.Println("启动LLM服务...")

	// 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/dev/app.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v\n", err)
	}

	// 创建LLM客户端
	llmClient := llm.NewLLMClient(
		cfg.LLM.APIURL,
		cfg.LLM.APIKey,
		cfg.LLM.ModelName,
	)

	// 连接NATS
	natsClient, err := messaging.NewNATSClient(
		cfg.NATS.URL,
		cfg.NATS.ClusterID,
		cfg.NATS.ClientID+"-llm",
	)
	if err != nil {
		log.Fatalf("连接NATS失败: %v\n", err)
	}
	defer natsClient.Close()

	// 订阅行情数据，生成分析
	_, err = natsClient.SubscribeQuotes(func(quote model.StockQuote) {
		// 只对大幅波动的股票生成分析
		if abs(quote.ChangePercent) >= 3.0 {
			analysis, err := natsClient.GenerateStockAnalysis(quote, llmClient)
			if err != nil {
				log.Printf("生成股票分析失败: %v\n", err)
				return
			}

			log.Printf("股票分析 - %s (%s): %s\n", quote.Symbol, quote.Name, analysis)
		}
	})
	if err != nil {
		log.Fatalf("订阅行情数据失败: %v\n", err)
	}

	log.Println("LLM服务已启动，等待行情数据...")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("接收到中断信号，正在关闭服务...")
}

// abs 返回绝对值
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
