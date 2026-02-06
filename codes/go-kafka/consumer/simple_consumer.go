package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go-kafka/config"
	"go-kafka/utils"
)

// MessageHandler 消息处理函数类型
type MessageHandler func(msg kafka.Message) error

// SimpleConsumer 简单消费者（单分区）
type SimpleConsumer struct {
	reader    *kafka.Reader
	config    *config.KafkaConfig
	logger    *utils.Logger
	partition int
}

// NewSimpleConsumer 创建简单消费者
// partition: 指定分区，-1表示不指定（使用消费者组）
func NewSimpleConsumer(cfg *config.KafkaConfig, partition int) *SimpleConsumer {
	return &SimpleConsumer{
		config:    cfg,
		partition: partition,
		logger:    utils.NewLogger("[SimpleConsumer]"),
	}
}

// Connect 连接到Kafka
func (c *SimpleConsumer) Connect() error {
	config := kafka.ReaderConfig{
		Brokers: c.config.Brokers,
		Topic:   c.config.Topic,
		GroupID: c.config.GroupID,

		// 分区配置
		Partition:        c.partition,
		MinBytes:         1,    // 最小抓取字节
		MaxBytes:         10e6, // 10MB 最大抓取字节
		MaxWait:          1 * time.Second,
		ReadBatchTimeout: 5 * time.Second,
		ReadLagInterval:  -1,

		// 提交配置
		CommitInterval:         0, // 手动提交设为0，自动提交设时间间隔
		WatchPartitionChanges:  true,
		PartitionWatchInterval: 5 * time.Second,

		// 错误处理
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			c.logger.Error(fmt.Sprintf(msg, args...))
		}),
	}

	c.reader = kafka.NewReader(config)
	c.logger.Info("消费者连接成功, topic:", c.config.Topic)
	return nil
}

// Start 开始消费（阻塞）
func (c *SimpleConsumer) Start(ctx context.Context, handler MessageHandler) error {
	c.logger.Info("开始消费消息...")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("收到停止信号，退出消费")
			return nil
		default:
		}

		// 读取消息
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // 上下文取消
			}
			c.logger.Error("读取消息失败:", err)
			continue
		}

		// 处理消息
		if err := c.processMessage(msg, handler); err != nil {
			c.logger.Error("处理消息失败:", err)
			// 可以选择重试或跳过
		}
	}
}

// processMessage 处理单条消息
func (c *SimpleConsumer) processMessage(msg kafka.Message, handler MessageHandler) error {
	c.logger.Info("收到消息, partition:", msg.Partition,
		"offset:", msg.Offset,
		"key:", string(msg.Key))

	// 调用业务处理函数
	if handler != nil {
		if err := handler(msg); err != nil {
			return err
		}
	}

	return nil
}

// StartWithGracefulShutdown 带优雅关闭的消费
func (c *SimpleConsumer) StartWithGracefulShutdown(ctx context.Context, handler MessageHandler) error {
	// 创建子上下文
	consumerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 处理关闭信号
	go func() {
		<-ctx.Done()
		c.logger.Info("正在优雅关闭消费者...")
		cancel()
	}()

	return c.Start(consumerCtx, handler)
}

// ReadMessage 读取单条消息（非阻塞式，适合低频消费）
func (c *SimpleConsumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	return c.reader.ReadMessage(ctx)
}

// Stats 获取消费者统计信息
func (c *SimpleConsumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}

// SetOffset 设置消费偏移量
func (c *SimpleConsumer) SetOffset(offset int64) error {
	return c.reader.SetOffset(offset)
}

// Close 关闭消费者
func (c *SimpleConsumer) Close() error {
	if c.reader != nil {
		if err := c.reader.Close(); err != nil {
			return fmt.Errorf("关闭消费者失败: %w", err)
		}
		c.logger.Info("消费者已关闭")
	}
	return nil
}

// FetchLag 获取消费延迟信息
func (c *SimpleConsumer) FetchLag(ctx context.Context) (kafka.Lag, error) {
	return c.reader.ReadLag(ctx)
}
