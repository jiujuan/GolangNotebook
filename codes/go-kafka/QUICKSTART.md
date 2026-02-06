# 快速开始指南

## 1. 环境准备

### 1.1 启动 Kafka

```bash
# 使用项目提供的 docker-compose
docker-compose up -d

# 等待服务启动
sleep 10

# 验证状态
docker-compose ps
```

### 1.2 安装依赖

```bash
# 下载依赖
go mod download

# 验证安装
go build ./...
```

## 2. 基础用法

### 2.1 发送第一条消息

```go
package main

import (
    "context"
    "log"
    
    "go-kafka/config"
    "go-kafka/producer"
)

func main() {
    // 1. 创建配置
    cfg := &config.KafkaConfig{
        Brokers: []string{"localhost:9092"},
        Topic:   "hello-world",
    }
    
    // 2. 创建生产者
    p := producer.NewSimpleProducer(cfg)
    if err := p.Connect(); err != nil {
        log.Fatal(err)
    }
    defer p.Close()
    
    // 3. 发送消息
    ctx := context.Background()
    if err := p.SendMessage(ctx, "key1", "Hello, Kafka!"); err != nil {
        log.Fatal(err)
    }
    
    log.Println("消息发送成功！")
}
```

运行：
```bash
go run main.go
```

### 2.2 接收消息

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os/signal"
    "syscall"
    
    "github.com/segmentio/kafka-go"
    "go-kafka/config"
    "go-kafka/consumer"
)

func main() {
    cfg := &config.KafkaConfig{
        Brokers: []string{"localhost:9092"},
        Topic:   "hello-world",
        GroupID: "my-group",
    }
    
    c := consumer.NewSimpleConsumer(cfg, -1)
    if err := c.Connect(); err != nil {
        log.Fatal(err)
    }
    defer c.Close()
    
    // 设置信号处理
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigChan
        cancel()
    }()
    
    // 开始消费
    fmt.Println("开始消费消息，按 Ctrl+C 停止...")
    if err := c.Start(ctx, func(msg kafka.Message) error {
        fmt.Printf("收到: %s\\n", string(msg.Value))
        return nil
    }); err != nil {
        log.Fatal(err)
    }
}
```

## 3. 运行示例项目

### 3.1 订单系统示例

```bash
# 1. 初始化 Topic
make run-order-setup

# 2. 终端1：启动生产者
make run-order-producer

# 3. 终端2：启动消费者
make run-order-consumer

# 4. 终端3：启动分析消费者（消费者组）
go run examples/order_system.go analytics
```

### 3.2 使用 Makefile

```bash
# 查看所有命令
make help

# 常用命令
make docker-up          # 启动 Kafka
make docker-down        # 停止 Kafka
make topics-list        # 列出所有 Topics
make groups-list        # 列出消费者组
```

## 4. 关键配置

### 4.1 生产者配置

```go
// 高可靠性
p := producer.NewSimpleProducer(cfg)
// 或自定义配置
writer := &kafka.Writer{
    Addr:         kafka.TCP("localhost:9092"),
    Topic:        "my-topic",
    RequiredAcks: kafka.RequireAll,  // 等待所有副本确认
    MaxAttempts:  3,                  // 最大重试3次
}
```

### 4.2 消费者配置

```go
config := kafka.ReaderConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "my-topic",
    GroupID: "my-group",
    
    // 性能优化
    MinBytes: 1e3,        // 1KB
    MaxBytes: 10e6,       // 10MB
    MaxWait:  1*time.Second,
    
    // 可靠性
    CommitInterval: 0,    // 0表示手动提交
}
```

## 5. 常见问题

### Q1: 连接被拒绝？

```bash
# 检查 Kafka 是否运行
docker-compose ps

# 检查端口
telnet localhost 9092

# 查看日志
docker-compose logs kafka
```

### Q2: 消费者不消费消息？

```bash
# 检查消费者组
docker exec -it kafka kafka-consumer-groups \\
    --bootstrap-server localhost:9092 \\
    --describe \\
    --group my-group

# 重置偏移量
docker exec -it kafka kafka-consumer-groups \\
    --bootstrap-server localhost:9092 \\
    --group my-group \\
    --reset-offsets \\
    --to-earliest \\
    --execute \\
    --topic hello-world
```

### Q3: 消息丢失？

- 生产者：`RequiredAcks: kafka.RequireAll`
- 消费者：处理成功后提交偏移量
- Topic：`replication.factor >= 2`

## 6. 下一步

- [架构设计文档](ARCHITECTURE.md)
- [生产者详细文档](producer/)
- [消费者详细文档](consumer/)
- [示例代码](examples/)

## 7. 性能测试

```bash
# 生产者性能测试
make perf-producer

# 消费者性能测试
make perf-consumer

# 自定义测试
go test -bench=. -benchmem
```

## 8. 生产检查清单

- [ ] 配置了合适的副本因子（>=2）
- [ ] 启用了生产者的重试机制
- [ ] 消费者实现了幂等性处理
- [ ] 配置了监控和告警
- [ ] 设置了健康检查端点
- [ ] 实现了优雅关闭逻辑
- [ ] 配置了日志轮转
- [ ] 进行了压力测试