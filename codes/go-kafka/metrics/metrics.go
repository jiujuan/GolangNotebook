package metrics

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
)

// Metrics 指标收集器
type Metrics struct {
	// 生产者指标
	MessagesProduced uint64
	BytesProduced    uint64
	ProduceErrors    uint64
	ProduceLatency   int64 // 纳秒

	// 消费者指标
	MessagesConsumed uint64
	BytesConsumed    uint64
	ConsumeErrors    uint64
	ConsumeLatency   int64
	CurrentLag       int64

	// 连接器指标
	ConnectionErrors uint64
	RebalanceEvents  uint64

	mu       sync.RWMutex
	handlers []MetricsHandler
	started  bool
	ctx      context.Context
	cancel   context.CancelFunc
}

// MetricsHandler 指标处理器接口
type MetricsHandler interface {
	Handle(m *Metrics)
}

// NewMetrics 创建指标收集器
func NewMetrics() *Metrics {
	ctx, cancel := context.WithCancel(context.Background())
	return &Metrics{
		ctx:    ctx,
		cancel: cancel,
	}
}

// RegisterHandler 注册指标处理器
func (m *Metrics) RegisterHandler(handler MetricsHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

// Start 启动指标报告
func (m *Metrics) Start(interval time.Duration) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.mu.Unlock()

	if interval == 0 {
		interval = 60 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.report()
		}
	}
}

// Stop 停止指标报告
func (m *Metrics) Stop() {
	m.cancel()
}

// report 报告指标
func (m *Metrics) report() {
	m.mu.RLock()
	handlers := make([]MetricsHandler, len(m.handlers))
	copy(handlers, m.handlers)
	m.mu.RUnlock()

	for _, h := range handlers {
		h.Handle(m)
	}
}

// RecordProduced 记录生产消息
func (m *Metrics) RecordProduced(bytes int, latency time.Duration) {
	atomic.AddUint64(&m.MessagesProduced, 1)
	atomic.AddUint64(&m.BytesProduced, uint64(bytes))
	atomic.AddInt64(&m.ProduceLatency, latency.Nanoseconds())
}

// RecordProduceError 记录生产错误
func (m *Metrics) RecordProduceError() {
	atomic.AddUint64(&m.ProduceErrors, 1)
}

// RecordConsumed 记录消费消息
func (m *Metrics) RecordConsumed(bytes int, latency time.Duration) {
	atomic.AddUint64(&m.MessagesConsumed, 1)
	atomic.AddUint64(&m.BytesConsumed, uint64(bytes))
	atomic.AddInt64(&m.ConsumeLatency, latency.Nanoseconds())
}

// RecordConsumeError 记录消费错误
func (m *Metrics) RecordConsumeError() {
	atomic.AddUint64(&m.ConsumeErrors, 1)
}

// UpdateLag 更新消费延迟
func (m *Metrics) UpdateLag(lag int64) {
	atomic.StoreInt64(&m.CurrentLag, lag)
}

// Snapshot 获取指标快照
func (m *Metrics) Snapshot() map[string]interface{} {
	return map[string]interface{}{
		"messages_produced": atomic.LoadUint64(&m.MessagesProduced),
		"bytes_produced":    atomic.LoadUint64(&m.BytesProduced),
		"produce_errors":    atomic.LoadUint64(&m.ProduceErrors),
		"messages_consumed": atomic.LoadUint64(&m.MessagesConsumed),
		"bytes_consumed":    atomic.LoadUint64(&m.BytesConsumed),
		"consume_errors":    atomic.LoadUint64(&m.ConsumeErrors),
		"current_lag":       atomic.LoadInt64(&m.CurrentLag),
	}
}

// Reset 重置指标
func (m *Metrics) Reset() {
	atomic.StoreUint64(&m.MessagesProduced, 0)
	atomic.StoreUint64(&m.BytesProduced, 0)
	atomic.StoreUint64(&m.ProduceErrors, 0)
	atomic.StoreInt64(&m.ProduceLatency, 0)
	atomic.StoreUint64(&m.MessagesConsumed, 0)
	atomic.StoreUint64(&m.BytesConsumed, 0)
	atomic.StoreUint64(&m.ConsumeErrors, 0)
	atomic.StoreInt64(&m.ConsumeLatency, 0)
	atomic.StoreInt64(&m.CurrentLag, 0)
}

// String 返回字符串表示
func (m *Metrics) String() string {
	snapshot := m.Snapshot()
	return fmt.Sprintf(
		"Messages: [P:%d/C:%d], Bytes: [P:%d/C:%d], Errors: [P:%d/C:%d], Lag: %d",
		snapshot["messages_produced"],
		snapshot["messages_consumed"],
		snapshot["bytes_produced"],
		snapshot["bytes_consumed"],
		snapshot["produce_errors"],
		snapshot["consume_errors"],
		snapshot["current_lag"],
	)
}

// LoggerHandler 日志指标处理器
type LoggerHandler struct {
	logger *log.Logger
}

func NewLoggerHandler(logger *log.Logger) *LoggerHandler {
	return &LoggerHandler{logger: logger}
}

func (h *LoggerHandler) Handle(m *Metrics) {
	h.logger.Printf("[Metrics] %s", m.String())
}

// PrometheusHandler Prometheus指标处理器（示例）
type PrometheusHandler struct {
	prefix string
}

func NewPrometheusHandler(prefix string) *PrometheusHandler {
	if prefix == "" {
		prefix = "kafka"
	}
	return &PrometheusHandler{prefix: prefix}
}

func (h *PrometheusHandler) Handle(m *Metrics) {
	// 实际项目中这里会更新Prometheus指标
	// 例如：h.messagesProduced.Set(float64(m.MessagesProduced))
	snapshot := m.Snapshot()
	fmt.Printf("# %s_messages_produced %v\\n", h.prefix, snapshot["messages_produced"])
	fmt.Printf("# %s_messages_consumed %v\\n", h.prefix, snapshot["messages_consumed"])
}

// InstrumentedProducer 带指标的生产者包装器
type InstrumentedProducer struct {
	producer interface {
		WriteMessages(ctx context.Context, msgs ...kafka.Message) error
		Close() error
	}
	metrics *Metrics
}

func NewInstrumentedProducer(producer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}, metrics *Metrics) *InstrumentedProducer {
	return &InstrumentedProducer{
		producer: producer,
		metrics:  metrics,
	}
}

func (p *InstrumentedProducer) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	start := time.Now()

	totalBytes := 0
	for _, msg := range msgs {
		totalBytes += len(msg.Value)
	}

	err := p.producer.WriteMessages(ctx, msgs...)

	latency := time.Since(start)

	if err != nil {
		p.metrics.RecordProduceError()
	} else {
		p.metrics.RecordProduced(totalBytes, latency)
	}

	return err
}

func (p *InstrumentedProducer) Close() error {
	return p.producer.Close()
}

// InstrumentedConsumer 带指标的消费者包装器
type InstrumentedConsumer struct {
	consumer interface {
		ReadMessage(ctx context.Context) (kafka.Message, error)
		Close() error
	}
	metrics *Metrics
}

func NewInstrumentedConsumer(consumer interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	Close() error
}, metrics *Metrics) *InstrumentedConsumer {
	return &InstrumentedConsumer{
		consumer: consumer,
		metrics:  metrics,
	}
}

func (c *InstrumentedConsumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	start := time.Now()

	msg, err := c.consumer.ReadMessage(ctx)

	latency := time.Since(start)

	if err != nil {
		c.metrics.RecordConsumeError()
	} else {
		c.metrics.RecordConsumed(len(msg.Value), latency)
	}

	return msg, err
}

func (c *InstrumentedConsumer) Close() error {
	return c.consumer.Close()
}
