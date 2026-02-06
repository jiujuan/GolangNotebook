package consumer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go-kafka/config"
	"go-kafka/utils"
)

// ManualCommitConsumer 手动提交偏移量的消费者，确保消息不丢失
type ManualCommitConsumer struct {
	reader         *kafka.Reader
	config         *config.KafkaConfig
	logger         *utils.Logger
	uncommitted    []kafka.Message
	commitMutex    sync.Mutex
	maxUncommitted int
}

// NewManualCommitConsumer 创建手动提交消费者
// maxUncommitted: 最大未提交消息数，达到该数量会触发提交
func NewManualCommitConsumer(cfg *config.KafkaConfig, maxUncommitted int) *ManualCommitConsumer {
	if maxUncommitted <= 0 {
		maxUncommitted = 100
	}
	return &ManualCommitConsumer{
		config:         cfg,
		logger:         utils.NewLogger("[ManualCommitConsumer]"),
		maxUncommitted: maxUncommitted,
		uncommitted:    make([]kafka.Message, 0, maxUncommitted),
	}
}

// Connect 连接到Kafka
func (c *ManualCommitConsumer) Connect() error {
	config := kafka.ReaderConfig{
		Brokers: c.config.Brokers,
		Topic:   c.config.Topic,
		GroupID: c.config.GroupID,

		// 关键：禁用自动提交
		CommitInterval: 0, // 设为0表示不自动提交

		// 消费配置
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        500 * time.Millisecond,
		ReadBackoffMin: 100 * time.Millisecond,
		ReadBackoffMax: 500 * time.Millisecond,

		// 错误处理
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			c.logger.Error(fmt.Sprintf(msg, args...))
		}),
	}

	c.reader = kafka.NewReader(config)
	c.logger.Info("手动提交消费者连接成功")
	return nil
}

// Start 开始消费并手动提交
func (c *ManualCommitConsumer) Start(ctx context.Context, handler MessageHandler) error {
	c.logger.Info("开始消费（手动提交模式）...")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("收到停止信号")
			// 退出前尝试提交剩余消息
			c.commitUncommitted(ctx)
			return nil
		default:
		}

		// 读取消息
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.logger.Error("读取消息失败:", err)
			continue
		}

		// 处理消息
		processErr := c.processAndCommit(msg, handler)
		if processErr != nil {
			c.logger.Error("处理消息失败:", processErr)
			// 可以选择重试或记录死信队列
		}
	}
}

// processAndCommit 处理消息并管理提交
func (c *ManualCommitConsumer) processAndCommit(msg kafka.Message, handler MessageHandler) error {
	c.logger.Info("处理消息, partition:", msg.Partition, "offset:", msg.Offset)

	// 执行业务逻辑
	if handler != nil {
		if err := handler(msg); err != nil {
			// 业务处理失败，不提交偏移量，消息会被重新消费
			return fmt.Errorf("业务处理失败: %w", err)
		}
	}

	// 业务处理成功，加入未提交队列
	c.commitMutex.Lock()
	c.uncommitted = append(c.uncommitted, msg)
	shouldCommit := len(c.uncommitted) >= c.maxUncommitted
	c.commitMutex.Unlock()

	// 达到阈值，执行提交
	if shouldCommit {
		if err := c.Commit(ctx.Background()); err != nil {
			c.logger.Error("批量提交失败:", err)
			return err
		}
	}

	return nil
}

// Commit 手动提交所有未提交的消息
func (c *ManualCommitConsumer) Commit(ctx context.Context) error {
	c.commitMutex.Lock()
	defer c.commitMutex.Unlock()

	if len(c.uncommitted) == 0 {
		return nil
	}

	// 复制消息
	msgs := make([]kafka.Message, len(c.uncommitted))
	copy(msgs, c.uncommitted)

	// 提交偏移量
	start := time.Now()
	if err := c.reader.CommitMessages(ctx, msgs...); err != nil {
		return fmt.Errorf("提交偏移量失败: %w", err)
	}

	c.logger.Info("提交成功，数量:", len(msgs), "耗时:", time.Since(start))

	// 清空已提交消息
	c.uncommitted = c.uncommitted[:0]
	return nil
}

// commitUncommitted 尝试提交剩余消息（用于关闭时）
func (c *ManualCommitConsumer) commitUncommitted(ctx context.Context) {
	c.commitMutex.Lock()
	count := len(c.uncommitted)
	c.commitMutex.Unlock()

	if count > 0 {
		c.logger.Info("关闭前提交剩余消息:", count)
		if err := c.Commit(ctx); err != nil {
			c.logger.Error("关闭时提交失败:", err)
		}
	}
}

// Close 关闭消费者
func (c *ManualCommitConsumer) Close() error {
	// 最后尝试提交
	c.commitUncommitted(context.Background())

	if c.reader != nil {
		if err := c.reader.Close(); err != nil {
			return fmt.Errorf("关闭消费者失败: %w", err)
		}
	}

	c.logger.Info("手动提交消费者已关闭")
	return nil
}

// GetLag 获取当前消费延迟
func (c *ManualCommitConsumer) GetLag(ctx context.Context) (kafka.Lag, error) {
	return c.reader.ReadLag(ctx)
}

// GetUncommittedCount 获取未提交消息数量
func (c *ManualCommitConsumer) GetUncommittedCount() int {
	c.commitMutex.Lock()
	defer c.commitMutex.Unlock()
	return len(c.uncommitted)
}
