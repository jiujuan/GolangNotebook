# Go Kafka 实战开发

使用 [github.com/segmentio/kafka-go](https://github.com/segmentio/kafka-go) 实现 Kafka 常用功能的完整示例。

## 项目结构

```
go-kafka/
├── config/              # 配置管理
│   └── config.go
├── producer/            # 生产者实现
│   ├── simple_producer.go   # 同步生产者
│   ├── async_producer.go    # 异步生产者
│   └── batch_producer.go    # 批量生产者
├── consumer/            # 消费者实现
│   ├── simple_consumer.go   # 简单消费者
│   ├── group_consumer.go    # 消费者组
│   └── manual_commit.go     # 手动提交
├── topic/               # Topic 管理
│   └── topic_manager.go
├── admin/               # 管理操作
│   └── admin_ops.go
├── examples/            # 使用示例
│   ├── producer_example.go
│   └── consumer_example.go
└── utils/               # 工具函数
    └── logger.go
```

## 快速开始

### 1. 环境准备

```bash
# 使用 Docker 启动 Kafka
$ docker-compose -f docker-compose.yml up -d
```

### 2. 安装依赖

```bash
$ go mod tidy
```

### 3. 运行示例

```bash
# 运行生产者示例
$ go run examples/producer_example.go

# 运行消费者示例
$ go run examples/consumer_example.go
```

### 4. 环境变量配置

```bash
# 设置 Kafka 地址（多个用逗号分隔）
$ export KAFKA_BROKERS=localhost:9092,localhost:9093

# 设置默认 Topic
$ export KAFKA_TOPIC=my-topic

# 设置消费者组 ID
$ export KAFKA_GROUP_ID=my-group
```

## 功能模块

### 生产者 (Producer)

#### 1. 简单同步生产者
```go
p := producer.NewSimpleProducer(cfg)
p.Connect()
defer p.Close()

// 发送单条消息
p.SendMessage(ctx, "key", "value")

// 发送带 Headers 的消息
headers := map[string]string{"source": "app1"}
p.SendMessageWithHeaders(ctx, "key", "value", headers)

// 批量发送
messages := []struct{Key, Value string}{{"k1","v1"}, {"k2","v2"}}
p.SendMessagesBatch(ctx, messages)
```

#### 2. 异步生产者
```go
// 定义回调函数
callback := func(msg kafka.Message, err error) {
    if err != nil {
        log.Printf("发送失败: %v", err)
    }
}

p := producer.NewAsyncProducer(cfg, callback)
p.Connect()

// 异步发送（非阻塞）
p.SendAsync("key", "value")
```

#### 3. 批量生产者
```go
// 配置批量大小和压缩算法
p := producer.NewBatchProducer(
    cfg,
    producer.WithBatchSize(500),
    producer.WithCompression(kafka.Lz4),
)
p.Connect()

// 发送消息（自动批量处理）
p.Send("key", "value")
p.SendStructured("key", data)

// 手动刷新缓冲区
p.Flush()
```

### 消费者 (Consumer)

#### 1. 简单消费者
```go
c := consumer.NewSimpleConsumer(cfg, -1) // -1 表示不指定分区
c.Connect()

handler := func(msg kafka.Message) error {
    log.Printf("收到消息: %s", string(msg.Value))
    return nil
}

c.Start(ctx, handler)
```

#### 2. 消费者组
```go
manager := consumer.NewConsumerGroupManager(cfg)

// 启动3个消费者实例
manager.StartConsumers(3, handler)

// 停止所有消费者
manager.StopAll()
```

#### 3. 手动提交消费者
```go
// 每50条提交一次
c := consumer.NewManualCommitConsumer(cfg, 50)
c.Connect()

// 处理消息，失败不提交
c.Start(ctx, handler)

// 手动提交
c.Commit(ctx)
```

### Topic 管理

```go
manager, _ := topic.NewTopicManager(cfg)
defer manager.Close()

// 创建 Topic
ctx := context.Background()
manager.CreateTopic(ctx, "my-topic", 3, 1, 86400000) // 1天保留期

// 列出所有 Topic
topics, _ := manager.ListTopics()

// 查看 Topic 详情
info, _ := manager.DescribeTopic("my-topic")

// 删除 Topic
manager.DeleteTopic(ctx, "my-topic")

// 获取分区偏移量
oldest, newest, _ := manager.GetPartitionOffsets("my-topic", 0)
```

### 管理操作

```go
admin := admin.NewAdminClient(cfg)

// 获取集群信息
info, _ := admin.GetClusterInfo()

// 列出消费者组
groups, _ := admin.ListConsumerGroups()

// 获取分区详情
partitions, _ := admin.GetPartitionDetails("my-topic")

// 重置消费者组偏移量
admin.ResetConsumerGroupOffset("my-group", "my-topic", 0, 100)
```

## 高级特性

### 压缩支持
```go
// 支持 Gzip, Snappy, Lz4, Zstd
producer.WithCompression(kafka.Lz4)
```

### 分区策略
- `Hash`: 基于 key 的哈希
- `LeastBytes`: 最小字节数
- `RoundRobin`: 轮询
- `CRC32Balancer`: CRC32 哈希（与 Java 兼容）

### 安全配置（可选）
```go
// TLS 配置
Dialer: &kafka.Dialer{
    TLS: &tls.Config{...},
}

// SASL 认证
Dialer: &kafka.Dialer{
    SASLMechanism: plain.Mechanism{
        Username: "user",
        Password: "pass",
    },
}
```

## 性能优化建议

1. **生产者优化**
   - 使用批量生产者处理大数据量
   - 启用压缩（推荐 Lz4）
   - 调整 `BatchSize` 和 `BatchTimeout`

2. **消费者优化**
   - 使用消费者组实现负载均衡
   - 根据处理能力调整 `MaxBytes`
   - 批量处理消息后提交

3. **监控**
   ```go
   // 获取生产者统计
   stats := producer.Stats()
   fmt.Printf("消息数: %d, 错误数: %d", stats.Messages, stats.Errors)
   
   // 获取消费延迟
   lag, _ := consumer.FetchLag(ctx)
   fmt.Printf("延迟: %d", lag.Lag)
   ```

## 错误处理

所有操作都返回详细的错误信息，建议分类处理：

```go
if err != nil {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        // 超时处理
    case errors.Is(err, kafka.UnknownTopicOrPartition):
        // Topic不存在
    default:
        // 其他错误
    }
}
```

## 贡献

欢迎提交 Issue 和 PR！

## 许可证

MIT License