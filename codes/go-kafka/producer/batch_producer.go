package producer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go-kafka/config"
	"go-kafka/utils"
)

// BatchProducer 高性能批量生产者，适用于大数据量场景
type BatchProducer struct {
	writer      *kafka.Writer
	config      *config.KafkaConfig
	logger      *utils.Logger
	buffer      []kafka.Message
	bufferMutex sync.Mutex
	batchSize   int
	flushTicker *time.Ticker
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	compressor  kafka.Compression
}

// BatchProducerOption 批量生产者配置选项
type BatchProducerOption func(*BatchProducer)

// WithBatchSize 设置批量大小
func WithBatchSize(size int) BatchProducerOption {
	return func(p *BatchProducer) {
		p.batchSize = size
	}
}

// WithCompression 设置压缩算法
func WithCompression(algo kafka.Compression) BatchProducerOption {
	return func(p *BatchProducer) {
		p.compressor = algo
	}
}

// NewBatchProducer 创建批量生产者
func NewBatchProducer(cfg *config.KafkaConfig, options ...BatchProducerOption) *BatchProducer {
	ctx, cancel := context.WithCancel(context.Background())

	p := &BatchProducer{
		config:     cfg,
		logger:     utils.NewLogger("[BatchProducer]"),
		buffer:     make([]kafka.Message, 0, 1000),
		batchSize:  500,
		compressor: kafka.Lz4, // 默认使用lz4压缩
		ctx:        ctx,
		cancel:     cancel,
	}

	// 应用选项
	for _, opt := range options {
		opt(p)
	}

	return p
}

// Connect 连接到Kafka
func (p *BatchProducer) Connect() error {
	p.writer = &kafka.Writer{
		Addr:         kafka.TCP(p.config.Brokers...),
		Topic:        p.config.Topic,
		Balancer:     &kafka.CRC32Balancer{}, // CRC32分区器，与Java客户端兼容
		RequiredAcks: kafka.RequireOne,
		Async:        false, // 同步模式，我们自己控制批量

		// 压缩配置
		Compression: p.compressor,

		// 批处理配置
		BatchSize:    p.batchSize,
		BatchBytes:   10 * 1048576, // 10MB
		BatchTimeout: 5 * time.Second,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
		MaxAttempts:  5,

		// 重试退避策略
		WriteBackoffMin: 100 * time.Millisecond,
		WriteBackoffMax: 1 * time.Second,
	}

	// 启动定时刷新器
	p.flushTicker = time.NewTicker(1 * time.Second)
	p.wg.Add(1)
	go p.autoFlush()

	p.logger.Info("批量生产者连接成功，压缩算法:", p.compressor)
	return nil
}

// Send 添加消息到缓冲区（立即返回）
func (p *BatchProducer) Send(key, value string) error {
	p.bufferMutex.Lock()
	defer p.bufferMutex.Unlock()

	msg := kafka.Message{
		Key:   []byte(key),
		Value: []byte(value),
		Time:  time.Now(),
	}

	p.buffer = append(p.buffer, msg)

	// 达到批量阈值，立即刷新
	if len(p.buffer) >= p.batchSize {
		go p.Flush()
	}

	return nil
}

// SendStructured 发送结构化数据（自动序列化）
func (p *BatchProducer) SendStructured(key string, data interface{}) error {
	// 这里可以根据需要实现JSON/Protobuf等序列化
	// 简单示例，实际项目中使用 json.Marshal 等
	value := fmt.Sprintf("%v", data)
	return p.Send(key, value)
}

// Flush 手动刷新缓冲区
func (p *BatchProducer) Flush() error {
	p.bufferMutex.Lock()

	if len(p.buffer) == 0 {
		p.bufferMutex.Unlock()
		return nil
	}

	// 复制消息并清空缓冲区
	messages := make([]kafka.Message, len(p.buffer))
	copy(messages, p.buffer)
	p.buffer = p.buffer[:0]
	p.bufferMutex.Unlock()

	// 异步发送
	return p.sendBatch(messages)
}

// sendBatch 批量发送消息
func (p *BatchProducer) sendBatch(messages []kafka.Message) error {
	if len(messages) == 0 {
		return nil
	}

	start := time.Now()
	err := p.writer.WriteMessages(p.ctx, messages...)
	duration := time.Since(start)

	if err != nil {
		// 检查是否是部分失败
		if writeErrors, ok := err.(kafka.WriteErrors); ok {
			successCount := len(messages) - writeErrors.Count()
			p.logger.Error("批量发送部分失败，成功:", successCount, "失败:", writeErrors.Count())

			// 处理失败的消息（可以加入重试队列）
			for i := range messages {
				if writeErrors[i] != nil {
					p.handleFailedMessage(messages[i], writeErrors[i])
				}
			}
			return writeErrors
		}

		p.logger.Error("批量发送失败:", err)
		return err
	}

	p.logger.Info("批量发送成功，数量:", len(messages), "耗时:", duration)
	return nil
}

// handleFailedMessage 处理发送失败的消息
func (p *BatchProducer) handleFailedMessage(msg kafka.Message, err error) {
	// 实际项目中可以加入死信队列或重试队列
	p.logger.Error("消息发送失败, key:", string(msg.Key), "error:", err)
}

// autoFlush 定时自动刷新
func (p *BatchProducer) autoFlush() {
	defer p.wg.Done()

	for {
		select {
		case <-p.flushTicker.C:
			if err := p.Flush(); err != nil {
				p.logger.Error("自动刷新失败:", err)
			}
		case <-p.ctx.Done():
			p.Flush() // 退出前最后刷新一次
			return
		}
	}
}

// Close 关闭批量生产者
func (p *BatchProducer) Close() error {
	p.cancel()
	p.flushTicker.Stop()
	p.wg.Wait()

	// 最后刷新剩余消息
	p.Flush()

	if p.writer != nil {
		if err := p.writer.Close(); err != nil {
			return fmt.Errorf("关闭生产者失败: %w", err)
		}
	}

	p.logger.Info("批量生产者已关闭")
	return nil
}

// BufferSize 获取当前缓冲区大小
func (p *BatchProducer) BufferSize() int {
	p.bufferMutex.Lock()
	defer p.bufferMutex.Unlock()
	return len(p.buffer)
}

// Stats 获取统计信息
func (p *BatchProducer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}
