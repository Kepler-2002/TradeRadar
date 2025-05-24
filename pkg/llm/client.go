package llm

import (
	"TradeRadar/pkg/model"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// LLMClient 大模型客户端
type LLMClient struct {
	apiURL    string
	apiKey    string
	modelName string
	client    *http.Client
}

// Message 表示对话中的一条消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 表示聊天请求
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// ChatResponse 表示聊天响应
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// NewLLMClient 创建新的大模型客户端
func NewLLMClient(apiURL, apiKey, modelName string) *LLMClient {
	return &LLMClient{
		apiURL:    apiURL,
		apiKey:    apiKey,
		modelName: modelName,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Chat 发送聊天请求并获取响应
func (c *LLMClient) Chat(messages []Message) (string, error) {
	// 构建请求
	reqBody := ChatRequest{
		Model:    c.modelName,
		Messages: messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", c.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API返回错误: %s", string(body))
	}

	// 解析响应
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查是否有响应内容
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API返回空响应")
	}

	// 返回响应内容
	return chatResp.Choices[0].Message.Content, nil
}

// GenerateAnalysis 生成股票分析
func (c *LLMClient) GenerateAnalysis(symbol string, name string, price float64, changePercent float64) (string, error) {
	// 构建系统提示
	systemPrompt := "你是一位专业的股票分析师，请根据提供的股票信息进行简要分析。"

	// 构建用户提示
	userPrompt := fmt.Sprintf("请对以下股票进行简要分析：\n股票代码：%s\n股票名称：%s\n当前价格：%.2f\n涨跌幅：%.2f%%",
		symbol, name, price, changePercent)

	// 构建消息
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// 发送请求并获取响应
	return c.Chat(messages)
}

// GenerateAlertExplanation 生成异动解释
func (c *LLMClient) GenerateAlertExplanation(alert string, symbol string, name string, alertType model.AlertType) (string, error) {
	// 构建系统提示
	systemPrompt := "你是一位专业的股票分析师，请解释股票异动的可能原因。"

	// 构建用户提示
	userPrompt := fmt.Sprintf("股票%s（%s）出现了%s异动，请简要分析可能的原因。",
		symbol, name, alertType)

	// 构建消息
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// 发送请求并获取响应
	return c.Chat(messages)
}
