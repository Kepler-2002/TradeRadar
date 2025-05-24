package main

import (
	"TradeRadar/pkg/model"
	"fmt"
	"log"
	"os"

	"TradeRadar/pkg/config"
	"TradeRadar/pkg/llm"
)

func main() {
	log.Println("开始验证大模型接入...")

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

	// 测试简单问题
	testSimpleQuestion(llmClient)

	// 测试股票分析
	testStockAnalysis(llmClient)

	// 测试异动解释
	testAlertExplanation(llmClient)

	log.Println("大模型验证完成")
}

func testSimpleQuestion(client *llm.LLMClient) {
	log.Println("测试简单问题...")

	messages := []llm.Message{
		{Role: "system", Content: "你是人工智能助手."},
		{Role: "user", Content: "常见的十字花科植物有哪些？"},
	}

	response, err := client.Chat(messages)
	if err != nil {
		log.Printf("测试失败: %v\n", err)
		return
	}

	fmt.Println("\n===== 简单问题测试结果 =====")
	fmt.Println(response)
	fmt.Println("===========================\n")

	log.Println("简单问题测试成功")
}

func testStockAnalysis(client *llm.LLMClient) {
	log.Println("测试股票分析...")

	analysis, err := client.GenerateAnalysis(
		"600000.SH",
		"浦发银行",
		12.34,
		2.5,
	)
	if err != nil {
		log.Printf("测试失败: %v\n", err)
		return
	}

	fmt.Println("\n===== 股票分析测试结果 =====")
	fmt.Println(analysis)
	fmt.Println("===========================\n")

	log.Println("股票分析测试成功")
}

func testAlertExplanation(client *llm.LLMClient) {
	log.Println("测试异动解释...")

	explanation, err := client.GenerateAlertExplanation(
		"异动警报",
		"000001.SZ",
		"平安银行",
		model.AlertType(1),
	)
	if err != nil {
		log.Printf("测试失败: %v\n", err)
		return
	}

	fmt.Println("\n===== 异动解释测试结果 =====")
	fmt.Println(explanation)
	fmt.Println("===========================\n")

	log.Println("异动解释测试成功")
}
