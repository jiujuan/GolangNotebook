package kafka_test

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"go-kafka/config"
	"go-kafka/consumer"
	"go-kafka/producer"
)

// TestSimpleProducer 测试简单生产者
func TestSimpleProducer(t *testing.T) {
	cfg := &config.KafkaConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
	}

	p := producer.NewSimpleProducer(cfg)
	if err := p.Connect(); err != nil {
		t.Skip("无法连接Kafka，跳过测试:", err)
	}
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 测试发送消息
	err := p.SendMessage(ctx, "test-key", "test-value")
	if err != nil {
		t.Errorf("发送消息失败: %v", err)
	}
}

// TestSimpleConsumer 测试简单消费者
func TestSimpleConsumer(t *testing.T) {
	cfg := &config.KafkaConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	c := consumer.NewSimpleConsumer(cfg, -1)
	if err := c.Connect(); err != nil {
		t.Skip("无法连接Kafka，跳过测试:", err)
	}
	defer c.Close()

	// 测试读取消息（超时模式）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan bool)
	go func() {
		_, err := c.ReadMessage(ctx)
		if err != nil {
			// 超时是预期的
			t.Logf("读取消息超时（预期）: %v", err)
		}
		done <- true
	}()

	select {
	case <-done:
		// 测试通过
	case <-time.After(10 * time.Second):
		t.Error("消费者测试超时")
	}
}

// TestBatchProducer 测试批量生产者
func TestBatchProducer(t *testing.T) {
	cfg := &config.KafkaConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-batch-topic",
	}

	p := producer.NewBatchProducer(cfg)
	if err := p.Connect(); err != nil {
		t.Skip("无法连接Kafka，跳过测试:", err)
	}
	defer p.Close()

	// 发送多条消息
	for i := 0; i < 10; i++ {
		err := p.Send("batch-key", "batch-value")
		if err != nil {
			t.Errorf("发送消息失败: %v", err)
		}
	}

	// 强制刷新
	err := p.Flush()
	if err != nil {
		t.Errorf("刷新失败: %v", err)
	}
}

// TestMiddlewareChain 测试中间件链
func TestMiddlewareChain(t *testing.T) {
	callOrder := []string{}

	m1 := func(next func(kafka.Message) error) func(kafka.Message) error {
		return func(msg kafka.Message) error {
			callOrder = append(callOrder, "m1-before")
			err := next(msg)
			callOrder = append(callOrder, "m1-after")
			return err
		}
	}

	m2 := func(next func(kafka.Message) error) func(kafka.Message) error {
		return func(msg kafka.Message) error {
			callOrder = append(callOrder, "m2-before")
			err := next(msg)
			callOrder = append(callOrder, "m2-after")
			return err
		}
	}

	chain := func(final func(kafka.Message) error) func(kafka.Message) error {
		return func(msg kafka.Message) error {
			handler := final
			handler = m1(handler)
			handler = m2(handler)
			return handler(msg)
		}
	}

	handler := chain(func(msg kafka.Message) error {
		callOrder = append(callOrder, "handler")
		return nil
	})

	msg := kafka.Message{Value: []byte("test")}
	handler(msg)

	// 验证调用顺序
	expected := []string{"m2-before", "m1-before", "handler", "m1-after", "m2-after"}
	for i, v := range expected {
		if i >= len(callOrder) || callOrder[i] != v {
			t.Errorf("中间件调用顺序错误，期望 %v，得到 %v", expected, callOrder)
			break
		}
	}
}

// BenchmarkProducer 生产者性能测试
func BenchmarkProducer(b *testing.B) {
	cfg := &config.KafkaConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "benchmark-topic",
	}

	p := producer.NewBatchProducer(cfg)
	if err := p.Connect(); err != nil {
		b.Skip("无法连接Kafka，跳过测试:", err)
	}
	defer p.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Send("benchmark-key", "benchmark-value")
	}
	p.Flush()
}

// IntegrationTest 集成测试
func IntegrationTest(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	cfg := &config.KafkaConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "integration-test-topic",
		GroupID: "integration-test-group",
	}

	// 生产者
	p := producer.NewSimpleProducer(cfg)
	if err := p.Connect(); err != nil {
		t.Skip("无法连接Kafka:", err)
	}

	// 发送测试消息
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := p.SendMessage(ctx, "integration-key", "integration-value"); err != nil {
			t.Fatalf("发送消息失败: %v", err)
		}
	}
	p.Close()

	// 消费者
	received := 0
	done := make(chan bool)

	c := consumer.NewSimpleConsumer(cfg, -1)
	if err := c.Connect(); err != nil {
		t.Fatalf("连接消费者失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		c.Start(ctx, func(msg kafka.Message) error {
			received++
			t.Logf("收到消息: %s", string(msg.Value))
			if received >= 5 {
				done <- true
			}
			return nil
		})
	}()

	select {
	case <-done:
		t.Logf("成功消费 %d 条消息", received)
	case <-ctx.Done():
		t.Fatalf("测试超时，只收到 %d 条消息", received)
	}

	c.Close()
}
