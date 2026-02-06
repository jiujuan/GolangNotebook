package topic

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go-kafka/config"
	"go-kafka/utils"
)

// TopicManager Topic管理器
type TopicManager struct {
	conn   *kafka.Conn
	config *config.KafkaConfig
	logger *utils.Logger
}

// NewTopicManager 创建Topic管理器
func NewTopicManager(cfg *config.KafkaConfig) (*TopicManager, error) {
	// 连接到Kafka集群（选择一个broker即可）
	conn, err := kafka.Dial("tcp", cfg.Brokers[0])
	if err != nil {
		return nil, fmt.Errorf("连接Kafka失败: %w", err)
	}

	return &TopicManager{
		conn:   conn,
		config: cfg,
		logger: utils.NewLogger("[TopicManager]"),
	}, nil
}

// CreateTopic 创建Topic
// partitions: 分区数
// replicationFactor: 副本因子
// retentionMs: 消息保留时间（毫秒），-1表示使用默认
func (tm *TopicManager) CreateTopic(
	ctx context.Context,
	topic string,
	partitions int,
	replicationFactor int,
	retentionMs int64,
) error {
	// 配置Topic参数
	configEntries := []kafka.ConfigEntry{
		{
			ConfigName:  "cleanup.policy",
			ConfigValue: "delete",
		},
	}

	if retentionMs > 0 {
		configEntries = append(configEntries, kafka.ConfigEntry{
			ConfigName:  "retention.ms",
			ConfigValue: fmt.Sprintf("%d", retentionMs),
		})
	}

	// 创建Topic请求
	req := &kafka.CreateTopicsRequest{
		Topics: []kafka.TopicConfig{
			{
				Topic:             topic,
				NumPartitions:     partitions,
				ReplicationFactor: replicationFactor,
				ConfigEntries:     configEntries,
			},
		},
		ValidateOnly: false,
	}

	// 使用控制器连接创建Topic
	controller, err := tm.conn.Controller()
	if err != nil {
		return fmt.Errorf("获取控制器失败: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("连接控制器失败: %w", err)
	}
	defer controllerConn.Close()

	resp, err := controllerConn.CreateTopics(req)
	if err != nil {
		return fmt.Errorf("创建Topic失败: %w", err)
	}

	// 检查响应
	for _, topicErr := range resp.Errors {
		if topicErr != nil {
			return fmt.Errorf("创建Topic错误: %w", topicErr)
		}
	}

	tm.logger.Info("Topic创建成功:", topic)
	return nil
}

// DeleteTopic 删除Topic
func (tm *TopicManager) DeleteTopic(ctx context.Context, topics ...string) error {
	req := &kafka.DeleteTopicsRequest{
		Topics: topics,
	}

	controller, err := tm.conn.Controller()
	if err != nil {
		return fmt.Errorf("获取控制器失败: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("连接控制器失败: %w", err)
	}
	defer controllerConn.Close()

	resp, err := controllerConn.DeleteTopics(req)
	if err != nil {
		return fmt.Errorf("删除Topic失败: %w", err)
	}

	// 检查错误
	for i, topicErr := range resp.Errors {
		if topicErr != nil {
			tm.logger.Error("删除Topic失败", topics[i], ":", topicErr)
		} else {
			tm.logger.Info("Topic删除成功:", topics[i])
		}
	}

	return nil
}

// ListTopics 列出所有Topic
func (tm *TopicManager) ListTopics() ([]string, error) {
	partitions, err := tm.conn.ReadPartitions()
	if err != nil {
		return nil, fmt.Errorf("读取分区失败: %w", err)
	}

	// 使用map去重
	topicMap := make(map[string]struct{})
	for _, p := range partitions {
		topicMap[p.Topic] = struct{}{}
	}

	// 转换为slice
	topics := make([]string, 0, len(topicMap))
	for topic := range topicMap {
		topics = append(topics, topic)
	}

	return topics, nil
}

// TopicInfo Topic详细信息
type TopicInfo struct {
	Name         string
	Partitions   int
	Replicas     int
	IsInternal   bool
	Config       map[string]string
	PartitionIDs []int
}

// DescribeTopic 获取Topic详细信息
func (tm *TopicManager) DescribeTopic(topic string) (*TopicInfo, error) {
	partitions, err := tm.conn.ReadPartitions(topic)
	if err != nil {
		return nil, fmt.Errorf("读取Topic分区失败: %w", err)
	}

	if len(partitions) == 0 {
		return nil, fmt.Errorf("Topic不存在: %s", topic)
	}

	info := &TopicInfo{
		Name:         topic,
		Partitions:   len(partitions),
		IsInternal:   partitions[0].IsInternal,
		PartitionIDs: make([]int, len(partitions)),
		Config:       make(map[string]string),
	}

	for i, p := range partitions {
		info.PartitionIDs[i] = p.ID
		if len(p.Replicas) > 0 && info.Replicas == 0 {
			info.Replicas = len(p.Replicas)
		}
	}

	return info, nil
}

// GetTopicConfig 获取Topic配置
func (tm *TopicManager) GetTopicConfig(topic string) (map[string]string, error) {
	// 注意：kafka-go v0.4.x 可能需要直接发送请求获取配置
	// 这里简化处理，实际使用可能需要更底层的API
	return map[string]string{}, nil
}

// UpdateTopicConfig 更新Topic配置
func (tm *TopicManager) UpdateTopicConfig(topic string, configs map[string]string) error {
	entries := make([]kafka.ConfigEntry, 0, len(configs))
	for k, v := range configs {
		entries = append(entries, kafka.ConfigEntry{
			ConfigName:  k,
			ConfigValue: v,
		})
	}

	req := &kafka.AlterConfigsRequest{
		Resources: []kafka.AlterConfigRequestResource{
			{
				ResourceType: kafka.ResourceTypeTopic,
				ResourceName: topic,
				Configs:      entries,
			},
		},
	}

	controller, err := tm.conn.Controller()
	if err != nil {
		return fmt.Errorf("获取控制器失败: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("连接控制器失败: %w", err)
	}
	defer controllerConn.Close()

	_, err = controllerConn.AlterConfigs(req)
	if err != nil {
		return fmt.Errorf("更新配置失败: %w", err)
	}

	tm.logger.Info("Topic配置更新成功:", topic)
	return nil
}

// CheckTopicExists 检查Topic是否存在
func (tm *TopicManager) CheckTopicExists(topic string) (bool, error) {
	partitions, err := tm.conn.ReadPartitions(topic)
	if err != nil {
		return false, err
	}
	return len(partitions) > 0, nil
}

// CreateTopicIfNotExists 如果不存在则创建Topic
func (tm *TopicManager) CreateTopicIfNotExists(
	ctx context.Context,
	topic string,
	partitions int,
	replicationFactor int,
	retentionMs int64,
) error {
	exists, err := tm.CheckTopicExists(topic)
	if err != nil {
		return err
	}

	if exists {
		tm.logger.Info("Topic已存在，跳过创建:", topic)
		return nil
	}

	return tm.CreateTopic(ctx, topic, partitions, replicationFactor, retentionMs)
}

// GetPartitionOffsets 获取分区偏移量信息
func (tm *TopicManager) GetPartitionOffsets(topic string, partition int) (oldest, newest int64, err error) {
	conn, err := tm.conn.DialLeader(context.Background(), topic, partition)
	if err != nil {
		return 0, 0, fmt.Errorf("连接分区leader失败: %w", err)
	}
	defer conn.Close()

	// 获取最旧偏移量
	oldest, err = conn.ReadFirstOffset()
	if err != nil {
		return 0, 0, fmt.Errorf("读取最旧偏移量失败: %w", err)
	}

	// 获取最新偏移量
	newest, err = conn.ReadLastOffset()
	if err != nil {
		return 0, 0, fmt.Errorf("读取最新偏移量失败: %w", err)
	}

	return oldest, newest, nil
}

// Close 关闭连接
func (tm *TopicManager) Close() error {
	if tm.conn != nil {
		return tm.conn.Close()
	}
	return nil
}
