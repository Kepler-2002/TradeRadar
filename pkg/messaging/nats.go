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
	
	return c.conn.Publish("quotes", data)
}

// PublishAlert 发布异动事件
func (c *NATSClient) PublishAlert(alert model.AlertEvent) error {
	data, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("序列化异动事件失败: %w", err)
	}
	
	return c.conn.Publish("alerts", data)
}

// SubscribeQuotes 订阅行情数据
func (c *NATSClient) SubscribeQuotes(handler func(model.StockQuote)) (stan.Subscription, error) {
	return c.conn.Subscribe(
		"quotes",
		func(msg *stan.Msg) {
			var quote model.StockQuote
			if err := json.Unmarshal(msg.Data, &quote); err != nil {
				log.Printf("解析行情数据失败: %v\n", err)
				return
			}
			handler(quote)
		},
		stan.StartWithLastReceived(),
	)
}

// SubscribeAlerts 订阅异动事件
func (c *NATSClient) SubscribeAlerts(handler func(model.AlertEvent)) (stan.Subscription, error) {
	return c.conn.Subscribe(
		"alerts",
		func(msg *stan.Msg) {
			var alert model.AlertEvent
			if err := json.Unmarshal(msg.Data, &alert); err != nil {
				log.Printf("解析异动事件失败: %v\n", err)
				return
			}
			handler(alert)
		},
		stan.StartWithLastReceived(),
	)
}