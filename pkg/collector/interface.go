// pkg/collector/interfaces.go
package collector

import (
	"TradeRadar/pkg/model"
	"time"
)

// NewsCollector 新闻收集器接口 - 与MonitorEngine期望的接口保持一致
type NewsCollector interface {
	StartCollecting() error
	Stop() error
	GetLatestNews(symbol string) ([]*model.NewsEvent, error)
	GetNewsByTimeRange(startTime, endTime time.Time) ([]*model.NewsEvent, error)
	SubscribeNewsUpdates(handler func(*model.NewsEvent)) error
}

// QuoteFetcher 行情获取器接口
type QuoteFetcher interface {
	FetchQuote(symbol string) (*model.StockQuote, error)
	FetchBatchQuotes(symbols []string) ([]*model.StockQuote, error)
	SubscribeQuotes(symbols []string, handler func(*model.StockQuote)) error
}
