package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"go-kafka/client"
	"go-kafka/config"
	"go-kafka/consumer"
	"go-kafka/middleware"
	"go-kafka/producer"
	"go-kafka/topic"
)

// Order 订单结构
type Order struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	Items     []Item    `json:"items"`
	CreatedAt time.Time `json:"created_at"`
}

type Item struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// OrderEvent 订单事件
type OrderEvent struct {
	Type      string    `json:"type"` // created, paid, shipped, cancelled
	OrderID   string    `json:"order_id"`
	Timestamp time.Time `json:"timestamp"`
	Data      Order     `json:"data"`
}

const (
	TopicOrders         = "orders"
	TopicOrderEvents    = "order-events"
	ConsumerGroupOrders = "order-service"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	cfg := &config.KafkaConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   TopicOrders,
		GroupID: ConsumerGroupOrders,
	}

	switch os.Args[1] {
	case "setup":
		setupTopics(cfg)
	case "producer":
		runOrderProducer(cfg)
	case "consumer":
		runOrderConsumer(cfg)
	case "analytics":
		runAnalyticsConsumer(cfg)
	case "advanced":
		runAdvancedClient(cfg)
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("使用方式:")
	fmt.Println("  go run examples/order_system.go setup      - 初始化Topic")
	fmt.Println("  go run examples/order_system.go producer   - 启动订单生产者")
	fmt.Println("  go run examples/order_system.go consumer   - 启动订单消费者")
	fmt.Println("  go run examples/order_system.go analytics  - 启动分析消费者")
	fmt.Println("  go run examples/order_system.go advanced   - 高级客户端示例")
}

// setupTopics 初始化所需的Topic
func setupTopics(cfg *config.KafkaConfig) {
	fmt.Println("初始化 Kafka Topics...")

	manager, err := topic.NewTopicManager(cfg)
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 创建订单Topic
	if err := manager.CreateTopicIfNotExists(ctx, TopicOrders, 3, 1, 7*24*60*60*1000); err != nil {
		log.Printf("创建 %s 失败: %v", TopicOrders, err)
	} else {
		fmt.Printf("✓ Topic '%s' 已就绪\\n", TopicOrders)
	}

	// 创建订单事件Topic
	if err := manager.CreateTopicIfNotExists(ctx, TopicOrderEvents, 3, 1, 30*24*60*60*1000); err != nil {
		log.Printf("创建 %s 失败: %v", TopicOrderEvents, err)
	} else {
		fmt.Printf("✓ Topic '%s' 已就绪\\n", TopicOrderEvents)
	}

	// 列出所有Topic
	topics, _ := manager.ListTopics()
	fmt.Println("\\n现有Topics:", topics)
}

// runOrderProducer 订单生产者
func runOrderProducer(cfg *config.KafkaConfig) {
	fmt.Println("启动订单生产者...")

	p := producer.NewBatchProducer(
		cfg,
		producer.WithBatchSize(10),
		producer.WithCompression(kafka.Snappy),
	)

	if err := p.Connect(); err != nil {
		log.Fatal("连接失败:", err)
	}
	defer p.Close()

	// 模拟生成订单
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	orderCount := 0

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			fmt.Println("\\n停止生产者...")
			return

		case <-ticker.C:
			order := generateRandomOrder(orderCount)
			event := OrderEvent{
				Type:      "created",
				OrderID:   order.ID,
				Timestamp: time.Now(),
				Data:      order,
			}

			data, _ := json.Marshal(event)

			if err := p.Send(order.ID, string(data)); err != nil {
				log.Printf("发送失败: %v", err)
			} else {
				orderCount++
				fmt.Printf("✓ 订单 #%s 已发送 (总计: %d)\\n", order.ID, orderCount)
			}
		}
	}
}

// runOrderConsumer 订单消费者
func runOrderConsumer(cfg *config.KafkaConfig) {
	fmt.Println("启动订单消费者...")
	fmt.Println("使用中间件: Recovery, Logger, Retry")

	// 创建中间件链
	chain := middleware.Chain(
		middleware.Recovery(),
		middleware.Logger(),
		middleware.Retry(3, 2*time.Second),
	)

	handler := chain(func(msg kafka.Message) error {
		var event OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return err
		}

		fmt.Printf("\\n📦 处理订单:\\n")
		fmt.Printf("   ID: %s\\n", event.Data.ID)
		fmt.Printf("   User: %s\\n", event.Data.UserID)
		fmt.Printf("   Amount: $%.2f\\n", event.Data.Amount)
		fmt.Printf("   Items: %d\\n", len(event.Data.Items))

		// 模拟处理时间
		time.Sleep(500 * time.Millisecond)

		return nil
	})

	c := consumer.NewSimpleConsumer(cfg, -1)
	if err := c.Connect(); err != nil {
		log.Fatal("连接失败:", err)
	}
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\\n正在停止消费者...")
		cancel()
	}()

	if err := c.Start(ctx, handler); err != nil {
		log.Printf("消费错误: %v", err)
	}
}

// runAnalyticsConsumer 分析消费者（使用消费者组）
func runAnalyticsConsumer(cfg *config.KafkaConfig) {
	fmt.Println("启动分析消费者（消费者组模式）...")

	// 统计信息
	stats := make(map[string]int)
	var mu sync.Mutex

	handler := func(msg kafka.Message) error {
		var event OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return err
		}

		mu.Lock()
		stats["total"]++
		stats[event.Type]++

		totalAmount := 0.0
		if event.Type == "created" {
			totalAmount += event.Data.Amount
		}
		mu.Unlock()

		// 每处理10条打印一次统计
		if stats["total"]%10 == 0 {
			fmt.Printf("\\n📊 统计报告:\\n")
			fmt.Printf("   总订单数: %d\\n", stats["total"])
			fmt.Printf("   事件类型: %v\\n", stats)
		}

		return nil
	}

	manager := consumer.NewConsumerGroupManager(cfg)

	// 启动2个消费者实例
	if err := manager.StartConsumers(2, handler); err != nil {
		log.Fatal("启动消费者失败:", err)
	}

	fmt.Println("启动了2个消费者实例，按 Ctrl+C 停止")

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\\n正在停止...")
	manager.StopAll()
}

// runAdvancedClient 高级客户端示例
func runAdvancedClient(cfg *config.KafkaConfig) {
	fmt.Println("运行高级客户端示例...")

	kc := client.NewClient(cfg)

	// 生产者示例
	producer, err := kc.Producer().
		WithBatchSize(50).
		Build()

	if err != nil {
		log.Fatal("创建生产者失败:", err)
	}
	defer producer.Close()

	// 发送一些订单
	for i := 0; i < 5; i++ {
		order := generateRandomOrder(i)
		if err := producer.Send(context.Background(), order.ID, order); err != nil {
			log.Printf("发送失败: %v", err)
		}
	}

	fmt.Println("5个订单已发送")

	// 消费者示例
	consumer, err := kc.Consumer("advanced-consumer-group").
		Use(middleware.Logger()).
		Use(middleware.Recovery()).
		Use(middleware.Timeout(5 * time.Second)).
		Build()

	if err != nil {
		log.Fatal("创建消费者失败:", err)
	}
	defer consumer.Close()

	// 消费3条消息
	msgCount := 0
	consumer.Handle(func(key string, data interface{}) error {
		msgCount++
		fmt.Printf("收到订单: key=%s, data=%+v\\n", key, data)
		if msgCount >= 3 {
			return fmt.Errorf("enough messages consumed")
		}
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	consumer.Start(ctx)
}

// generateRandomOrder 生成随机订单
func generateRandomOrder(seq int) Order {
	items := []Item{
		{ProductID: fmt.Sprintf("PROD-%d", seq*2), Name: "iPhone", Quantity: 1, Price: 999.99},
		{ProductID: fmt.Sprintf("PROD-%d", seq*2+1), Name: "AirPods", Quantity: 1, Price: 199.99},
	}

	total := 0.0
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}

	return Order{
		ID:        fmt.Sprintf("ORD-%s-%d", time.Now().Format("20060102150405"), seq),
		UserID:    fmt.Sprintf("USER-%d", seq%100),
		Amount:    total,
		Status:    "created",
		Items:     items,
		CreatedAt: time.Now(),
	}
}
