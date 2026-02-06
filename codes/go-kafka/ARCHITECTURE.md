# Go-Kafka 架构设计文档

## 1. 项目概述

本项目是一个基于 `github.com/segmentio/kafka-go` 的 Go 语言 Kafka 客户端库，提供了完整的消息队列操作能力，包括生产者、消费者、Topic管理、监控追踪等功能。

## 2. 架构设计

### 2.1 模块结构

```
go-kafka/
├── config/          # 配置管理
├── producer/        # 生产者实现
├── consumer/        # 消费者实现
├── topic/           # Topic管理
├── admin/           # 集群管理
├── client/          # 高级客户端封装
├── middleware/      # 中间件系统
├── serializer/      # 序列化工具
├── pool/            # 连接池
├── health/          # 健康检查
├── metrics/         # 监控指标
├── tracer/          # 分布式追踪
└── utils/           # 工具函数
```

### 2.2 核心组件关系

```
                    ┌─────────────────────────────────────┐
                    │           Application               │
                    └──────────────┬──────────────────────┘
                                   │
              ┌────────────────────┼────────────────────┐
              │                    │                    │
              ▼                    ▼                    ▼
       ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
       │   Client    │     │  Middleware │     │  Tracer     │
       └──────┬──────┘     └──────┬──────┘     └──────┬──────┘
              │                    │                    │
       ┌──────┴──────┐            │             ┌─────┴──────┐
       │             │            │             │            │
       ▼             ▼            ▼             ▼            ▼
 ┌─────────┐  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
 │Producer │  │ Consumer │ │   Pool   │ │ Metrics  │ │  Health  │
 └────┬────┘  └─────┬────┘ └─────┬────┘ └─────┬────┘ └─────┬────┘
      │             │            │            │            │
      └─────────────┴────────────┴────────────┴────────────┘
                              │
                              ▼
                    ┌───────────────────┐
                    │   Kafka Cluster   │
                    └───────────────────┘
```

## 3. 核心功能实现

### 3.1 生产者设计

#### 3.1.1 生产者类型

1. **SimpleProducer** (同步生产者)
   - 特性：同步发送、即时确认、适合低延迟场景
   - 配置：`RequiredAcks: kafka.RequireAll`
   - 使用场景：订单创建、支付确认等关键业务

2. **AsyncProducer** (异步生产者)
   - 特性：异步发送、回调机制、高吞吐
   - 配置：`Async: true`
   - 使用场景：日志收集、埋点上报

3. **BatchProducer** (批量生产者)
   - 特性：自动批处理、压缩支持、可控刷新
   - 配置：`BatchSize: 500`, `Compression: kafka.Lz4`
   - 使用场景：大数据量写入、定时报表

#### 3.1.2 分区策略

```go
// Hash 分区 - 基于Key的哈希，保证相同Key进入同一分区
Balancer: &kafka.Hash{}

// 最小字节分区 - 写入消息最少的分区
Balancer: &kafka.LeastBytes{}

// 轮询分区 - 均匀分配
Balancer: &kafka.RoundRobin{}

// CRC32 分区 - 与Java客户端兼容
Balancer: &kafka.CRC32Balancer{}
```

### 3.2 消费者设计

#### 3.2.1 消费者类型

1. **SimpleConsumer** (简单消费者)
   - 支持指定分区或消费者组
   - 自动重平衡
   - 手动/自动偏移量提交

2. **GroupConsumer** (消费者组)
   - 多实例负载均衡
   - 支持 `Range` 和 `RoundRobin` 分配策略
   - 再平衡监听

3. **ManualCommitConsumer** (手动提交消费者)
   - 精确控制偏移量提交
   - 业务成功后提交
   - 支持批量提交

#### 3.2.2 消费模式

```go
// 推模式（回调）
consumer.Start(ctx, func(msg kafka.Message) error {
    // 处理消息
    return nil
})

// 拉模式（主动获取）
msg, err := consumer.ReadMessage(ctx)
```

### 3.3 中间件系统

#### 3.3.1 中间件链

```go
chain := middleware.Chain(
    middleware.Recovery(),        // 恢复panic
    middleware.Logger(),          // 日志记录
    middleware.Retry(3, 2*time.Second),  // 重试机制
    middleware.Timeout(5*time.Second),   // 超时控制
    middleware.DeadLetterQueue(dlq),     // 死信队列
)
```

#### 3.3.2 自定义中间件

```go
func MyMiddleware() middleware.Middleware {
    return func(next consumer.MessageHandler) consumer.MessageHandler {
        return func(msg kafka.Message) error {
            // 前置处理
            start := time.Now()
            
            err := next(msg)  // 调用下一个处理器
            
            // 后置处理
            duration := time.Since(start)
            recordMetrics(duration, err)
            
            return err
        }
    }
}
```

## 4. 高级功能

### 4.1 连接池

```go
pool := pool.NewConnPool(
    cfg,
    pool.WithMaxConns(20),
    pool.WithIdleConns(5),
    pool.WithMaxIdleTime(30*time.Minute),
)

conn := pool.Get()
defer pool.Put(conn)
```

### 4.2 序列化

```go
// JSON序列化
serializer := &serializer.JSONSerializer{}
data, _ := serializer.Serialize(order)

// 自定义序列化
type ProtoSerializer struct{}
func (s *ProtoSerializer) Serialize(data interface{}) ([]byte, error) {
    return proto.Marshal(data.(proto.Message))
}
```

### 4.3 分布式追踪

```go
tracer := tracer.NewTracer("order-service")

// 生产者注入追踪信息
span := tracer.NewSpan(ctx, "produce-order")
defer span.Finish()
msg := tracer.NewTracedMessage(span.Context(), data)

// 消费者提取追踪信息
handler := tracer.NewTracedHandler(tracer, func(ctx context.Context, msg *tracer.TracedMessage) error {
    span := tracer.NewSpan(ctx, "process-order")
    defer span.Finish()
    // 处理消息
    return nil
})
```

### 4.4 监控指标

```go
metrics := metrics.NewMetrics()
metrics.RegisterHandler(metrics.NewLoggerHandler(log.Default()))
metrics.RegisterHandler(metrics.NewPrometheusHandler("kafka"))
metrics.Start(60 * time.Second)

// 包装生产者
instrumentedProducer := metrics.NewInstrumentedProducer(producer, metrics)
```

## 5. 最佳实践

### 5.1 生产者最佳实践

1. **选择合适的生产者类型**
   - 关键业务：SimpleProducer + RequireAll
   - 高吞吐场景：BatchProducer + Lz4压缩
   - 实时性要求：AsyncProducer

2. **合理配置批量参数**
   ```go
   BatchSize:    500,              // 根据消息大小调整
   BatchTimeout: 100 * time.Millisecond,
   BatchBytes:   10 * 1024 * 1024, // 10MB
   ```

3. **错误处理**
   ```go
   if writeErrors, ok := err.(kafka.WriteErrors); ok {
       for i := range msgs {
           if writeErrors[i] != nil {
               // 处理失败的消息
           }
       }
   }
   ```

### 5.2 消费者最佳实践

1. **优雅关闭**
   ```go
   sigChan := make(chan os.Signal, 1)
   signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
   
   <-sigChan
   cancel()  // 取消上下文
   consumer.Close()  // 关闭连接
   ```

2. **幂等性设计**
   - 消息去重：基于业务ID
   - 状态检查：处理前检查是否已处理
   - 数据库唯一约束

3. **消费延迟监控**
   ```go
   lag, _ := consumer.FetchLag(ctx)
   if lag.Lag > 1000 {
       // 触发告警
   }
   ```

### 5.3 运维最佳实践

1. **健康检查端点**
   ```go
   hc := health.NewHealthChecker(cfg, 30*time.Second)
   http.Handle("/health", hc.HTTPHandler())
   ```

2. **动态配置**
   - 使用环境变量
   - 支持配置热更新
   - 配置验证

3. **日志规范**
   ```go
   // 结构化日志
   log.Printf(`{"level":"info","topic":"%s","partition":%d,"offset":%d}`,
       msg.Topic, msg.Partition, msg.Offset)
   ```

## 6. 性能优化

### 6.1 生产者优化

| 配置项 | 默认值 | 优化建议 |
|--------|--------|----------|
| BatchSize | 100 | 根据消息大小调整至 100-1000 |
| BatchBytes | 1MB | 增加至 10MB-32MB |
| LingerMs | 0 | 设置为 5-100ms |
| Compression | none | 使用 lz4 或 snappy |

### 6.2 消费者优化

| 配置项 | 默认值 | 优化建议 |
|--------|--------|----------|
| MinBytes | 1 | 增加至 1KB-1MB |
| MaxBytes | 1MB | 增加至 10MB-50MB |
| MaxWait | 500ms | 增加至 1-5s |
| MaxProcessingTime | 100ms | 根据业务调整 |

## 7. 故障处理

### 7.1 常见问题

1. **消息丢失**
   - 生产者：配置 `RequireAll`
   - 消费者：手动提交 + 业务成功后提交

2. **消息重复**
   - 生产者：启用幂等性（Idempotent Producer）
   - 消费者：业务幂等性设计

3. **消费延迟**
   - 增加消费者实例
   - 优化消息处理逻辑
   - 增加分区数

4. **重平衡风暴**
   - 调整 `session.timeout.ms`
   - 确保处理时间小于超时时间
   - 使用静态成员（Static Membership）

### 7.2 监控告警

```yaml
alerts:
  - name: HighLag
    condition: consumer_lag > 10000
    duration: 5m
    
  - name: ConsumerOffline
    condition: consumer_count < expected_count
    duration: 1m
    
  - name: ProduceErrorRate
    condition: produce_error_rate > 1%
    duration: 5m
```

## 8. 扩展功能

### 8.1 Schema Registry 集成

```go
registry := confluent.NewSchemaRegistry("http://localhost:8081")

// 注册schema
schema := `{"type":"record","name":"Order",...}`
registry.Register("orders-value", schema)

// 序列化时使用schema
serializer := avro.NewSerializer(registry)
```

### 8.2 Kafka Streams 替代方案

```go
// 使用本库实现简单的流处理
stream := kafka.NewStream()
stream.From("input-topic").
    Filter(func(msg Message) bool { return msg.Value != nil }).
    Map(func(msg Message) Message { /* 转换 */ return msg }).
    To("output-topic")
```

## 9. 部署架构

### 9.1 单机开发

```yaml
# docker-compose.yml 已提供
- Zookeeper: 2181
- Kafka: 9092
- Kafka UI: 8080
```

### 9.2 生产部署

```
                    ┌─────────────┐
                    │  Load Balancer  │
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
    ┌──────▼──────┐ ┌──────▼──────┐ ┌──────▼──────┐
    │  App Instance 1 │ │  App Instance 2 │ │  App Instance 3 │
    │  (Consumer 1)   │ │  (Consumer 2)   │ │  (Consumer 3)   │
    └──────┬──────┘ └──────┬──────┘ └──────┬──────┘
           │               │               │
           └───────────────┼───────────────┘
                           │
                    ┌──────▼──────┐
                    │ Kafka Cluster  │
                    │ (3 Brokers)    │
                    └───────────────┘
```

## 10. 版本兼容性

| 组件 | 版本 | 说明 |
|------|------|------|
| Go | 1.21+ | 最低要求 |
| kafka-go | 0.4.x | 当前使用 |
| Kafka Broker | 2.0+ | 推荐 2.8+ |
| Zookeeper | 3.4+ | 可选（使用KRaft模式） |

---

## 附录

### A. 配置参数参考

[详见 config/config.go ]

### B. API 文档

[详见各个模块的代码注释]

### C. 性能测试报告

[详见 test/benchmark_test.go ]