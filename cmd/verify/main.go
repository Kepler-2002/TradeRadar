package main

import (
	"fmt"
	"log"
	"time"

	"TradeRadar/pkg/model"
)

func main() {
	log.Println("开始简单验证...")

	// 创建一个模拟的股票行情
	quote := model.StockQuote{
		Symbol:        "000001.SZ",
		Name:          "平安银行",
		Price:         15.8,
		Open:          15.0,
		High:          16.0,
		Low:           14.9,
		Volume:        2000000,
		ChangePercent: 6.5,
		Timestamp:     time.Now(),
	}

	// 打印行情信息
	fmt.Printf("股票行情: %+v\n", quote)

	// 创建一个模拟的异动事件
	alert := model.AlertEvent{
		Symbol:    "000001.SZ",
		Type:      model.AlertPriceVolatility,
		Intensity: 1.3,
		Timestamp: time.Now(),
	}

	// 打印异动信息
	fmt.Printf("异动事件: %+v\n", alert)

	log.Println("验证完成")
}