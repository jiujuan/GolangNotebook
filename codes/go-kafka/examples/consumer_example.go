package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"go-kafka/config"
	"go-kafka/consumer"
)

func main() {
	// 加载配置
	cfg := config.LoadFromEnv()

	fmt.Println("========================================")
	fmt.Println("     Kafka 消费者示例")
	fmt.Println("========================================")
	fmt.Println("1. 简单消费者")
	fmt.Println("2. 消费者组")
	fmt.Println("3. 手动提交消费者")
	fmt.Println("========================================")

	// 示例1: 简单消费者
	runSimpleConsumer(cfg)

	// 示例2: 消费者组
	// runGroupConsumer(cfg)

	// 示例3: 手动提交消费者
	// runManualCommitConsumer(cfg)
}

// runSimpleConsumer 简单消费者示例
func runSimpleConsumer(cfg *config.KafkaConfig) {
	fmt.Println("\\n>>> 运行简单消费者示例")
	fmt.Println("按 Ctrl+C 停止消费")

	// 创建消费者，-1表示不指定分区（使用消费者组）
	c := consumer.NewSimpleConsumer(cfg, -1)
	if err := c.Connect(); err != nil {
		fmt.Println("连接失败:", err)
		return
	}
	defer c.Close()

	// 定义消息处理函数
	handler := func(msg kafka.Message) error {
		fmt.Printf("收到消息: partition=%d, offset=%d, key=%s, value=%s\\n",
			msg.Partition,
			msg.Offset,
			string(msg.Key),
			string(msg.Value))

		// 模拟业务处理时间
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\\n收到停止信号...")
		cancel()
	}()

	// 开始消费
	if err := c.Start(ctx, handler); err != nil {
		fmt.Println("消费错误:", err)
	}
}

// runGroupConsumer 消费者组示例
func runGroupConsumer(cfg *config.KafkaConfig) {
	fmt.Println("\\n>>> 运行消费者组示例")
	fmt.Println("启动3个消费者实例，按 Ctrl+C 停止")

	manager := consumer.NewConsumerGroupManager(cfg)

	// 定义消息处理函数
	handler := func(msg kafka.Message) error {
		fmt.Printf("[%s] 处理消息: partition=%d, offset=%d, value=%s\\n",
			time.Now().Format("15:04:05"),
			msg.Partition,
			msg.Offset,
			string(msg.Value))

		// 模拟处理时间
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	// 启动3个消费者实例
	if err := manager.StartConsumers(3, handler); err != nil {
		fmt.Println("启动消费者失败:", err)
		return
	}

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\\n正在停止所有消费者...")
	manager.StopAll()
	fmt.Println("所有消费者已停止")
}

// runManualCommitConsumer 手动提交消费者示例
func runManualCommitConsumer(cfg *config.KafkaConfig) {
	fmt.Println("\\n>>> 运行手动提交消费者示例")
	fmt.Println("确保消息处理完成才提交，按 Ctrl+C 停止")

	// 创建手动提交消费者，每50条提交一次
	c := consumer.NewManualCommitConsumer(cfg, 50)
	if err := c.Connect(); err != nil {
		fmt.Println("连接失败:", err)
		return
	}
	defer c.Close()

	// 消息处理函数
	handler := func(msg kafka.Message) error {
		fmt.Printf("处理消息: partition=%d, offset=%d, value=%s\\n",
			msg.Partition,
			msg.Offset,
			string(msg.Value))

		// 模拟业务处理
		time.Sleep(100 * time.Millisecond)

		// 模拟偶数失败
		if msg.Offset%10 == 0 {
			fmt.Printf("  [模拟失败] offset=%d 处理失败，将重试\\n", msg.Offset)
			return fmt.Errorf("处理失败")
		}

		fmt.Printf("  [成功] offset=%d 处理完成\\n", msg.Offset)
		return nil
	}

	// 设置上下文和信号处理
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := c.Start(ctx, handler); err != nil {
			fmt.Println("消费错误:", err)
		}
	}()

	// 定时打印未提交数量
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				fmt.Printf("[状态] 未提交消息数: %d\\n", c.GetUncommittedCount())
			case <-ctx.Done():
				return
			}
		}
	}()

	// 等待停止信号
	<-sigChan
	fmt.Println("\\n正在停止...")
	cancel()
	wg.Wait()

	fmt.Printf("最终未提交消息数: %d\\n", c.GetUncommittedCount())
}

// runLagMonitor 消费延迟监控示例
func runLagMonitor(cfg *config.KafkaConfig) {
	fmt.Println("\\n>>> 运行消费延迟监控示例")

	c := consumer.NewSimpleConsumer(cfg, -1)
	if err := c.Connect(); err != nil {
		fmt.Println("连接失败:", err)
		return
	}
	defer c.Close()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		lag, err := c.FetchLag(context.Background())
		if err != nil {
			fmt.Println("获取延迟失败:", err)
			continue
		}

		fmt.Printf("[%s] 消费延迟: partition=%d, lag=%d, offset=%d\\n",
			time.Now().Format("15:04:05"),
			lag.Partition,
			lag.Lag,
			lag.Offset)
	}
}
