// pkg/model/stock.go (更新版本)
package model

import (
	"time"
)

// Stock 股票基础信息
type Stock struct {
	Symbol      string    `gorm:"type:varchar(20);primaryKey" json:"symbol"`     // 股票代码
	Name        string    `gorm:"not null" json:"name"`                          // 股票名称
	Market      string    `gorm:"type:varchar(10);not null;index" json:"market"` // 市场 (SZ/SH)
	Industry    string    `gorm:"index" json:"industry"`                         // 行业
	Sector      string    `gorm:"index" json:"sector"`                           // 板块
	ListingDate time.Time `json:"listing_date"`                                  // 上市日期
	IsActive    bool      `gorm:"default:true;index" json:"is_active"`           // 是否活跃
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联关系
	Quotes     []StockQuote `gorm:"foreignKey:Symbol;references:Symbol" json:"quotes,omitempty"`
	NewsEvents []NewsEvent  `gorm:"foreignKey:Symbol;references:Symbol" json:"news_events,omitempty"`
}

// StockQuote 股票行情数据
type StockQuote struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Symbol        string    `gorm:"type:varchar(20);not null;index" json:"symbol"`
	Name          string    `gorm:"not null" json:"name"`
	Price         float64   `gorm:"type:decimal(10,3)" json:"price"`         // 当前价
	Open          float64   `gorm:"type:decimal(10,3)" json:"open"`          // 开盘价
	High          float64   `gorm:"type:decimal(10,3)" json:"high"`          // 最高价
	Low           float64   `gorm:"type:decimal(10,3)" json:"low"`           // 最低价
	Close         float64   `gorm:"type:decimal(10,3)" json:"close"`         // 收盘价
	PreClose      float64   `gorm:"type:decimal(10,3)" json:"pre_close"`     // 昨收价
	Volume        float64   `gorm:"type:bigint" json:"volume"`               // 成交量
	Amount        float64   `gorm:"type:decimal(20,2)" json:"amount"`        // 成交额
	Turnover      float64   `gorm:"type:decimal(8,4)" json:"turnover"`       // 换手率
	Change        float64   `gorm:"type:decimal(10,3)" json:"change"`        // 涨跌额
	ChangePercent float64   `gorm:"type:decimal(8,4)" json:"change_percent"` // 涨跌幅
	Timestamp     time.Time `gorm:"not null;index:idx_symbol_timestamp" json:"timestamp"`
	CreatedAt     time.Time `json:"created_at"`

	// 关联
	Stock Stock `gorm:"foreignKey:Symbol;references:Symbol" json:"stock,omitempty"`
}

// 创建复合索引
func (StockQuote) TableName() string {
	return "stock_quotes"
}
