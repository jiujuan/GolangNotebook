package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go-kafka/config"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	config   *config.KafkaConfig
	status   map[string]HealthStatus
	mu       sync.RWMutex
	checkers []Checker
	interval time.Duration
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status    string    `json:"status"` // healthy, degraded, unhealthy
	LastCheck time.Time `json:"last_check"`
	Message   string    `json:"message,omitempty"`
	Latency   int64     `json:"latency_ms"`
}

// Checker 健康检查接口
type Checker interface {
	Name() string
	Check(ctx context.Context) HealthStatus
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(cfg *config.KafkaConfig, interval time.Duration) *HealthChecker {
	if interval == 0 {
		interval = 30 * time.Second
	}

	hc := &HealthChecker{
		config:   cfg,
		status:   make(map[string]HealthStatus),
		interval: interval,
	}

	// 注册默认检查器
	hc.Register(&BrokerChecker{config: cfg})
	hc.Register(&TopicChecker{config: cfg})
	hc.Register(&LatencyChecker{config: cfg})

	return hc
}

// Register 注册检查器
func (hc *HealthChecker) Register(checker Checker) {
	hc.checkers = append(hc.checkers, checker)
}

// Start 启动健康检查
func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	// 立即执行一次检查
	hc.runChecks(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.runChecks(ctx)
		}
	}
}

// runChecks 执行所有检查
func (hc *HealthChecker) runChecks(ctx context.Context) {
	for _, checker := range hc.checkers {
		status := checker.Check(ctx)
		hc.mu.Lock()
		hc.status[checker.Name()] = status
		hc.mu.Unlock()
	}
}

// GetStatus 获取健康状态
func (hc *HealthChecker) GetStatus() map[string]HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make(map[string]HealthStatus)
	for k, v := range hc.status {
		result[k] = v
	}
	return result
}

// IsHealthy 是否健康
func (hc *HealthChecker) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	for _, status := range hc.status {
		if status.Status == "unhealthy" {
			return false
		}
	}
	return len(hc.status) > 0
}

// HTTPHandler HTTP健康检查端点
func (hc *HealthChecker) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := hc.GetStatus()

		healthy := hc.IsHealthy()

		w.Header().Set("Content-Type", "application/json")
		if !healthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		fmt.Fprintf(w, `{"healthy":%t,"checks":%v}`, healthy, status)
	}
}

// BrokerChecker Broker连接检查
type BrokerChecker struct {
	config *config.KafkaConfig
}

func (bc *BrokerChecker) Name() string {
	return "brokers"
}

func (bc *BrokerChecker) Check(ctx context.Context) HealthStatus {
	start := time.Now()

	for _, broker := range bc.config.Brokers {
		conn, err := kafka.DialContext(ctx, "tcp", broker)
		if err != nil {
			return HealthStatus{
				Status:  "unhealthy",
				Message: fmt.Sprintf("无法连接到 broker: %s", broker),
			}
		}
		conn.Close()
	}

	return HealthStatus{
		Status:    "healthy",
		LastCheck: time.Now(),
		Latency:   time.Since(start).Milliseconds(),
		Message:   fmt.Sprintf("已连接 %d 个 brokers", len(bc.config.Brokers)),
	}
}

// TopicChecker Topic存在性检查
type TopicChecker struct {
	config *config.KafkaConfig
}

func (tc *TopicChecker) Name() string {
	return "topic"
}

func (tc *TopicChecker) Check(ctx context.Context) HealthStatus {
	if tc.config.Topic == "" {
		return HealthStatus{
			Status:  "degraded",
			Message: "未配置默认topic",
		}
	}

	start := time.Now()

	conn, err := kafka.Dial("tcp", tc.config.Brokers[0])
	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "无法连接Kafka",
		}
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions(tc.config.Topic)
	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Topic '%s' 不存在", tc.config.Topic),
		}
	}

	return HealthStatus{
		Status:    "healthy",
		LastCheck: time.Now(),
		Latency:   time.Since(start).Milliseconds(),
		Message:   fmt.Sprintf("Topic '%s' 有 %d 个分区", tc.config.Topic, len(partitions)),
	}
}

// LatencyChecker 延迟检查
type LatencyChecker struct {
	config *config.KafkaConfig
}

func (lc *LatencyChecker) Name() string {
	return "latency"
}

func (lc *LatencyChecker) Check(ctx context.Context) HealthStatus {
	start := time.Now()

	// 尝试发送一个心跳消息
	writer := &kafka.Writer{
		Addr:  kafka.TCP(lc.config.Brokers...),
		Topic: lc.config.Topic,
	}
	defer writer.Close()

	err := writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte("health-check"),
		Value: []byte("ping"),
	})

	latency := time.Since(start)

	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("写入测试消息失败: %v", err),
		}
	}

	status := "healthy"
	if latency > 1*time.Second {
		status = "degraded"
	}

	return HealthStatus{
		Status:    status,
		LastCheck: time.Now(),
		Latency:   latency.Milliseconds(),
		Message:   fmt.Sprintf("写入延迟: %v", latency),
	}
}
