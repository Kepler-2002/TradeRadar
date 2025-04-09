package collector

import (
	"fmt"
	"strconv"
	"time"

	"TradeRadar/pkg/model"
)

// TushareAdapter Tushare数据源适配器
type TushareAdapter struct {
	client *TushareClient
}

// NewTushareAdapter 创建Tushare适配器
func NewTushareAdapter(apiKey, baseURL string) *TushareAdapter {
	return &TushareAdapter{
		client: NewTushareClient(apiKey, baseURL),
	}
}

// FetchRealtime 获取实时行情数据
func (t *TushareAdapter) FetchRealtime(codes []string) ([]model.StockQuote, error) {
	if len(codes) == 0 {
		return nil, fmt.Errorf("股票代码列表不能为空")
	}

	// 构建请求参数
	params := map[string]interface{}{
		"ts_code": joinCodes(codes),
	}

	// 调用API获取数据
	resp, err := t.client.GetRealtimeQuotes(params)
	if err != nil {
		return nil, fmt.Errorf("获取实时行情失败: %w", err)
	}

	// 转换为统一数据模型
	return t.normalizeQuotes(resp)
}

// FetchDaily 获取日线行情数据
func (t *TushareAdapter) FetchDaily(codes []string, date string) ([]model.StockQuote, error) {
	if len(codes) == 0 {
		return nil, fmt.Errorf("股票代码列表不能为空")
	}

	// 构建请求参数
	params := map[string]interface{}{
		"ts_code":    joinCodes(codes),
		"trade_date": date,
	}

	// 调用API获取数据
	resp, err := t.client.GetDailyQuotes(params)
	if err != nil {
		return nil, fmt.Errorf("获取日线行情失败: %w", err)
	}

	// 转换为统一数据模型
	return t.normalizeQuotes(resp)
}

// normalizeQuotes 将Tushare响应转换为统一数据模型
func (t *TushareAdapter) normalizeQuotes(resp *TushareResponse) ([]model.StockQuote, error) {
	result := make([]model.StockQuote, 0, len(resp.Data.Items))
	
	// 获取字段索引
	fieldIndices := make(map[string]int)
	for i, field := range resp.Data.Fields {
		fieldIndices[field] = i
	}
	
	// 检查必要字段是否存在
	requiredFields := []string{"ts_code", "close"}
	for _, field := range requiredFields {
		if _, exists := fieldIndices[field]; !exists {
			return nil, fmt.Errorf("响应中缺少必要字段: %s", field)
		}
	}
	
	// 转换数据
	for _, item := range resp.Data.Items {
		quote := model.StockQuote{
			Symbol:    item[fieldIndices["ts_code"]].(string),
			Timestamp: time.Now(),
		}
		
		// 设置价格
		if price, err := toFloat64(item[fieldIndices["close"]]); err == nil {
			quote.Price = price
		}
		
		// 设置开盘价（如果存在）
		if idx, exists := fieldIndices["open"]; exists {
			if open, err := toFloat64(item[idx]); err == nil {
				quote.Open = open
			}
		}
		
		// 设置最高价（如果存在）
		if idx, exists := fieldIndices["high"]; exists {
			if high, err := toFloat64(item[idx]); err == nil {
				quote.High = high
			}
		}
		
		// 设置最低价（如果存在）
		if idx, exists := fieldIndices["low"]; exists {
			if low, err := toFloat64(item[idx]); err == nil {
				quote.Low = low
			}
		}
		
		// 设置成交量（如果存在）
		if idx, exists := fieldIndices["vol"]; exists {
			if vol, err := toFloat64(item[idx]); err == nil {
				quote.Volume = vol
			}
		}
		
		// 设置涨跌幅（如果存在）
		if idx, exists := fieldIndices["pct_chg"]; exists {
			if pctChg, err := toFloat64(item[idx]); err == nil {
				quote.ChangePercent = pctChg
			}
		}
		
		// 设置股票名称（如果存在）
		if idx, exists := fieldIndices["name"]; exists {
			if name, ok := item[idx].(string); ok {
				quote.Name = name
			}
		}
		
		result = append(result, quote)
	}
	
	return result, nil
}

// joinCodes 将股票代码列表连接为字符串
func joinCodes(codes []string) string {
	if len(codes) == 1 {
		return codes[0]
	}
	
	result := ""
	for i, code := range codes {
		if i > 0 {
			result += ","
		}
		result += code
	}
	return result
}

// toFloat64 将接口类型转换为float64
func toFloat64(v interface{}) (float64, error) {
	switch value := v.(type) {
	case float64:
		return value, nil
	case float32:
		return float64(value), nil
	case int:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case string:
		return strconv.ParseFloat(value, 64)
	default:
		return 0, fmt.Errorf("无法转换为float64: %v", v)
	}
}