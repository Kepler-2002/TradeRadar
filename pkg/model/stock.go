package model

import (
	"time"
)

// StockQuote 股票行情数据
type StockQuote struct {
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	Price         float64   `json:"price"`
	Open          float64   `json:"open"`
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	Volume        float64   `json:"volume"`
	Timestamp     time.Time `json:"timestamp"`
	ChangePercent float64   `json:"change_percent"`
}

// AlertType 异动类型
type AlertType int

const (
	AlertPriceVolatility AlertType = iota
	AlertVolumeSpike
	AlertNewsImpact
)

// AlertEvent 异动事件
type AlertEvent struct {
	Symbol    string    `json:"symbol"`
	Type      AlertType `json:"type"`
	Intensity float64   `json:"intensity"`
	Timestamp time.Time `json:"timestamp"`
}

// DetectionRule 异动检测规则
type DetectionRule struct {
	Type      AlertType `json:"type"`
	Threshold float64   `json:"threshold"`
}