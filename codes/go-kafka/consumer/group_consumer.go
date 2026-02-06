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

// GroupConsumer 消费者组实现，支持多实例负载均衡
type GroupConsumer struct {
	config     *config.KafkaConfig
	logger     *utils.Logger
	handler    MessageHandler
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	consumer   *kafka.Reader
	instanceID string
}

// NewGroupConsumer 创建消费者组实例
// instanceID: 当前实例标识，用于日志区分
func NewGroupConsumer(cfg *config.KafkaConfig, instanceID string) *GroupConsumer {
	ctx, cancel := context.WithCancel(context.Background())
	return &GroupConsumer{
		config:     cfg,
		logger:     utils.NewLogger(fmt.Sprintf("[GroupConsumer-%s]", instanceID)),
		ctx:        ctx,
		cancel:     cancel,
		instanceID: instanceID,
	}
}

// Connect 连接到Kafka
func (c *GroupConsumer) Connect() error {
	config := kafka.ReaderConfig{
		Brokers: c.config.Brokers,
		Topic:   c.config.Topic,
		GroupID: c.config.GroupID, // 消费者组ID

		// 消费者组配置
		GroupBalancers: []kafka.GroupBalancer{
			kafka.RangeGroupBalancer{},      // 范围分区策略
			kafka.RoundRobinGroupBalancer{}, // 轮询分区策略
		},

		// 心跳配置
		HeartbeatInterval: 3 * time.Second,
		SessionTimeout:    30 * time.Second,
		RebalanceTimeout:  30 * time.Second,

		// 消费配置
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        1 * time.Second,
		ReadBackoffMin: 100 * time.Millisecond,
		ReadBackoffMax: 1 * time.Second,

		// 起始偏移量配置
		StartOffset: kafka.FirstOffset, // 首次消费从头开始
		// StartOffset: kafka.LastOffset, // 首次消费从最新开始

		// 再平衡回调
		GroupBalancerFunc: func(memberAssignments map[string][]int32) {
			c.logger.Info("分区再平衡完成，分配的分区:", memberAssignments[c.instanceID])
		},

		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			c.logger.Error(fmt.Sprintf(msg, args...))
		}),
	}

	c.consumer = kafka.NewReader(config)
	c.logger.Info("消费者组连接成功, groupID:", c.config.GroupID)
	return nil
}

// Start 开始消费
func (c *GroupConsumer) Start(handler MessageHandler) {
	c.handler = handler
	c.wg.Add(1)

	go func() {
		defer c.wg.Done()
		c.logger.Info("消费者实例启动:", c.instanceID)

		for {
			select {
			case <-c.ctx.Done():
				c.logger.Info("消费者实例停止:", c.instanceID)
				return
			default:
			}

			// 读取消息
			msg, err := c.consumer.ReadMessage(c.ctx)
			if err != nil {
				if c.ctx.Err() != nil {
					return
				}
				c.logger.Error("读取消息失败:", err)
				continue
			}

			// 处理消息
			if err := c.handleMessage(msg); err != nil {
				c.logger.Error("处理消息失败:", err)
			}
		}
	}()
}

// handleMessage 处理消息并提交偏移量
func (c *GroupConsumer) handleMessage(msg kafka.Message) error {
	c.logger.Info("处理消息, partition:", msg.Partition,
		"offset:", msg.Offset,
		"instance:", c.instanceID)

	// 执行业务逻辑
	if c.handler != nil {
		if err := c.handler(msg); err != nil {
			// 业务处理失败，可以选择重试或记录
			return err
		}
	}

	return nil
}

// Stop 停止消费者
func (c *GroupConsumer) Stop() {
	c.logger.Info("正在停止消费者...")
	c.cancel()
	c.wg.Wait()
}

// Close 关闭消费者连接
func (c *GroupConsumer) Close() error {
	c.Stop()

	if c.consumer != nil {
		if err := c.consumer.Close(); err != nil {
			return fmt.Errorf("关闭消费者失败: %w", err)
		}
	}

	c.logger.Info("消费者已关闭")
	return nil
}

// Stats 获取消费统计
func (c *GroupConsumer) Stats() kafka.ReaderStats {
	return c.consumer.Stats()
}

// CommitMessages 手动提交消息偏移量（如开启自动提交则无需调用）
func (c *GroupConsumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	return c.consumer.CommitMessages(ctx, msgs...)
}

// ConsumerGroupManager 消费者组管理器，管理多个消费实例
type ConsumerGroupManager struct {
	consumers []*GroupConsumer
	config    *config.KafkaConfig
	logger    *utils.Logger
}

// NewConsumerGroupManager 创建消费者组管理器
func NewConsumerGroupManager(cfg *config.KafkaConfig) *ConsumerGroupManager {
	return &ConsumerGroupManager{
		config: cfg,
		logger: utils.NewLogger("[ConsumerGroupManager]"),
	}
}

// StartConsumers 启动多个消费者实例
func (m *ConsumerGroupManager) StartConsumers(count int, handler MessageHandler) error {
	m.consumers = make([]*GroupConsumer, count)

	for i := 0; i < count; i++ {
		instanceID := fmt.Sprintf("instance-%d", i)
		consumer := NewGroupConsumer(m.config, instanceID)

		if err := consumer.Connect(); err != nil {
			return fmt.Errorf("连接消费者%d失败: %w", i, err)
		}

		consumer.Start(handler)
		m.consumers[i] = consumer
		m.logger.Info("启动消费者实例:", instanceID)
	}

	m.logger.Info("共启动消费者实例数:", count)
	return nil
}

// StopAll 停止所有消费者
func (m *ConsumerGroupManager) StopAll() {
	m.logger.Info("正在停止所有消费者...")
	for _, c := range m.consumers {
		if c != nil {
			c.Close()
		}
	}
}
