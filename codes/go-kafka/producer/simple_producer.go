package producer

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go-kafka/config"
	"go-kafka/utils"
)

// SimpleProducer 简单同步生产者
type SimpleProducer struct {
	writer *kafka.Writer
	config *config.KafkaConfig
	logger *utils.Logger
}

// NewSimpleProducer 创建简单生产者
func NewSimpleProducer(cfg *config.KafkaConfig) *SimpleProducer {
	return &SimpleProducer{
		config: cfg,
		logger: utils.NewLogger("[SimpleProducer]"),
	}
}

// Connect 连接到Kafka
func (p *SimpleProducer) Connect() error {
	p.writer = &kafka.Writer{
		Addr:     kafka.TCP(p.config.Brokers...),
		Topic:    p.config.Topic,
		Balancer: &kafka.Hash{}, // 使用Hash分区器，确保相同key的消息进入同一分区

		// 写入配置
		RequiredAcks: kafka.RequireAll, // 需要所有副本确认
		Async:        false,            // 同步模式

		// 超时配置
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,

		// 重试配置
		MaxAttempts:  3,
		BatchTimeout: 100 * time.Millisecond,
		BatchSize:    100,
		BatchBytes:   1048576, // 1MB
	}

	p.logger.Info("生产者连接成功，brokers:", p.config.Brokers)
	return nil
}

// SendMessage 发送单条消息
func (p *SimpleProducer) SendMessage(ctx context.Context, key, value string) error {
	msg := kafka.Message{
		Key:   []byte(key),
		Value: []byte(value),
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}

	p.logger.Info("消息发送成功, key:", key)
	return nil
}

// SendMessageWithHeaders 发送带消息头的消息
func (p *SimpleProducer) SendMessageWithHeaders(
	ctx context.Context,
	key, value string,
	headers map[string]string,
) error {
	kafkaHeaders := make([]kafka.Header, 0, len(headers))
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	msg := kafka.Message{
		Key:     []byte(key),
		Value:   []byte(value),
		Headers: kafkaHeaders,
		Time:    time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}

	p.logger.Info("消息发送成功(带Headers), key:", key)
	return nil
}

// SendMessagesBatch 批量发送消息
func (p *SimpleProducer) SendMessagesBatch(ctx context.Context, messages []struct{ Key, Value string }) error {
	kafkaMessages := make([]kafka.Message, len(messages))
	for i, msg := range messages {
		kafkaMessages[i] = kafka.Message{
			Key:   []byte(msg.Key),
			Value: []byte(msg.Value),
			Time:  time.Now(),
		}
	}

	if err := p.writer.WriteMessages(ctx, kafkaMessages...); err != nil {
		return fmt.Errorf("批量发送消息失败: %w", err)
	}

	p.logger.Info("批量消息发送成功，数量:", len(messages))
	return nil
}

// Close 关闭生产者
func (p *SimpleProducer) Close() error {
	if p.writer != nil {
		if err := p.writer.Close(); err != nil {
			return fmt.Errorf("关闭生产者失败: %w", err)
		}
		p.logger.Info("生产者已关闭")
	}
	return nil
}

// Stats 获取生产者统计信息
func (p *SimpleProducer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}
