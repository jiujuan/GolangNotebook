package tracer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// Tracer 消息追踪器
type Tracer struct {
	serviceName string
	traceIDKey  string
	spanIDKey   string
}

// Span 追踪跨度
type Span struct {
	TraceID   string            `json:"trace_id"`
	SpanID    string            `json:"span_id"`
	ParentID  string            `json:"parent_id,omitempty"`
	Service   string            `json:"service"`
	Operation string            `json:"operation"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time,omitempty"`
	Duration  int64             `json:"duration_ms"`
	Tags      map[string]string `json:"tags,omitempty"`
	Logs      []LogEntry        `json:"logs,omitempty"`
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Event     string                 `json:"event"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// NewTracer 创建追踪器
func NewTracer(serviceName string) *Tracer {
	return &Tracer{
		serviceName: serviceName,
		traceIDKey:  "trace-id",
		spanIDKey:   "span-id",
	}
}

// ExtractTraceContext 从消息中提取追踪上下文
func (t *Tracer) ExtractTraceContext(msg kafka.Message) context.Context {
	ctx := context.Background()

	for _, header := range msg.Headers {
		switch header.Key {
		case t.traceIDKey:
			ctx = context.WithValue(ctx, t.traceIDKey, string(header.Value))
		case t.spanIDKey:
			ctx = context.WithValue(ctx, t.spanIDKey, string(header.Value))
		}
	}

	return ctx
}

// InjectTraceContext 向消息中注入追踪上下文
func (t *Tracer) InjectTraceContext(ctx context.Context, msg *kafka.Message) {
	if traceID, ok := ctx.Value(t.traceIDKey).(string); ok && traceID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   t.traceIDKey,
			Value: []byte(traceID),
		})
	}

	if spanID, ok := ctx.Value(t.spanIDKey).(string); ok && spanID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   t.spanIDKey,
			Value: []byte(spanID),
		})
	}
}

// NewSpan 创建新的跨度
func (t *Tracer) NewSpan(ctx context.Context, operation string) *Span {
	traceID := t.getTraceID(ctx)
	if traceID == "" {
		traceID = t.generateID()
	}

	parentID := ""
	if id, ok := ctx.Value(t.spanIDKey).(string); ok {
		parentID = id
	}

	return &Span{
		TraceID:   traceID,
		SpanID:    t.generateID(),
		ParentID:  parentID,
		Service:   t.serviceName,
		Operation: operation,
		StartTime: time.Now(),
		Tags:      make(map[string]string),
		Logs:      make([]LogEntry, 0),
	}
}

// Finish 结束跨度
func (s *Span) Finish() {
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime).Milliseconds()
}

// SetTag 设置标签
func (s *Span) SetTag(key, value string) {
	s.Tags[key] = value
}

// LogEvent 记录事件
func (s *Span) LogEvent(event string, fields map[string]interface{}) {
	s.Logs = append(s.Logs, LogEntry{
		Timestamp: time.Now(),
		Event:     event,
		Fields:    fields,
	})
}

// Context 获取上下文
func (s *Span) Context() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace-id", s.TraceID)
	ctx = context.WithValue(ctx, "span-id", s.SpanID)
	return ctx
}

// ToJSON 转换为JSON
func (s *Span) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// getTraceID 从上下文获取TraceID
func (t *Tracer) getTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(t.traceIDKey).(string); ok {
		return id
	}
	return ""
}

// generateID 生成唯一ID（简化版，实际使用UUID）
func (t *Tracer) generateID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}

// TracedMessage 带追踪信息的消息
type TracedMessage struct {
	TraceID   string    `json:"trace_id"`
	SpanID    string    `json:"span_id"`
	Payload   []byte    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// NewTracedMessage 创建带追踪的消息
func (t *Tracer) NewTracedMessage(ctx context.Context, payload []byte) *TracedMessage {
	span := t.NewSpan(ctx, "produce")
	defer span.Finish()

	return &TracedMessage{
		TraceID:   span.TraceID,
		SpanID:    span.SpanID,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

// TracedHandler 带追踪的消息处理器
type TracedHandler struct {
	tracer  *Tracer
	handler func(ctx context.Context, msg *TracedMessage) error
}

func NewTracedHandler(tracer *Tracer, handler func(ctx context.Context, msg *TracedMessage) error) *TracedHandler {
	return &TracedHandler{
		tracer:  tracer,
		handler: handler,
	}
}

func (th *TracedHandler) Handle(msg kafka.Message) error {
	ctx := th.tracer.ExtractTraceContext(msg)
	span := th.tracer.NewSpan(ctx, "consume")
	defer span.Finish()

	var tracedMsg TracedMessage
	if err := json.Unmarshal(msg.Value, &tracedMsg); err != nil {
		// 如果不是TracedMessage格式，创建一个新的
		tracedMsg = TracedMessage{
			TraceID:   span.TraceID,
			SpanID:    span.SpanID,
			Payload:   msg.Value,
			Timestamp: msg.Time,
		}
	}

	span.SetTag("message.offset", fmt.Sprintf("%d", msg.Offset))
	span.SetTag("message.partition", fmt.Sprintf("%d", msg.Partition))

	ctx = span.Context()
	return th.handler(ctx, &tracedMsg)
}

// TraceReporter 追踪报告器接口
type TraceReporter interface {
	Report(span *Span)
}

// ConsoleReporter 控制台报告器
type ConsoleReporter struct{}

func (c *ConsoleReporter) Report(span *Span) {
	data, _ := json.MarshalIndent(span, "", "  ")
	fmt.Printf("[Trace] %s\\n", string(data))
}
