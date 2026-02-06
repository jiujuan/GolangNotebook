package serializer

import (
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// Serializer 消息序列化接口
type Serializer interface {
	Serialize(data interface{}) ([]byte, error)
	Deserialize(data []byte, v interface{}) error
}

// JSONSerializer JSON序列化
type JSONSerializer struct{}

func (s *JSONSerializer) Serialize(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (s *JSONSerializer) Deserialize(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// XMLSerializer XML序列化
type XMLSerializer struct{}

func (s *XMLSerializer) Serialize(data interface{}) ([]byte, error) {
	return xml.Marshal(data)
}

func (s *XMLSerializer) Deserialize(data []byte, v interface{}) error {
	return xml.Unmarshal(data, v)
}

// StringSerializer 字符串序列化
type StringSerializer struct{}

func (s *StringSerializer) Serialize(data interface{}) ([]byte, error) {
	switch v := data.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	default:
		return []byte(fmt.Sprintf("%v", v)), nil
	}
}

func (s *StringSerializer) Deserialize(data []byte, v interface{}) error {
	if ptr, ok := v.(*string); ok {
		*ptr = string(data)
		return nil
	}
	return fmt.Errorf("destination must be *string")
}

// Message 通用消息结构
type Message struct {
	Topic     string            `json:"topic,omitempty"`
	Key       string            `json:"key,omitempty"`
	Data      interface{}       `json:"data"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timestamp int64             `json:"timestamp"`
}

// EncodeMessage 编码消息
func EncodeMessage(msg Message, serializer Serializer) (kafka.Message, error) {
	value, err := serializer.Serialize(msg.Data)
	if err != nil {
		return kafka.Message{}, err
	}

	kafkaMsg := kafka.Message{
		Key:   []byte(msg.Key),
		Value: value,
	}

	// 添加消息头
	for k, v := range msg.Headers {
		kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	return kafkaMsg, nil
}

// DecodeMessage 解码消息
func DecodeMessage(kafkaMsg kafka.Message, serializer Serializer, dest interface{}) (*Message, error) {
	if err := serializer.Deserialize(kafkaMsg.Value, dest); err != nil {
		return nil, err
	}

	msg := &Message{
		Topic:     kafkaMsg.Topic,
		Key:       string(kafkaMsg.Key),
		Data:      dest,
		Timestamp: kafkaMsg.Time.UnixMilli(),
		Headers:   make(map[string]string),
	}

	for _, h := range kafkaMsg.Headers {
		msg.Headers[h.Key] = string(h.Value)
	}

	return msg, nil
}

// SchemaRegistry 模拟schema注册中心（实际项目中使用 Confluent Schema Registry）
type SchemaRegistry struct {
	schemas map[string]Schema
}

type Schema struct {
	Version int
	Type    string
	Data    []byte
}

func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{
		schemas: make(map[string]Schema),
	}
}

func (sr *SchemaRegistry) Register(subject string, schema Schema) error {
	sr.schemas[subject] = schema
	return nil
}

func (sr *SchemaRegistry) Get(subject string) (Schema, error) {
	schema, ok := sr.schemas[subject]
	if !ok {
		return Schema{}, fmt.Errorf("schema not found: %s", subject)
	}
	return schema, nil
}
