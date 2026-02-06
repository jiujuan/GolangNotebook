package config

import (
	"os"
	"strings"
)

// KafkaConfig 保存Kafka连接配置
type KafkaConfig struct {
	Brokers []string // Kafka集群地址列表
	Topic   string   // 默认Topic
	GroupID string   // 消费者组ID
}

// DefaultConfig 返回默认配置
func DefaultConfig() *KafkaConfig {
	return &KafkaConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}
}

// LoadFromEnv 从环境变量加载配置
func LoadFromEnv() *KafkaConfig {
	config := DefaultConfig()

	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		config.Brokers = strings.Split(brokers, ",")
	}

	if topic := os.Getenv("KAFKA_TOPIC"); topic != "" {
		config.Topic = topic
	}

	if groupID := os.Getenv("KAFKA_GROUP_ID"); groupID != "" {
		config.GroupID = groupID
	}

	return config
}
