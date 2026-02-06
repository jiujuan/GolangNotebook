package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go-kafka/config"
)

// ConnPool 连接池
type ConnPool struct {
	config      *config.KafkaConfig
	maxConns    int
	idleConns   int
	maxIdleTime time.Duration

	pool    chan *kafka.Conn
	mu      sync.RWMutex
	closed  bool
	factory func() (*kafka.Conn, error)
}

// PoolOption 连接池配置选项
type PoolOption func(*ConnPool)

// WithMaxConns 设置最大连接数
func WithMaxConns(n int) PoolOption {
	return func(p *ConnPool) {
		p.maxConns = n
	}
}

// WithIdleConns 设置空闲连接数
func WithIdleConns(n int) PoolOption {
	return func(p *ConnPool) {
		p.idleConns = n
	}
}

// WithMaxIdleTime 设置最大空闲时间
func WithMaxIdleTime(d time.Duration) PoolOption {
	return func(p *ConnPool) {
		p.maxIdleTime = d
	}
}

// NewConnPool 创建连接池
func NewConnPool(cfg *config.KafkaConfig, options ...PoolOption) *ConnPool {
	pool := &ConnPool{
		config:      cfg,
		maxConns:    10,
		idleConns:   3,
		maxIdleTime: 30 * time.Minute,
		pool:        make(chan *kafka.Conn, 10),
	}

	// 应用选项
	for _, opt := range options {
		opt(pool)
	}

	// 设置连接工厂
	pool.factory = func() (*kafka.Conn, error) {
		return kafka.Dial("tcp", cfg.Brokers[0])
	}

	// 初始化连接
	for i := 0; i < pool.idleConns; i++ {
		if conn, err := pool.factory(); err == nil {
			pool.pool <- conn
		}
	}

	// 启动清理协程
	go pool.cleanup()

	return pool
}

// Get 获取连接
func (p *ConnPool) Get() (*kafka.Conn, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.RUnlock()

	select {
	case conn := <-p.pool:
		if p.isValid(conn) {
			return conn, nil
		}
		conn.Close()
		return p.factory()
	case <-time.After(time.Second):
		return p.factory()
	}
}

// Put 归还连接
func (p *ConnPool) Put(conn *kafka.Conn) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		conn.Close()
		return
	}
	p.mu.RUnlock()

	select {
	case p.pool <- conn:
	default:
		conn.Close()
	}
}

// isValid 检查连接是否有效
func (p *ConnPool) isValid(conn *kafka.Conn) bool {
	// 简单检查，实际可能需要更复杂的健康检查
	if conn == nil {
		return false
	}
	return true
}

// cleanup 清理过期连接
func (p *ConnPool) cleanup() {
	ticker := time.NewTicker(p.maxIdleTime / 2)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.RLock()
		if p.closed {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()

		// 清理多余连接
		for len(p.pool) > p.idleConns {
			select {
			case conn := <-p.pool:
				conn.Close()
			default:
				break
			}
		}
	}
}

// Close 关闭连接池
func (p *ConnPool) Close() error {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()

	close(p.pool)
	for conn := range p.pool {
		conn.Close()
	}

	return nil
}

// Stats 获取连接池统计
func (p *ConnPool) Stats() map[string]interface{} {
	return map[string]interface{}{
		"available": len(p.pool),
		"max":       p.maxConns,
		"idle":      p.idleConns,
	}
}

// WriterPool Writer连接池
type WriterPool struct {
	pool    map[string]*kafka.Writer
	mu      sync.RWMutex
	config  *config.KafkaConfig
	options []kafka.WriterOption
}

// NewWriterPool 创建Writer池
func NewWriterPool(cfg *config.KafkaConfig, options ...kafka.WriterOption) *WriterPool {
	return &WriterPool{
		pool:    make(map[string]*kafka.Writer),
		config:  cfg,
		options: options,
	}
}

// GetWriter 获取Writer
func (wp *WriterPool) GetWriter(topic string) *kafka.Writer {
	wp.mu.RLock()
	writer, exists := wp.pool[topic]
	wp.mu.RUnlock()

	if exists {
		return writer
	}

	wp.mu.Lock()
	defer wp.mu.Unlock()

	// 双重检查
	if writer, exists := wp.pool[topic]; exists {
		return writer
	}

	// 创建新Writer
	writer = &kafka.Writer{
		Addr:  kafka.TCP(wp.config.Brokers...),
		Topic: topic,
	}

	wp.pool[topic] = writer
	return writer
}

// Close 关闭所有Writer
func (wp *WriterPool) Close() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	for _, writer := range wp.pool {
		writer.Close()
	}

	return nil
}
