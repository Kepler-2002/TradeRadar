package collector

import (
	"TradeRadar/pkg/model"
)

// QuoteFetcher 行情数据获取接口
type QuoteFetcher interface {
	FetchRealtime(codes []string) ([]model.StockQuote, error)
}