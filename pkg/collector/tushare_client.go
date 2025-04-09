package collector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TushareClient Tushare API客户端
type TushareClient struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

// TushareRequest Tushare API请求结构
type TushareRequest struct {
	APIName string      `json:"api_name"`
	Token   string      `json:"token"`
	Params  interface{} `json:"params,omitempty"`
	Fields  string      `json:"fields,omitempty"`
}

// TushareResponse Tushare API响应结构
type TushareResponse struct {
	RequestID string `json:"request_id"`
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	Data      struct {
		Fields []string        `json:"fields"`
		Items  [][]interface{} `json:"items"`
	} `json:"data"`
}

// NewTushareClient 创建新的Tushare客户端
func NewTushareClient(apiKey, baseURL string) *TushareClient {
	return &TushareClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Execute 执行Tushare API请求
func (c *TushareClient) Execute(apiName string, params interface{}, fields string) (*TushareResponse, error) {
	req := TushareRequest{
		APIName: apiName,
		Token:   c.APIKey,
		Params:  params,
		Fields:  fields,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("执行HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回非200状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	var tushareResp TushareResponse
	if err := json.Unmarshal(body, &tushareResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if tushareResp.Code != 0 {
		return nil, fmt.Errorf("API返回错误: %s", tushareResp.Msg)
	}

	return &tushareResp, nil
}

// GetStockBasic 获取股票基本信息
func (c *TushareClient) GetStockBasic(params map[string]interface{}) (*TushareResponse, error) {
	fields := "ts_code,symbol,name,area,industry,list_date"
	return c.Execute("stock_basic", params, fields)
}

// GetDailyQuotes 获取日线行情
func (c *TushareClient) GetDailyQuotes(params map[string]interface{}) (*TushareResponse, error) {
	fields := "ts_code,trade_date,open,high,low,close,pre_close,change,pct_chg,vol,amount"
	return c.Execute("daily", params, fields)
}

// GetRealtimeQuotes 获取实时行情
func (c *TushareClient) GetRealtimeQuotes(params map[string]interface{}) (*TushareResponse, error) {
	// 注意：Tushare实际上没有提供实时行情API，这里是模拟
	fields := "ts_code,trade_time,open,high,low,close,pre_close,change,pct_chg,vol,amount"
	return c.Execute("quotes", params, fields)
}