package middleware

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// HandlerFunc 处理函数类型
type HandlerFunc func(msg kafka.Message) error

// Middleware 中间件类型
type Middleware func(HandlerFunc) HandlerFunc

// Chain 中间件链
func Chain(middlewares ...Middleware) Middleware {
	return func(final HandlerFunc) HandlerFunc {
		return func(msg kafka.Message) error {
			handler := final
			for i := len(middlewares) - 1; i >= 0; i-- {
				handler = middlewares[i](handler)
			}
			return handler(msg)
		}
	}
}

// Recovery  panic 恢复中间件
func Recovery() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg kafka.Message) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic recovered: %v", r)
					log.Printf("[Recovery] panic: %v, key: %s", r, string(msg.Key))
				}
			}()
			return next(msg)
		}
	}
}

// Logger 日志中间件
func Logger() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg kafka.Message) error {
			start := time.Now()

			log.Printf("[Before] partition=%d, offset=%d, key=%s",
				msg.Partition, msg.Offset, string(msg.Key))

			err := next(msg)

			duration := time.Since(start)
			if err != nil {
				log.Printf("[Error] partition=%d, offset=%d, error=%v, duration=%v",
					msg.Partition, msg.Offset, err, duration)
			} else {
				log.Printf("[Success] partition=%d, offset=%d, duration=%v",
					msg.Partition, msg.Offset, duration)
			}

			return err
		}
	}
}

// Retry 重试中间件
func Retry(maxRetries int, delay time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg kafka.Message) error {
			var err error

			for i := 0; i <= maxRetries; i++ {
				err = next(msg)
				if err == nil {
					return nil
				}

				if i < maxRetries {
					log.Printf("[Retry] attempt %d/%d failed for key=%s: %v",
						i+1, maxRetries, string(msg.Key), err)
					time.Sleep(delay * time.Duration(i+1)) // 指数退避
				}
			}

			return fmt.Errorf("max retries exceeded: %w", err)
		}
	}
}

// Timeout 超时中间件
func Timeout(d time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg kafka.Message) error {
			ctx, cancel := context.WithTimeout(context.Background(), d)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				done <- next(msg)
			}()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				return fmt.Errorf("handler timeout after %v", d)
			}
		}
	}
}

// CircuitBreaker 熔断中间件（简化版）
type CircuitBreaker struct {
	failures    int
	threshold   int
	lastFailure time.Time
	resetTime   time.Duration
	state       string // "closed", "open", "half-open"
}

func NewCircuitBreaker(threshold int, resetTime time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold: threshold,
		resetTime: resetTime,
		state:     "closed",
	}
}

func (cb *CircuitBreaker) Middleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg kafka.Message) error {
			// 检查熔断状态
			if cb.state == "open" {
				if time.Since(cb.lastFailure) > cb.resetTime {
					cb.state = "half-open"
					cb.failures = 0
					log.Printf("[CircuitBreaker] entering half-open state")
				} else {
					return fmt.Errorf("circuit breaker is open")
				}
			}

			err := next(msg)

			if err != nil {
				cb.failures++
				cb.lastFailure = time.Now()

				if cb.failures >= cb.threshold {
					cb.state = "open"
					log.Printf("[CircuitBreaker] entering open state after %d failures", cb.failures)
				}
			} else if cb.state == "half-open" {
				cb.state = "closed"
				cb.failures = 0
				log.Printf("[CircuitBreaker] entering closed state")
			}

			return err
		}
	}
}

// DeadLetterQueue 死信队列中间件
type DeadLetterHandler interface {
	SendToDLQ(ctx context.Context, msg kafka.Message, err error) error
}

func DeadLetterQueue(dlqHandler DeadLetterHandler) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg kafka.Message) error {
			err := next(msg)
			if err != nil {
				log.Printf("[DLQ] sending message to DLQ: key=%s, error=%v",
					string(msg.Key), err)

				dlqErr := dlqHandler.SendToDLQ(context.Background(), msg, err)
				if dlqErr != nil {
					log.Printf("[DLQ] failed to send to DLQ: %v", dlqErr)
				}
			}
			return err
		}
	}
}

// RateLimiter 限流中间件
type RateLimiter struct {
	tokens   chan struct{}
	interval time.Duration
}

func NewRateLimiter(rate int, per time.Duration) *RateLimiter {
	rl := &RateLimiter{
		tokens:   make(chan struct{}, rate),
		interval: per / time.Duration(rate),
	}

	// 预填充令牌
	for i := 0; i < rate; i++ {
		rl.tokens <- struct{}{}
	}

	// 定时补充令牌
	go func() {
		ticker := time.NewTicker(rl.interval)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case rl.tokens <- struct{}{}:
			default:
			}
		}
	}()

	return rl
}

func (rl *RateLimiter) Middleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg kafka.Message) error {
			select {
			case <-rl.tokens:
				return next(msg)
			default:
				return fmt.Errorf("rate limit exceeded")
			}
		}
	}
}
