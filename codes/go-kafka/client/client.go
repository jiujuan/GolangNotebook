package client

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"go-kafka/config"
	"go-kafka/consumer"
	"go-kafka/middleware"
	"go-kafka/producer"
	"go-kafka/serializer"
)

// KafkaClient 高级Kafka客户端
type KafkaClient struct {
	config     *config.KafkaConfig
	serializer serializer.Serializer
}

// NewClient 创建客户端
func NewClient(cfg *config.KafkaConfig) *KafkaClient {
	return &KafkaClient{
		config:     cfg,
		serializer: &serializer.JSONSerializer{},
	}
}

// SetSerializer 设置序列化器
func (c *KafkaClient) SetSerializer(s serializer.Serializer) {
	c.serializer = s
}

// ProducerBuilder 生产者构建器
type ProducerBuilder struct {
	client    *KafkaClient
	batchSize int
	async     bool
	compress  kafka.Compression
}

// Producer 开始构建生产者
func (c *KafkaClient) Producer() *ProducerBuilder {
	return &ProducerBuilder{
		client:   c,
		async:    false,
		compress: kafka.Lz4,
	}
}

// WithBatchSize 设置批量大小
func (pb *ProducerBuilder) WithBatchSize(size int) *ProducerBuilder {
	pb.batchSize = size
	return pb
}

// Async 设置异步模式
func (pb *ProducerBuilder) Async() *ProducerBuilder {
	pb.async = true
	return pb
}

// Build 构建生产者
func (pb *ProducerBuilder) Build() (*ProducerWrapper, error) {
	var p interface{}

	if pb.async {
		ap := producer.NewAsyncProducer(pb.client.config, nil)
		if err := ap.Connect(); err != nil {
			return nil, err
		}
		p = ap
	} else if pb.batchSize > 0 {
		bp := producer.NewBatchProducer(
			pb.client.config,
			producer.WithBatchSize(pb.batchSize),
			producer.WithCompression(pb.compress),
		)
		if err := bp.Connect(); err != nil {
			return nil, err
		}
		p = bp
	} else {
		sp := producer.NewSimpleProducer(pb.client.config)
		if err := sp.Connect(); err != nil {
			return nil, err
		}
		p = sp
	}

	return &ProducerWrapper{
		producer:   p,
		serializer: pb.client.serializer,
	}, nil
}

// ProducerWrapper 生产者包装器
type ProducerWrapper struct {
	producer   interface{}
	serializer serializer.Serializer
}

// Send 发送消息
func (pw *ProducerWrapper) Send(ctx context.Context, key string, data interface{}) error {
	value, err := pw.serializer.Serialize(data)
	if err != nil {
		return err
	}

	switch p := pw.producer.(type) {
	case *producer.SimpleProducer:
		return p.SendMessage(ctx, key, string(value))
	case *producer.AsyncProducer:
		return p.SendAsync(key, string(value))
	case *producer.BatchProducer:
		return p.Send(key, string(value))
	default:
		return fmt.Errorf("unknown producer type")
	}
}

// Close 关闭生产者
func (pw *ProducerWrapper) Close() error {
	switch p := pw.producer.(type) {
	case *producer.SimpleProducer:
		return p.Close()
	case *producer.AsyncProducer:
		return p.Close()
	case *producer.BatchProducer:
		return p.Close()
	default:
		return nil
	}
}

// ConsumerBuilder 消费者构建器
type ConsumerBuilder struct {
	client       *KafkaClient
	groupID      string
	manualCommit bool
	middlewares  []middleware.Middleware
}

// Consumer 开始构建消费者
func (c *KafkaClient) Consumer(groupID string) *ConsumerBuilder {
	return &ConsumerBuilder{
		client:  c,
		groupID: groupID,
	}
}

// ManualCommit 设置手动提交
func (cb *ConsumerBuilder) ManualCommit() *ConsumerBuilder {
	cb.manualCommit = true
	return cb
}

// Use 添加中间件
func (cb *ConsumerBuilder) Use(mws ...middleware.Middleware) *ConsumerBuilder {
	cb.middlewares = append(cb.middlewares, mws...)
	return cb
}

// Build 构建消费者
func (cb *ConsumerBuilder) Build() (*ConsumerWrapper, error) {
	cfg := &config.KafkaConfig{
		Brokers: cb.client.config.Brokers,
		Topic:   cb.client.config.Topic,
		GroupID: cb.groupID,
	}

	var c interface{}

	if cb.manualCommit {
		mc := consumer.NewManualCommitConsumer(cfg, 100)
		if err := mc.Connect(); err != nil {
			return nil, err
		}
		c = mc
	} else {
		sc := consumer.NewSimpleConsumer(cfg, -1)
		if err := sc.Connect(); err != nil {
			return nil, err
		}
		c = sc
	}

	return &ConsumerWrapper{
		consumer:    c,
		middlewares: cb.middlewares,
		serializer:  cb.client.serializer,
	}, nil
}

// ConsumerWrapper 消费者包装器
type ConsumerWrapper struct {
	consumer    interface{}
	middlewares []middleware.Middleware
	serializer  serializer.Serializer
	handler     consumer.MessageHandler
}

// Handle 设置处理器
func (cw *ConsumerWrapper) Handle(handler interface{}) *ConsumerWrapper {
	// 包装中间件
	var final middleware.HandlerFunc = func(msg kafka.Message) error {
		return cw.handleMessage(msg, handler)
	}

	// 应用中间件链
	for i := len(cw.middlewares) - 1; i >= 0; i-- {
		final = cw.middlewares[i](final)
	}

	cw.handler = final
	return cw
}

// handleMessage 处理消息
func (cw *ConsumerWrapper) handleMessage(msg kafka.Message, handler interface{}) error {
	if h, ok := handler.(func(kafka.Message) error); ok {
		return h(msg)
	}
	if h, ok := handler.(func(string, interface{}) error); ok {
		var data interface{}
		if err := cw.serializer.Deserialize(msg.Value, &data); err != nil {
			return err
		}
		return h(string(msg.Key), data)
	}
	return fmt.Errorf("unsupported handler type")
}

// Start 开始消费
func (cw *ConsumerWrapper) Start(ctx context.Context) error {
	switch c := cw.consumer.(type) {
	case *consumer.ManualCommitConsumer:
		return c.Start(ctx, cw.handler)
	case *consumer.SimpleConsumer:
		return c.Start(ctx, cw.handler)
	default:
		return fmt.Errorf("unknown consumer type")
	}
}

// Close 关闭消费者
func (cw *ConsumerWrapper) Close() error {
	switch c := cw.consumer.(type) {
	case *consumer.ManualCommitConsumer:
		return c.Close()
	case *consumer.SimpleConsumer:
		return c.Close()
	default:
		return nil
	}
}

// ClientExample 使用示例
/*
func Example() {
	cfg := config.LoadFromEnv()
	client := client.NewClient(cfg)

	// 生产者示例
	producer, _ := client.Producer().
		WithBatchSize(100).
		Async().
		Build()

	defer producer.Close()

	data := map[string]string{"name": "test", "value": "123"}
	producer.Send(context.Background(), "key1", data)

	// 消费者示例
	consumer, _ := client.Consumer("my-group").
		ManualCommit().
		Use(middleware.Logger()).
		Use(middleware.Recovery()).
		Use(middleware.Retry(3, 2*time.Second)).
		Build()

	defer consumer.Close()

	consumer.Handle(func(msg kafka.Message) error {
		fmt.Printf("Received: %s\\n", string(msg.Value))
		return nil
	}).Start(context.Background())
}
*/
