package collector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	// 移除未使用的包
	"strings"
	"time"

	"TradeRadar/pkg/model"
)

// AKShareAdapter AKShare数据适配器
type AKShareAdapter struct {
	baseURL    string
	httpClient *http.Client
}

// NewAKShareAdapter 创建新的AKShare数据适配器
func NewAKShareAdapter(baseURL string) *AKShareAdapter {
	return &AKShareAdapter{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // 增加超时时间到120秒
		},
	}
}

// FetchRealtime 获取实时行情数据
func (a *AKShareAdapter) FetchRealtime(codes []string) ([]model.StockQuote, error) {
	result := make([]model.StockQuote, 0, len(codes))
	
	// 缓存A股和港股数据，避免重复请求
	var aStockList []map[string]interface{}
	var hkStockList []map[string]interface{}
	
	// 添加请求间隔，避免频率过高
	requestInterval := 2 * time.Second
	lastRequestTime := time.Now().Add(-requestInterval)
	
	for _, code := range codes {
		// 根据股票代码格式判断是A股还是港股
		var apiPath string
		var stockCode string
		var stockList *[]map[string]interface{}
		
		if strings.HasSuffix(code, ".SH") || strings.HasSuffix(code, ".SZ") {
			// A股
			apiPath = "/api/public/stock_zh_a_spot_em"
			stockCode = strings.Split(code, ".")[0]
			stockList = &aStockList
		} else if strings.HasSuffix(code, ".HK") {
			// 港股
			apiPath = "/api/public/stock_hk_spot_em"
			stockCode = strings.Split(code, ".")[0]
			stockList = &hkStockList
		} else {
			return nil, fmt.Errorf("不支持的股票代码格式: %s", code)
		}
		
		// 如果还没有获取过该类型的股票数据，则发起请求
		if *stockList == nil {
			// 添加请求间隔，避免频率过高
			elapsed := time.Since(lastRequestTime)
			if elapsed < requestInterval {
				sleepTime := requestInterval - elapsed
				fmt.Printf("等待 %.1f 秒后发起请求...\n", sleepTime.Seconds())
				time.Sleep(sleepTime)
			}
			
			// 构建请求URL
			apiURL := fmt.Sprintf("%s%s", a.baseURL, apiPath)
			
			// 添加重试机制
			var resp *http.Response
			var err error
			for retries := 0; retries < 3; retries++ {
				stockType := "A股"
				if strings.HasSuffix(code, ".HK") {
					stockType = "港股"
				}
				fmt.Printf("正在获取%s数据，尝试 %d/3...\n", stockType, retries+1)
				
				// 记录请求时间
				lastRequestTime = time.Now()
				resp, err = a.httpClient.Get(apiURL)
				if err == nil {
					break
				}
				
				fmt.Printf("API请求失败: %v, 重试中...\n", err)
				// 增加退避时间，避免频率过高
				time.Sleep(time.Duration(3*(retries+1)) * time.Second)
			}
			
			if err != nil {
				// 不立即返回错误，而是记录错误并继续处理其他股票
				stockType := "A股"
				if strings.HasSuffix(code, ".HK") {
					stockType = "港股"
				}
				fmt.Printf("获取%s数据失败: %v，跳过此类型股票\n", stockType, err)
				continue
			}
			
			// 读取响应内容
			body, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				fmt.Printf("读取响应失败: %v，跳过此类型股票\n", err)
				continue
			}
			
			// 检查HTTP状态码
			if resp.StatusCode != http.StatusOK {
				fmt.Printf("API返回错误状态码: %d，跳过此类型股票\n", resp.StatusCode)
				continue
			}
			
			// 解析JSON响应
			var tempList []map[string]interface{}
			if err := json.Unmarshal(body, &tempList); err != nil {
				fmt.Printf("解析JSON失败: %v，跳过此类型股票\n", err)
				continue
			}
			
			*stockList = tempList
			stockType := "A股"
			if strings.HasSuffix(code, ".HK") {
				stockType = "港股"
			}
			fmt.Printf("成功获取%s数据，共%d条记录\n", stockType, len(tempList))
		}
		
		// 查找对应的股票
		var found bool
		for _, stock := range *stockList {
			stockCodeStr := fmt.Sprintf("%v", stock["代码"])
			// 移除可能的前导零
			stockCodeStr = strings.TrimLeft(stockCodeStr, "0")
			stockCode = strings.TrimLeft(stockCode, "0")
			
			if stockCodeStr == stockCode {
				// 转换为模型
				quote := model.StockQuote{
					Symbol:        code,
					Name:          fmt.Sprintf("%v", stock["名称"]),
					Price:         parseFloat(stock["最新价"]),
					Open:          parseFloat(stock["开盘价"]),
					High:          parseFloat(stock["最高价"]),
					Low:           parseFloat(stock["最低价"]),
					Volume:        parseFloat(stock["成交量"]),
					ChangePercent: parseFloat(stock["涨跌幅"]),
					Timestamp:     time.Now(),
				}
				result = append(result, quote)
				found = true
				break
			}
		}
		
		if !found {
			// 如果没找到，尝试打印一些调试信息
			fmt.Printf("未找到股票: %s (代码: %s)，数据集大小: %d\n", 
				code, stockCode, len(*stockList))
			
			// 打印前5个股票代码作为参考
			if len(*stockList) > 0 {
				fmt.Println("数据集中的前5个股票代码:")
				for i := 0; i < min(5, len(*stockList)); i++ {
					fmt.Printf("  %v\n", (*stockList)[i]["代码"])
				}
			}
			
			// 不返回错误，继续处理其他股票
			continue
		}
	}
	
	if len(result) == 0 {
		return nil, fmt.Errorf("未找到任何请求的股票")
	}
	
	return result, nil
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseFloat 将接口类型转换为float64
func parseFloat(v interface{}) float64 {
	switch value := v.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case string:
		var f float64
		fmt.Sscanf(value, "%f", &f) // 修复：直接将结果存入float64变量
		return f
	default:
		return 0
	}
}