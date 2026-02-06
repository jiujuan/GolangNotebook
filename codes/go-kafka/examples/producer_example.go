package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go-kafka/config"
	"go-kafka/producer"
)

func main() {
	// 加载配置
	cfg := config.LoadFromEnv()

	fmt.Println("========================================")
	fmt.Println("     Kafka 生产者示例")
	fmt.Println("========================================")
	fmt.Println("1. 简单生产者 (同步)")
	fmt.Println("2. 异步生产者")
	fmt.Println("3. 批量生产者")
	fmt.Println("4. 性能测试")
	fmt.Println("========================================")

	// 示例1: 简单生产者
	runSimpleProducer(cfg)

	// 示例2: 异步生产者
	// runAsyncProducer(cfg)

	// 示例3: 批量生产者
	// runBatchProducer(cfg)
}

// runSimpleProducer 简单生产者示例
func runSimpleProducer(cfg *config.KafkaConfig) {
	fmt.Println("\\n>>> 运行简单生产者示例")

	p := producer.NewSimpleProducer(cfg)
	if err := p.Connect(); err != nil {
		fmt.Println("连接失败:", err)
		return
	}
	defer p.Close()

	ctx := context.Background()

	// 发送单条消息
	if err := p.SendMessage(ctx, "key1", "Hello, Kafka! 这是第一条消息"); err != nil {
		fmt.Println("发送失败:", err)
		return
	}

	// 发送带Headers的消息
	headers := map[string]string{
		"source":    "producer-example",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	if err := p.SendMessageWithHeaders(ctx, "key2", "带消息头的消息", headers); err != nil {
		fmt.Println("发送失败:", err)
		return
	}

	// 批量发送
	messages := []struct{ Key, Value string }{
		{"batch1", "批量消息1"},
		{"batch2", "批量消息2"},
		{"batch3", "批量消息3"},
	}
	if err := p.SendMessagesBatch(ctx, messages); err != nil {
		fmt.Println("批量发送失败:", err)
		return
	}

	fmt.Println("所有消息发送成功!")
	fmt.Printf("生产者统计: %+v\\n", p.Stats())
}

// runAsyncProducer 异步生产者示例
func runAsyncProducer(cfg *config.KafkaConfig) {
	fmt.Println("\\n>>> 运行异步生产者示例")

	// 定义回调函数
	callback := func(msg kafka.Message, err error) {
		if err != nil {
			fmt.Printf("消息发送失败: key=%s, error=%v\\n", string(msg.Key), err)
		} else {
			fmt.Printf("消息发送成功: key=%s\\n", string(msg.Key))
		}
	}

	p := producer.NewAsyncProducer(cfg, callback)
	if err := p.Connect(); err != nil {
		fmt.Println("连接失败:", err)
		return
	}
	defer p.Close()

	// 异步发送消息
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("async-key-%d", i)
		value := fmt.Sprintf("异步消息 %d", i)

		if err := p.SendAsync(key, value); err != nil {
			fmt.Println("发送失败:", err)
		}
	}

	fmt.Println("100条消息已加入发送队列")
	fmt.Printf("待发送消息数: %d\\n", p.PendingMessages())

	// 等待一段时间让消息发送完成
	time.Sleep(2 * time.Second)
}

// runBatchProducer 批量生产者示例
func runBatchProducer(cfg *config.KafkaConfig) {
	fmt.Println("\\n>>> 运行批量生产者示例")

	p := producer.NewBatchProducer(
		cfg,
		producer.WithBatchSize(50),
		producer.WithCompression(kafka.Lz4),
	)

	if err := p.Connect(); err != nil {
		fmt.Println("连接失败:", err)
		return
	}
	defer p.Close()

	// 发送大量消息
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("batch-key-%d", i%10)
		value := fmt.Sprintf("批量生产消息 %d - %s", i, time.Now().Format(time.RFC3339))

		if err := p.Send(key, value); err != nil {
			fmt.Println("发送失败:", err)
		}

		// 每100条打印一次状态
		if i%100 == 0 {
			fmt.Printf("已发送: %d, 缓冲区: %d\\n", i, p.BufferSize())
		}
	}

	// 手动刷新确保所有消息发送
	if err := p.Flush(); err != nil {
		fmt.Println("刷新失败:", err)
	}

	fmt.Println("1000条消息发送完成!")
	fmt.Printf("最终统计: %+v\\n", p.Stats())
}

// runProducerBenchmark 生产者性能测试
func runProducerBenchmark(cfg *config.KafkaConfig) {
	fmt.Println("\\n>>> 运行生产者性能测试")

	p := producer.NewBatchProducer(cfg)
	if err := p.Connect(); err != nil {
		fmt.Println("连接失败:", err)
		return
	}
	defer p.Close()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// 生产者协程
	wg.Add(1)
	go func() {
		defer wg.Done()

		msgCount := 0
		start := time.Now()

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				duration := time.Since(start)
				fmt.Printf("\\n测试完成！发送 %d 条消息，耗时 %v，速率 %.2f msg/s\\n",
					msgCount, duration, float64(msgCount)/duration.Seconds())
				return
			case <-ticker.C:
				duration := time.Since(start)
				fmt.Printf("\\r已发送: %d 条消息，速率: %.2f msg/s",
					msgCount, float64(msgCount)/duration.Seconds())
			default:
				key := fmt.Sprintf("bench-%d", msgCount)
				value := fmt.Sprintf("性能测试消息 %d", msgCount)

				if err := p.Send(key, value); err != nil {
					fmt.Println("\\n发送失败:", err)
					return
				}
				msgCount++
			}
		}
	}()

	// 等待信号
	<-sigChan
	cancel()
	wg.Wait()
}
