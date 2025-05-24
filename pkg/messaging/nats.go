// pkg/messaging/nats.go
package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// NATSClient NATS JetStream客户端 - 纯基础能力封装
type NATSClient struct {
	conn      *nats.Conn
	jetStream jetstream.JetStream
	natsURL   string
	ctx       context.Context
	cancel    context.CancelFunc
	consumers map[string]jetstream.Consumer // 消费者管理
	mu        sync.RWMutex                  // 保护consumers
}

// MessageHandler 通用消息处理函数类型
type MessageHandler func(data []byte) error

// NewNATSClient 创建新的NATS客户端
func NewNATSClient(natsURL string) (*NATSClient, error) {
	// 连接NATS
	nc, err := nats.Connect(natsURL,
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1), // 无限重连
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS连接断开: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Println("NATS重新连接成功")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("连接NATS失败: %w", err)
	}

	// 创建JetStream上下文
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("创建JetStream失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &NATSClient{
		conn:      nc,
		jetStream: js,
		natsURL:   natsURL,
		ctx:       ctx,
		cancel:    cancel,
		consumers: make(map[string]jetstream.Consumer),
	}

	// 初始化基础Streams
	if err := client.setupStreams(); err != nil {
		log.Printf("警告: 设置Streams失败: %v", err)
	}

	return client, nil
}

// setupStreams 设置基础的Streams
func (c *NATSClient) setupStreams() error {
	// 定义基础的Streams配置
	streams := []jetstream.StreamConfig{
		{
			Name:        "NEWS_STREAM",
			Subjects:    []string{"news.*"},
			Description: "新闻数据流",
			Retention:   jetstream.LimitsPolicy,
			MaxMsgs:     10000,
			MaxBytes:    100 * 1024 * 1024,  // 100MB
			MaxAge:      7 * 24 * time.Hour, // 保留7天
		},
		{
			Name:        "QUOTES_STREAM",
			Subjects:    []string{"quotes.*"},
			Description: "股票行情数据流",
			Retention:   jetstream.LimitsPolicy,
			MaxMsgs:     100000,
			MaxBytes:    100 * 1024 * 1024, // 100MB
			MaxAge:      24 * time.Hour,    // 保留24小时
		},
		{
			Name:        "ALERTS_STREAM",
			Subjects:    []string{"alerts.*"},
			Description: "异动事件数据流",
			Retention:   jetstream.LimitsPolicy,
			MaxMsgs:     50000,
			MaxBytes:    50 * 1024 * 1024,   // 50MB
			MaxAge:      7 * 24 * time.Hour, // 保留7天
		},
	}

	for _, streamConfig := range streams {
		_, err := c.jetStream.CreateOrUpdateStream(c.ctx, streamConfig)
		if err != nil {
			log.Printf("创建/更新Stream %s 失败: %v", streamConfig.Name, err)
		} else {
			log.Printf("Stream %s 设置成功", streamConfig.Name)
		}
	}

	return nil
}

// Publish 发布消息到指定主题
func (c *NATSClient) Publish(subject string, data interface{}) error {
	var payload []byte
	var err error

	switch v := data.(type) {
	case []byte:
		payload = v
	case string:
		payload = []byte(v)
	default:
		payload, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("序列化数据失败: %w", err)
		}
	}

	_, err = c.jetStream.Publish(c.ctx, subject, payload)
	if err != nil {
		return fmt.Errorf("发布消息到 %s 失败: %w", subject, err)
	}

	log.Printf("发布消息到主题: %s, 数据大小: %d bytes", subject, len(payload))
	return nil
}

// Subscribe 订阅指定主题的消息
func (c *NATSClient) Subscribe(streamName, consumerName, filterSubject string, handler MessageHandler) error {
	// 创建消费者配置
	consumerConfig := jetstream.ConsumerConfig{
		Name:          consumerName,
		Description:   fmt.Sprintf("%s 消费者", consumerName),
		FilterSubject: filterSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverNewPolicy,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
	}

	// 创建或获取消费者
	consumer, err := c.jetStream.CreateOrUpdateConsumer(c.ctx, streamName, consumerConfig)
	if err != nil {
		return fmt.Errorf("创建消费者 %s 失败: %w", consumerName, err)
	}

	// 保存消费者引用
	c.mu.Lock()
	c.consumers[consumerName] = consumer
	c.mu.Unlock()

	// 开始消费消息
	go c.consumeMessages(consumer, consumerName, handler)

	log.Printf("已订阅 %s (Stream: %s, Consumer: %s)", filterSubject, streamName, consumerName)
	return nil
}

// consumeMessages 消费消息的通用逻辑
func (c *NATSClient) consumeMessages(consumer jetstream.Consumer, consumerName string, handler MessageHandler) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("消费者 %s 异常退出: %v", consumerName, r)
		}
	}()

	iter, err := consumer.Messages(jetstream.PullMaxMessages(10))
	if err != nil {
		log.Printf("获取 %s 消息迭代器失败: %v", consumerName, err)
		return
	}

	for {
		select {
		case <-c.ctx.Done():
			log.Printf("消费者 %s 收到停止信号", consumerName)
			return
		default:
			msg, err := iter.Next()
			if err != nil {
				if err == jetstream.ErrNoMessages {
					continue
				}
				log.Printf("获取 %s 消息失败: %v", consumerName, err)
				time.Sleep(1 * time.Second)
				continue
			}

			// 调用处理器
			if err := handler(msg.Data()); err != nil {
				log.Printf("消费者 %s 处理消息失败: %v", consumerName, err)
				msg.Nak() // 拒绝消息
			} else {
				msg.Ack() // 确认消息
			}
		}
	}
}

// CreateStream 创建新的Stream
func (c *NATSClient) CreateStream(config jetstream.StreamConfig) error {
	_, err := c.jetStream.CreateOrUpdateStream(c.ctx, config)
	if err != nil {
		return fmt.Errorf("创建Stream %s 失败: %w", config.Name, err)
	}
	log.Printf("Stream %s 创建成功", config.Name)
	return nil
}

// DeleteConsumer 删除消费者
func (c *NATSClient) DeleteConsumer(streamName, consumerName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.jetStream.DeleteConsumer(c.ctx, streamName, consumerName); err != nil {
		return fmt.Errorf("删除消费者 %s 失败: %w", consumerName, err)
	}

	delete(c.consumers, consumerName)
	log.Printf("消费者 %s 已删除", consumerName)
	return nil
}

// GetStreamInfo 获取Stream信息
func (c *NATSClient) GetStreamInfo(streamName string) (*jetstream.StreamInfo, error) {
	return c.jetStream.Stream(c.ctx, streamName)
}

// GetConsumerInfo 获取消费者信息
func (c *NATSClient) GetConsumerInfo(streamName, consumerName string) (*jetstream.ConsumerInfo, error) {
	return c.jetStream.Consumer(c.ctx, streamName, consumerName)
}

// Close 关闭连接
func (c *NATSClient) Close() error {
	log.Println("正在关闭NATS连接...")

	c.cancel() // 取消所有上下文

	// 等待所有消费者退出
	time.Sleep(1 * time.Second)

	// 清理消费者记录
	c.mu.Lock()
	for name := range c.consumers {
		log.Printf("清理消费者: %s", name)
	}
	c.consumers = make(map[string]jetstream.Consumer)
	c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}

	log.Println("NATS连接已关闭")
	return nil
}

// IsConnected 检查连接状态
func (c *NATSClient) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// GetStats 获取连接统计信息
func (c *NATSClient) GetStats() nats.Statistics {
	if c.conn != nil {
		return c.conn.Stats()
	}
	return nats.Statistics{}
}
