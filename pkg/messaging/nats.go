package messaging

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/stan.go"
	
	"TradeRadar/pkg/model"
)

// NATSClient NATS客户端
type NATSClient struct {
	conn      stan.Conn
	clusterID string
	clientID  string
	natsURL   string
}

// NewNATSClient 创建新的NATS客户端
func NewNATSClient(natsURL, clusterID, clientID string) (*NATSClient, error) {
	client := &NATSClient{
		natsURL:   natsURL,
		clusterID: clusterID,
		clientID:  clientID,
	}
	
	// 连接到NATS Streaming
	conn, err := stan.Connect(
		clusterID,
		clientID,
		stan.NatsURL(natsURL),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			log.Printf("连接丢失: %v\n", reason)
			// 实际项目中应该实现重连逻辑
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("连接NATS Streaming失败: %w", err)
	}
	
	client.conn = conn
	return client, nil
}

// Close 关闭连接
func (c *NATSClient) Close() error {
	return c.conn.Close()
}

// PublishQuote 发布行情数据
func (c *NATSClient) PublishQuote(quote model.StockQuote) error {
	data, err := json.Marshal(quote)
	if err != nil {
		return fmt.Errorf("序列化行情数据失败: %w", err)
	}
	
	// 打印详细的发布信息
	log.Printf("发布行情数据: %s (%.2f) 涨跌幅: %.2f%%, 成交量: %.0f\n", 
		quote.Symbol, quote.Price, quote.ChangePercent, quote.Volume)
	
	return c.conn.Publish("quotes", data)
}

// PublishAlert 发布异动事件
func (c *NATSClient) PublishAlert(alert model.AlertEvent) error {
	data, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("序列化异动事件失败: %w", err)
	}
	
	// 打印详细的异动事件信息，使用正确的字段名
	log.Printf("发布异动事件: %s (%s) 类型: %s\n", 
		alert.Symbol, alert.Name, alert.Type)
	
	return c.conn.Publish("alerts", data)
}

// SubscribeQuotes 订阅行情数据
func (c *NATSClient) SubscribeQuotes(handler func(model.StockQuote)) (stan.Subscription, error) {
	log.Printf("订阅行情数据主题: quotes\n")
	return c.conn.Subscribe(
		"quotes",
		func(msg *stan.Msg) {
			var quote model.StockQuote
			if err := json.Unmarshal(msg.Data, &quote); err != nil {
				log.Printf("解析行情数据失败: %v\n", err)
				return
			}
			
			// 打印接收到的行情数据
			log.Printf("接收行情数据: %s (%.2f) 涨跌幅: %.2f%%, 成交量: %.0f\n", 
				quote.Symbol, quote.Price, quote.ChangePercent, quote.Volume)
			
			handler(quote)
		},
		stan.StartWithLastReceived(),
	)
}

// SubscribeAlerts 订阅异动事件
func (c *NATSClient) SubscribeAlerts(handler func(model.AlertEvent)) (stan.Subscription, error) {
	log.Printf("订阅异动事件主题: alerts\n")
	return c.conn.Subscribe(
		"alerts",
		func(msg *stan.Msg) {
			var alert model.AlertEvent
			if err := json.Unmarshal(msg.Data, &alert); err != nil {
				log.Printf("解析异动事件失败: %v\n", err)
				return
			}
			
			// 打印接收到的异动事件，使用正确的字段名
			log.Printf("接收异动事件: %s (%s) 类型: %s\n", 
				alert.Symbol, alert.Name, alert.Type)
			
			handler(alert)
		},
		stan.StartWithLastReceived(),
	)
}

// PrintQuoteDetails 打印行情数据详情（用于调试）
func PrintQuoteDetails(quote model.StockQuote) {
	fmt.Printf("\n===== 行情数据详情 =====\n")
	fmt.Printf("股票代码: %s\n", quote.Symbol)
	fmt.Printf("股票名称: %s\n", quote.Name)
	fmt.Printf("最新价格: %.2f\n", quote.Price)
	fmt.Printf("开盘价格: %.2f\n", quote.Open)
	fmt.Printf("最高价格: %.2f\n", quote.High)
	fmt.Printf("最低价格: %.2f\n", quote.Low)
	fmt.Printf("成交量: %.0f\n", quote.Volume)
	fmt.Printf("涨跌幅: %.2f%%\n", quote.ChangePercent)
	fmt.Printf("时间戳: %s\n", quote.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("========================\n\n")
}