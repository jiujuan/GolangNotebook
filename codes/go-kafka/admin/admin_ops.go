package admin

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"go-kafka/config"
	"go-kafka/utils"
)

// AdminClient Kafka管理客户端
type AdminClient struct {
	brokers []string
	logger  *utils.Logger
}

// NewAdminClient 创建管理客户端
func NewAdminClient(cfg *config.KafkaConfig) *AdminClient {
	return &AdminClient{
		brokers: cfg.Brokers,
		logger:  utils.NewLogger("[AdminClient]"),
	}
}

// ClusterInfo 集群信息
type ClusterInfo struct {
	Brokers      []BrokerInfo
	ControllerID int32
	Topics       []string
}

type BrokerInfo struct {
	ID   int32
	Host string
	Port int
}

// GetClusterInfo 获取集群信息
func (a *AdminClient) GetClusterInfo() (*ClusterInfo, error) {
	// 连接到任意一个broker获取信息
	conn, err := kafka.Dial("tcp", a.brokers[0])
	if err != nil {
		return nil, fmt.Errorf("连接Kafka失败: %w", err)
	}
	defer conn.Close()

	info := &ClusterInfo{}

	// 获取broker列表
	brokers, err := conn.Brokers()
	if err != nil {
		return nil, fmt.Errorf("获取broker列表失败: %w", err)
	}

	info.Brokers = make([]BrokerInfo, len(brokers))
	for i, b := range brokers {
		info.Brokers[i] = BrokerInfo{
			ID:   b.ID,
			Host: b.Host,
			Port: b.Port,
		}
	}

	// 获取控制器
	controller, err := conn.Controller()
	if err != nil {
		return nil, fmt.Errorf("获取控制器失败: %w", err)
	}
	info.ControllerID = controller.ID

	// 获取Topic列表
	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, fmt.Errorf("获取分区信息失败: %w", err)
	}

	topicMap := make(map[string]struct{})
	for _, p := range partitions {
		topicMap[p.Topic] = struct{}{}
	}

	info.Topics = make([]string, 0, len(topicMap))
	for topic := range topicMap {
		info.Topics = append(info.Topics, topic)
	}

	return info, nil
}

// ConsumerGroupInfo 消费者组信息
type ConsumerGroupInfo struct {
	GroupID         string
	State           string
	Protocol        string
	ProtocolType    string
	Members         []MemberInfo
	Coordinator     string
	TopicPartitions map[string][]PartitionAssignment
}

type MemberInfo struct {
	ID       string
	ClientID string
	Host     string
}

type PartitionAssignment struct {
	Partition int
	Offset    int64
	Lag       int64
}

// ListConsumerGroups 列出所有消费者组
func (a *AdminClient) ListConsumerGroups() ([]string, error) {
	conn, err := kafka.Dial("tcp", a.brokers[0])
	if err != nil {
		return nil, fmt.Errorf("连接Kafka失败: %w", err)
	}
	defer conn.Close()

	groups, err := conn.ReadGroups()
	if err != nil {
		return nil, fmt.Errorf("读取消费者组失败: %w", err)
	}

	groupIDs := make([]string, len(groups.Groups))
	for i, g := range groups.Groups {
		groupIDs[i] = g.GroupID
	}

	return groupIDs, nil
}

// DescribeConsumerGroup 获取消费者组详情
func (a *AdminClient) DescribeConsumerGroup(groupID string) (*ConsumerGroupInfo, error) {
	// 连接到消费者组协调器
	conn, err := kafka.Dial("tcp", a.brokers[0])
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 查找协调器
	req := &kafka.FindCoordinatorRequest{
		Addr:    conn.RemoteAddr(),
		Key:     groupID,
		KeyType: kafka.CoordinatorKeyTypeGroup,
	}

	resp, err := conn.FindCoordinator(req)
	if err != nil {
		return nil, fmt.Errorf("查找协调器失败: %w", err)
	}

	// 连接协调器获取详细信息
	coordinatorAddr := fmt.Sprintf("%s:%d", resp.Host, resp.Port)
	coordConn, err := kafka.Dial("tcp", coordinatorAddr)
	if err != nil {
		return nil, fmt.Errorf("连接协调器失败: %w", err)
	}
	defer coordConn.Close()

	// 获取消费者组描述（简化版）
	info := &ConsumerGroupInfo{
		GroupID:     groupID,
		Coordinator: coordinatorAddr,
	}

	return info, nil
}

// DeleteConsumerGroup 删除消费者组
func (a *AdminClient) DeleteConsumerGroup(groupID string) error {
	conn, err := kafka.Dial("tcp", a.brokers[0])
	if err != nil {
		return fmt.Errorf("连接Kafka失败: %w", err)
	}
	defer conn.Close()

	req := &kafka.DeleteGroupsRequest{
		Groups: []string{groupID},
	}

	resp, err := conn.DeleteGroups(req)
	if err != nil {
		return fmt.Errorf("删除消费者组失败: %w", err)
	}

	for _, err := range resp.Errors {
		if err != nil {
			return fmt.Errorf("删除消费者组错误: %w", err)
		}
	}

	a.logger.Info("消费者组删除成功:", groupID)
	return nil
}

// PartitionInfo 分区信息
type PartitionInfo struct {
	ID       int
	Leader   string
	Replicas []string
	Isr      []string
	Oldest   int64
	Newest   int64
	Lag      int64
}

// GetPartitionDetails 获取分区详细信息
func (a *AdminClient) GetPartitionDetails(topic string) ([]PartitionInfo, error) {
	conn, err := kafka.Dial("tcp", a.brokers[0])
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions(topic)
	if err != nil {
		return nil, fmt.Errorf("读取分区失败: %w", err)
	}

	infos := make([]PartitionInfo, len(partitions))
	for i, p := range partitions {
		infos[i] = PartitionInfo{
			ID:       p.ID,
			Leader:   fmt.Sprintf("%s:%d", p.Leader.Host, p.Leader.Port),
			Replicas: make([]string, len(p.Replicas)),
			Isr:      make([]string, len(p.Isr)),
		}

		for j, r := range p.Replicas {
			infos[i].Replicas[j] = fmt.Sprintf("%s:%d", r.Host, r.Port)
		}

		for j, isr := range p.Isr {
			infos[i].Isr[j] = fmt.Sprintf("%s:%d", isr.Host, isr.Port)
		}

		// 获取偏移量信息
		leaderConn, err := conn.DialLeader(context.Background(), topic, p.ID)
		if err == nil {
			infos[i].Oldest, _ = leaderConn.ReadFirstOffset()
			infos[i].Newest, _ = leaderConn.ReadLastOffset()
			infos[i].Lag = infos[i].Newest - infos[i].Oldest
			leaderConn.Close()
		}
	}

	return infos, nil
}

// Metrics Kafka指标
type Metrics struct {
	Topics         int
	Partitions     int
	ConsumerGroups int
}

// GetMetrics 获取集群指标
func (a *AdminClient) GetMetrics() (*Metrics, error) {
	conn, err := kafka.Dial("tcp", a.brokers[0])
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	m := &Metrics{}

	// Topic和分区数
	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, err
	}

	topicMap := make(map[string]struct{})
	for _, p := range partitions {
		topicMap[p.Topic] = struct{}{}
		m.Partitions++
	}
	m.Topics = len(topicMap)

	// 消费者组数
	groups, err := conn.ReadGroups()
	if err != nil {
		return nil, err
	}
	m.ConsumerGroups = len(groups.Groups)

	return m, nil
}

// ResetConsumerGroupOffset 重置消费者组偏移量
func (a *AdminClient) ResetConsumerGroupOffset(
	groupID string,
	topic string,
	partition int,
	offset int64,
) error {
	// 创建临时消费者来提交偏移量
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   a.brokers,
		Topic:     topic,
		GroupID:   groupID,
		Partition: partition,
	})
	defer reader.Close()

	// 设置偏移量
	if err := reader.SetOffset(offset); err != nil {
		return fmt.Errorf("设置偏移量失败: %w", err)
	}

	a.logger.Info("偏移量重置成功, group:", groupID, "topic:", topic,
		"partition:", partition, "offset:", offset)
	return nil
}
