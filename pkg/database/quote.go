// pkg/database/quote.go
package database

import (
	"TradeRadar/pkg/model"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type QuoteDB struct {
	db *gorm.DB
}

func (t *TimescaleDB) Quote() *QuoteDB {
	return &QuoteDB{db: t.db}
}

func (q *QuoteDB) Save(quote *model.StockQuote) error {
	return q.db.Create(quote).Error
}

func (q *QuoteDB) SaveBatch(quotes []*model.StockQuote) error {
	return q.db.CreateInBatches(quotes, 1000).Error
}

func (q *QuoteDB) GetLatest(symbol string) (*model.StockQuote, error) {
	var quote model.StockQuote
	err := q.db.Where("symbol = ?", symbol).
		Order("timestamp DESC").
		First(&quote).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("行情数据不存在")
		}
		return nil, fmt.Errorf("获取行情数据失败: %w", err)
	}
	return &quote, nil
}

func (q *QuoteDB) GetHistorical(symbol string, startTime, endTime time.Time, limit int) ([]*model.StockQuote, error) {
	var quotes []*model.StockQuote
	err := q.db.Where("symbol = ? AND timestamp BETWEEN ? AND ?", symbol, startTime, endTime).
		Order("timestamp DESC").
		Limit(limit).
		Find(&quotes).Error

	if err != nil {
		return nil, fmt.Errorf("查询历史行情失败: %w", err)
	}
	return quotes, nil
}

func (q *QuoteDB) GetByTimeRange(symbols []string, startTime, endTime time.Time) ([]*model.StockQuote, error) {
	var quotes []*model.StockQuote
	query := q.db.Where("timestamp BETWEEN ? AND ?", startTime, endTime)

	if len(symbols) > 0 {
		query = query.Where("symbol IN ?", symbols)
	}

	err := query.Order("symbol ASC, timestamp DESC").Find(&quotes).Error
	if err != nil {
		return nil, fmt.Errorf("查询时间范围行情失败: %w", err)
	}
	return quotes, nil
}

func (q *QuoteDB) GetKLineData(symbol string, interval string, limit int) ([]*model.StockQuote, error) {
	// 根据间隔获取K线数据，这里简化处理
	var quotes []*model.StockQuote
	err := q.db.Where("symbol = ?", symbol).
		Order("timestamp DESC").
		Limit(limit).
		Find(&quotes).Error

	if err != nil {
		return nil, fmt.Errorf("查询K线数据失败: %w", err)
	}
	return quotes, nil
}

func (q *QuoteDB) GetLatestBySymbols(symbols []string) ([]*model.StockQuote, error) {
	var quotes []*model.StockQuote

	// 使用子查询获取每个股票的最新行情
	subQuery := q.db.Model(&model.StockQuote{}).
		Select("symbol, MAX(timestamp) as max_timestamp").
		Where("symbol IN ?", symbols).
		Group("symbol")

	err := q.db.Table("stock_quotes sq1").
		Joins("JOIN (?) sq2 ON sq1.symbol = sq2.symbol AND sq1.timestamp = sq2.max_timestamp", subQuery).
		Find(&quotes).Error

	if err != nil {
		return nil, fmt.Errorf("查询多股票最新行情失败: %w", err)
	}
	return quotes, nil
}

func (q *QuoteDB) GetPriceChanges(symbols []string, hours int) ([]*model.StockQuote, error) {
	var quotes []*model.StockQuote
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	err := q.db.Where("symbol IN ? AND timestamp >= ?", symbols, startTime).
		Order("symbol ASC, timestamp DESC").
		Find(&quotes).Error

	if err != nil {
		return nil, fmt.Errorf("查询价格变化失败: %w", err)
	}
	return quotes, nil
}

func (q *QuoteDB) DeleteOldData(days int) error {
	cutoffTime := time.Now().AddDate(0, 0, -days)
	return q.db.Where("timestamp < ?", cutoffTime).Delete(&model.StockQuote{}).Error
}
