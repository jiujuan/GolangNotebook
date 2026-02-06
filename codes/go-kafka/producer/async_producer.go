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

// AsyncProducer 异步生产者，支持回调
type AsyncProducer struct {
	writer    *kafka.Writer
	config    *config.KafkaConfig
	logger    *utils.Logger
	callback  func(msg kafka.Message, err error)
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	msgChan   chan kafka.Message
	batchSize int
}

// NewAsyncProducer 创建异步生产者
// callback: 消息发送后的回调函数
func NewAsyncProducer(cfg *config.KafkaConfig, callback func(msg kafka.Message, err error)) *AsyncProducer {
	ctx, cancel := context.WithCancel(context.Background())
	return &AsyncProducer{
		config:    cfg,
		logger:    utils.NewLogger("[AsyncProducer]"),
		callback:  callback,
		ctx:       ctx,
		cancel:    cancel,
		msgChan:   make(chan kafka.Message, 1000), // 缓冲通道
		batchSize: 100,
	}
}

// Connect 连接到Kafka
func (p *AsyncProducer) Connect() error {
	p.writer = &kafka.Writer{
		Addr:         kafka.TCP(p.config.Brokers...),
		Topic:        p.config.Topic,
		Balancer:     &kafka.LeastBytes{}, // 使用最小字节分区器
		RequiredAcks: kafka.RequireOne,    // 只需leader确认
		Async:        true,                // 异步模式
		WriteTimeout: 10 * time.Second,
		BatchTimeout: 50 * time.Millisecond, // 更短的批量超时，提高实时性
		BatchSize:    p.batchSize,
		BatchBytes:   1048576,

		// 错误处理回调
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			p.logger.Error(fmt.Sprintf(msg, args...))
		}),
	}

	// 启动后台发送协程
	p.wg.Add(1)
	go p.processMessages()

	p.logger.Info("异步生产者连接成功")
	return nil
}

// SendAsync 异步发送消息，非阻塞
func (p *AsyncProducer) SendAsync(key, value string) error {
	select {
	case p.msgChan <- kafka.Message{
		Key:   []byte(key),
		Value: []byte(value),
		Time:  time.Now(),
	}:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("producer is closed")
	default:
		return fmt.Errorf("message channel is full")
	}
}

// SendAsyncWithCallback 异步发送并立即回调
func (p *AsyncProducer) SendAsyncWithCallback(key, value string, cb func(error)) {
	go func() {
		err := p.writer.WriteMessages(p.ctx, kafka.Message{
			Key:   []byte(key),
			Value: []byte(value),
			Time:  time.Now(),
		})
		if cb != nil {
			cb(err)
		}
		if p.callback != nil {
			p.callback(kafka.Message{Key: []byte(key), Value: []byte(value)}, err)
		}
	}()
}

// processMessages 后台处理消息发送
func (p *AsyncProducer) processMessages() {
	defer p.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]kafka.Message, 0, p.batchSize)

	for {
		select {
		case msg := <-p.msgChan:
			batch = append(batch, msg)

			// 批量达到阈值，立即发送
			if len(batch) >= p.batchSize {
				p.flushBatch(batch)
				batch = batch[:0] // 清空切片但保留容量
			}

		case <-ticker.C:
			// 定时刷新，避免消息滞留
			if len(batch) > 0 {
				p.flushBatch(batch)
				batch = batch[:0]
			}

		case <-p.ctx.Done():
			// 关闭前发送剩余消息
			if len(batch) > 0 {
				p.flushBatch(batch)
			}
			return
		}
	}
}

// flushBatch 批量发送消息
func (p *AsyncProducer) flushBatch(batch []kafka.Message) {
	if len(batch) == 0 {
		return
	}

	// 复制消息避免并发问题
	msgs := make([]kafka.Message, len(batch))
	copy(msgs, batch)

	go func(messages []kafka.Message) {
		err := p.writer.WriteMessages(context.Background(), messages...)

		// 触发回调
		if p.callback != nil {
			for _, msg := range messages {
				p.callback(msg, err)
			}
		}

		if err != nil {
			p.logger.Error("批量发送失败:", err)
		} else {
			p.logger.Info("批量发送成功，数量:", len(messages))
		}
	}(msgs)
}

// Close 关闭异步生产者
func (p *AsyncProducer) Close() error {
	p.cancel()  // 通知协程退出
	p.wg.Wait() // 等待后台协程完成

	if p.writer != nil {
		if err := p.writer.Close(); err != nil {
			return fmt.Errorf("关闭生产者失败: %w", err)
		}
	}

	close(p.msgChan)
	p.logger.Info("异步生产者已关闭")
	return nil
}

// Stats 获取统计信息
func (p *AsyncProducer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}

// PendingMessages 获取待发送消息数量
func (p *AsyncProducer) PendingMessages() int {
	return len(p.msgChan)
}
